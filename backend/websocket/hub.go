package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"pulsepoll-backend/database"
	"pulsepoll-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin WebSocket connections
	},
}

// HandleWebSocket upgrades HTTP to WS and streams live poll updates from Redis Pub/Sub
func HandleWebSocket(c *gin.Context, mongo *database.MongoDB, redis *database.RedisService) {
	pollIDHex := c.Param("id")
	pollID, err := primitive.ObjectIDFromHex(pollIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID"})
		return
	}

	// 1. Verify poll exists in MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	if err := mongo.Polls.FindOne(ctx, bson.M{"_id": pollID}).Decode(&poll); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// 2. Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 3. Increment viewer counter in Redis
	viewers, _ := redis.IncrementViewers(context.Background(), pollIDHex)

	// Notify all connected clients of viewer count increase
	_ = redis.PublishUpdate(context.Background(), pollIDHex, &models.WebSocketMessage{
		Type:        "viewer_count",
		PollID:      pollIDHex,
		ViewerCount: viewers,
		IsActive:    poll.IsActive,
		Timestamp:   time.Now(),
	})

	// Ensure viewer count is decremented when client disconnects
	defer func() {
		newViewers, _ := redis.DecrementViewers(context.Background(), pollIDHex)
		_ = redis.PublishUpdate(context.Background(), pollIDHex, &models.WebSocketMessage{
			Type:        "viewer_count",
			PollID:      pollIDHex,
			ViewerCount: newViewers,
			IsActive:    poll.IsActive,
			Timestamp:   time.Now(),
		})
	}()

	// 4. Send initial snapshot immediately to the connected client
	votesMap, totalVotes, _ := redis.GetVotes(context.Background(), pollIDHex, len(poll.Options))
	percentages := make(map[int]float64)
	for idx, v := range votesMap {
		if totalVotes > 0 {
			percentages[idx] = float64(v) / float64(totalVotes) * 100.0
		} else {
			percentages[idx] = 0
		}
	}

	initialMsg := models.WebSocketMessage{
		Type:        "initial_state",
		PollID:      pollIDHex,
		TotalVotes:  totalVotes,
		OptionVotes: votesMap,
		Percentages: percentages,
		ViewerCount: viewers,
		IsActive:    poll.IsActive,
		Timestamp:   time.Now(),
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(initialMsg); err != nil {
		log.Printf("Error sending initial WS message: %v", err)
		return
	}

	// 5. Subscribe to Redis Pub/Sub channel for this poll
	pubsub := redis.Client.Subscribe(context.Background(), redis.PubSubChannel(pollIDHex))
	defer pubsub.Close()

	ch := pubsub.Channel()

	// Goroutine to read client messages (pings/heartbeats) and detect disconnects
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	// 6. Listen for Redis Pub/Sub events and forward them to WebSocket client
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Send ping heartbeat to keep connection alive
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}

			var payload models.WebSocketMessage
			if err := json.Unmarshal([]byte(redisMsg.Payload), &payload); err != nil {
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(payload); err != nil {
				log.Printf("Error writing WS payload: %v", err)
				return
			}
		}
	}
}
