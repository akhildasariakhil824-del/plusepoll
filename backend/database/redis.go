package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"pulsepoll-backend/models"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	Client *redis.Client
}

func InitRedis(uri string) (*RedisService, error) {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		// Fallback to basic options if not a full URL
		opt = &redis.Options{
			Addr: uri,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println(" Successfully connected to Redis")
	return &RedisService{Client: client}, nil
}

// Key helpers
func (r *RedisService) votesKey(pollID string) string {
	return fmt.Sprintf("poll:%s:votes", pollID)
}

func (r *RedisService) votersKey(pollID string) string {
	return fmt.Sprintf("poll:%s:voters", pollID)
}

func (r *RedisService) viewersKey(pollID string) string {
	return fmt.Sprintf("poll:%s:viewers", pollID)
}

func (r *RedisService) PubSubChannel(pollID string) string {
	return fmt.Sprintf("poll:%s:updates", pollID)
}

// HasVoted checks if the voter has already voted in this poll
func (r *RedisService) HasVoted(ctx context.Context, pollID, voterID string) (bool, error) {
	return r.Client.SIsMember(ctx, r.votersKey(pollID), voterID).Result()
}

// RecordVoter adds the voter to the deduplication set
func (r *RedisService) RecordVoter(ctx context.Context, pollID, voterID string) error {
	return r.Client.SAdd(ctx, r.votersKey(pollID), voterID).Err()
}

// IncrementVote atomically increments the vote count for an option
func (r *RedisService) IncrementVote(ctx context.Context, pollID string, optionIndex int) (int64, error) {
	return r.Client.HIncrBy(ctx, r.votesKey(pollID), strconv.Itoa(optionIndex), 1).Result()
}

// InitializePollVotes sets initial count of 0 for all options in Redis
func (r *RedisService) InitializePollVotes(ctx context.Context, pollID string, optionCount int) error {
	pipe := r.Client.Pipeline()
	votesKey := r.votesKey(pollID)
	for i := 0; i < optionCount; i++ {
		pipe.HSetNX(ctx, votesKey, strconv.Itoa(i), 0)
	}
	// Expire keys after 30 days of inactivity
	pipe.Expire(ctx, votesKey, 30*24*time.Hour)
	pipe.Expire(ctx, r.votersKey(pollID), 30*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// GetVotes retrieves all option counts and calculates totals
func (r *RedisService) GetVotes(ctx context.Context, pollID string, optionCount int) (map[int]int64, int64, error) {
	result, err := r.Client.HGetAll(ctx, r.votesKey(pollID)).Result()
	if err != nil {
		return nil, 0, err
	}

	votes := make(map[int]int64)
	var total int64 = 0

	for i := 0; i < optionCount; i++ {
		keyStr := strconv.Itoa(i)
		if countStr, exists := result[keyStr]; exists {
			c, _ := strconv.ParseInt(countStr, 10, 64)
			votes[i] = c
			total += c
		} else {
			votes[i] = 0
		}
	}

	return votes, total, nil
}

// PublishUpdate sends real-time message through Redis Pub/Sub
func (r *RedisService) PublishUpdate(ctx context.Context, pollID string, msg *models.WebSocketMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Client.Publish(ctx, r.PubSubChannel(pollID), data).Err()
}

// Viewer tracking
func (r *RedisService) IncrementViewers(ctx context.Context, pollID string) (int64, error) {
	return r.Client.Incr(ctx, r.viewersKey(pollID)).Result()
}

func (r *RedisService) DecrementViewers(ctx context.Context, pollID string) (int64, error) {
	count, err := r.Client.Decr(ctx, r.viewersKey(pollID)).Result()
	if count < 0 {
		r.Client.Set(ctx, r.viewersKey(pollID), 0, 0)
		return 0, nil
	}
	return count, err
}

func (r *RedisService) GetViewers(ctx context.Context, pollID string) (int64, error) {
	val, err := r.Client.Get(ctx, r.viewersKey(pollID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}
