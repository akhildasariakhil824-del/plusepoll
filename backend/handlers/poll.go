package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pulsepoll-backend/database"
	"pulsepoll-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PollHandler struct {
	store database.DataStore
	redis *database.RedisService
}

func NewPollHandler(store database.DataStore, redis *database.RedisService) *PollHandler {
	return &PollHandler{
		store: store,
		redis: redis,
	}
}

// CreatePoll handles creating a new poll
func (h *PollHandler) CreatePoll(c *gin.Context) {
	userIDHex, _ := c.Get("userID")
	userName, _ := c.Get("userName")

	creatorID, err := primitive.ObjectIDFromHex(userIDHex.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid creator ID"})
		return
	}

	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}

	// Validate title & options
	title := strings.TrimSpace(req.Title)
	if len(title) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll title must be at least 3 characters"})
		return
	}

	var validOptions []models.PollOption
	for idx, optText := range req.Options {
		trimmed := strings.TrimSpace(optText)
		if trimmed == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Option %d cannot be blank", idx+1)})
			return
		}
		validOptions = append(validOptions, models.PollOption{
			Index: idx,
			Text:  trimmed,
			Votes: 0,
		})
	}

	if len(validOptions) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A poll requires at least 2 non-empty options"})
		return
	}

	pollID := primitive.NewObjectID()
	poll := models.Poll{
		ID:          pollID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Options:     validOptions,
		CreatorID:   creatorID,
		CreatorName: userName.(string),
		IsActive:    true,
		TotalVotes:  0,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. Save to Database Store (Mongo or MemoryStore fallback)
	err = h.store.CreatePoll(ctx, &poll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist poll in database"})
		return
	}

	// 2. Initialize real-time vote counters in Redis
	if err := h.redis.InitializePollVotes(ctx, pollID.Hex(), len(validOptions)); err != nil {
		log.Printf("Warning: Redis vote initialization failed for poll %s: %v", pollID.Hex(), err)
	}

	c.JSON(http.StatusCreated, poll)
}

// ListMyPolls retrieves all polls created by the authenticated user
func (h *PollHandler) ListMyPolls(c *gin.Context) {
	userIDHex, _ := c.Get("userID")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	polls, err := h.store.GetPollsByCreator(ctx, userIDHex.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query polls"})
		return
	}

	// Enrich each poll with live Redis vote numbers and active viewer counts
	type EnrichedPoll struct {
		models.Poll
		LiveViewers int64 `json:"live_viewers"`
	}

	enriched := make([]EnrichedPoll, 0, len(polls))
	for _, p := range polls {
		pollHex := p.ID.Hex()
		votesMap, total, err := h.redis.GetVotes(ctx, pollHex, len(p.Options))
		if err == nil && total > 0 {
			p.TotalVotes = total
			for i := range p.Options {
				p.Options[i].Votes = votesMap[p.Options[i].Index]
			}
		}
		viewers, _ := h.redis.GetViewers(ctx, pollHex)
		enriched = append(enriched, EnrichedPoll{
			Poll:        p,
			LiveViewers: viewers,
		})
	}

	c.JSON(http.StatusOK, enriched)
}

// GetPoll retrieves single poll with real-time tally merged from Redis
func (h *PollHandler) GetPoll(c *gin.Context) {
	idHex := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	poll, err := h.store.GetPollByID(ctx, idHex)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// Fetch live counts from Redis
	votesMap, totalVotes, err := h.redis.GetVotes(ctx, idHex, len(poll.Options))
	if err == nil {
		poll.TotalVotes = totalVotes
		for i := range poll.Options {
			poll.Options[i].Votes = votesMap[poll.Options[i].Index]
		}
	}

	// Check if this voter has already voted
	voterID := c.Query("voter_id")
	hasVoted := false
	if voterID != "" {
		hasVoted, _ = h.redis.HasVoted(ctx, idHex, voterID)
	}

	viewers, _ := h.redis.GetViewers(ctx, idHex)

	c.JSON(http.StatusOK, gin.H{
		"poll":         poll,
		"has_voted":    hasVoted,
		"live_viewers": viewers,
	})
}

// SubmitVote handles casting a vote with atomic Redis counter & deduplication
func (h *PollHandler) SubmitVote(c *gin.Context) {
	idHex := c.Param("id")

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Option index and voter ID are required"})
		return
	}

	req.VoterID = strings.TrimSpace(req.VoterID)
	if req.VoterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voter_id identifier is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. Fetch Poll from store to check active status & option boundaries
	poll, err := h.store.GetPollByID(ctx, idHex)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	if !poll.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This poll is closed and no longer accepting votes"})
		return
	}

	if req.OptionIndex < 0 || req.OptionIndex >= len(poll.Options) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid option index: must be between 0 and %d", len(poll.Options)-1),
		})
		return
	}

	// 2. Redis duplicate check
	alreadyVoted, err := h.redis.HasVoted(ctx, idHex, req.VoterID)
	if err == nil && alreadyVoted {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already voted in this poll"})
		return
	}

	// 3. Mark voter in Redis set
	if err := h.redis.RecordVoter(ctx, idHex, req.VoterID); err != nil {
		log.Printf("Warning: Failed to record voter %s in Redis: %v", req.VoterID, err)
	}

	// 4. Atomically increment vote counter in Redis
	_, err = h.redis.IncrementVote(ctx, idHex, req.OptionIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote in real-time engine"})
		return
	}

	// 5. Read current snapshot from Redis for immediate broadcast
	votesMap, totalVotes, err := h.redis.GetVotes(ctx, idHex, len(poll.Options))
	if err != nil {
		totalVotes = 1
		votesMap = map[int]int64{req.OptionIndex: 1}
	}

	percentages := make(map[int]float64)
	for idx, v := range votesMap {
		if totalVotes > 0 {
			percentages[idx] = float64(v) / float64(totalVotes) * 100.0
		} else {
			percentages[idx] = 0
		}
	}

	viewers, _ := h.redis.GetViewers(ctx, idHex)

	// 6. Redis Pub/Sub: broadcast to all connected WebSocket clients across instances
	wsMsg := &models.WebSocketMessage{
		Type:        "vote_update",
		PollID:      idHex,
		TotalVotes:  totalVotes,
		OptionVotes: votesMap,
		Percentages: percentages,
		ViewerCount: viewers,
		IsActive:    poll.IsActive,
		Timestamp:   time.Now(),
	}
	if err := h.redis.PublishUpdate(ctx, idHex, wsMsg); err != nil {
		log.Printf("Warning: Redis publish failed for poll %s: %v", idHex, err)
	}

	// 7. Persist to DataStore (Async background write for zero-latency voter response)
	go func(pIDHex string, pID primitive.ObjectID, optIdx int, vID, clientIP, userAgent string) {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer bgCancel()

		voteRecord := models.VoteRecord{
			ID:          primitive.NewObjectID(),
			PollID:      pID,
			OptionIndex: optIdx,
			VoterID:     vID,
			IP:          clientIP,
			UserAgent:   userAgent,
			VotedAt:     time.Now(),
		}
		_ = h.store.RecordVote(bgCtx, &voteRecord)
		_ = h.store.IncrementPollVotes(bgCtx, pIDHex, optIdx)
	}(idHex, poll.ID, req.OptionIndex, req.VoterID, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message":     "Vote recorded successfully",
		"total_votes": totalVotes,
		"options":     votesMap,
		"percentages": percentages,
	})
}

// TogglePollStatus opens or closes a poll (creator only)
func (h *PollHandler) TogglePollStatus(c *gin.Context) {
	idHex := c.Param("id")
	userIDHex, _ := c.Get("userID")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	poll, err := h.store.GetPollByID(ctx, idHex)
	if err != nil || poll.CreatorID.Hex() != userIDHex.(string) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or unauthorized"})
		return
	}

	newStatus := !poll.IsActive
	err = h.store.UpdatePollStatus(ctx, idHex, userIDHex.(string), newStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update poll status"})
		return
	}

	votesMap, totalVotes, _ := h.redis.GetVotes(ctx, idHex, len(poll.Options))
	viewers, _ := h.redis.GetViewers(ctx, idHex)

	wsMsg := &models.WebSocketMessage{
		Type:        "status_change",
		PollID:      idHex,
		TotalVotes:  totalVotes,
		OptionVotes: votesMap,
		ViewerCount: viewers,
		IsActive:    newStatus,
		Message:     fmt.Sprintf("Poll is now %s", map[bool]string{true: "open", false: "closed"}[newStatus]),
		Timestamp:   time.Now(),
	}
	_ = h.redis.PublishUpdate(ctx, idHex, wsMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Poll status updated",
		"is_active": newStatus,
	})
}

// DeletePoll removes poll (creator only)
func (h *PollHandler) DeletePoll(c *gin.Context) {
	idHex := c.Param("id")
	userIDHex, _ := c.Get("userID")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.store.DeletePoll(ctx, idHex, userIDHex.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll successfully deleted"})
}

// ExportPollResults exports votes history as CSV for analytics
func (h *PollHandler) ExportPollResults(c *gin.Context) {
	idHex := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	poll, err := h.store.GetPollByID(ctx, idHex)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	votes, _ := h.store.GetVotesByPoll(ctx, idHex)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=poll-%s-results.csv", idHex))
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"Vote ID", "Option Index", "Option Text", "Voter Identifier", "Voted At (UTC)"})

	optionMap := make(map[int]string)
	for _, opt := range poll.Options {
		optionMap[opt.Index] = opt.Text
	}

	for _, v := range votes {
		_ = writer.Write([]string{
			v.ID.Hex(),
			strconv.Itoa(v.OptionIndex),
			optionMap[v.OptionIndex],
			v.VoterID,
			v.VotedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}
