package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"pulsepoll-backend/models"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	Client     *redis.Client
	isMemory   bool
	memVotes   sync.Map // string (pollID) -> map[int]int64
	memVoters  sync.Map // string (pollID) -> map[string]bool
	memViewers sync.Map // string (pollID) -> int64
	memSubs    sync.Map // string (pollID) -> []chan string
	memMu      sync.RWMutex
}

func InitRedis(uri string) (*RedisService, error) {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		opt = &redis.Options{
			Addr: uri,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	log.Println("✔ Successfully connected to Redis (Pub/Sub & Atomic Engine Active)")
	return &RedisService{Client: client}, nil
}

func NewMemoryRedisService() *RedisService {
	log.Println("⚡ Running in-memory Pub/Sub & Atomic Engine fallback")
	return &RedisService{
		isMemory: true,
	}
}

func (r *RedisService) IsRealRedis() bool {
	return r.Client != nil && !r.isMemory
}

func (r *RedisService) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
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
	if r.Client != nil {
		return r.Client.SIsMember(ctx, r.votersKey(pollID), voterID).Result()
	}

	r.memMu.RLock()
	defer r.memMu.RUnlock()
	val, ok := r.memVoters.Load(pollID)
	if !ok {
		return false, nil
	}
	voters := val.(map[string]bool)
	return voters[voterID], nil
}

// RecordVoter adds the voter to the deduplication set
func (r *RedisService) RecordVoter(ctx context.Context, pollID, voterID string) error {
	if r.Client != nil {
		return r.Client.SAdd(ctx, r.votersKey(pollID), voterID).Err()
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	val, ok := r.memVoters.Load(pollID)
	var voters map[string]bool
	if !ok {
		voters = make(map[string]bool)
	} else {
		voters = val.(map[string]bool)
	}
	voters[voterID] = true
	r.memVoters.Store(pollID, voters)
	return nil
}

// IncrementVote atomically increments the vote count for an option
func (r *RedisService) IncrementVote(ctx context.Context, pollID string, optionIndex int) (int64, error) {
	if r.Client != nil {
		return r.Client.HIncrBy(ctx, r.votesKey(pollID), strconv.Itoa(optionIndex), 1).Result()
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	val, ok := r.memVotes.Load(pollID)
	var votes map[int]int64
	if !ok {
		votes = make(map[int]int64)
	} else {
		votes = val.(map[int]int64)
	}
	votes[optionIndex]++
	r.memVotes.Store(pollID, votes)
	return votes[optionIndex], nil
}

// InitializePollVotes sets initial count of 0 for all options in Redis
func (r *RedisService) InitializePollVotes(ctx context.Context, pollID string, optionCount int) error {
	if r.Client != nil {
		pipe := r.Client.Pipeline()
		votesKey := r.votesKey(pollID)
		for i := 0; i < optionCount; i++ {
			pipe.HSetNX(ctx, votesKey, strconv.Itoa(i), 0)
		}
		pipe.Expire(ctx, votesKey, 30*24*time.Hour)
		pipe.Expire(ctx, r.votersKey(pollID), 30*24*time.Hour)
		_, err := pipe.Exec(ctx)
		return err
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	votes := make(map[int]int64)
	for i := 0; i < optionCount; i++ {
		votes[i] = 0
	}
	r.memVotes.Store(pollID, votes)
	return nil
}

// GetVotes retrieves all option counts and calculates totals
func (r *RedisService) GetVotes(ctx context.Context, pollID string, optionCount int) (map[int]int64, int64, error) {
	if r.Client != nil {
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

	r.memMu.RLock()
	defer r.memMu.RUnlock()
	votes := make(map[int]int64)
	var total int64 = 0
	val, ok := r.memVotes.Load(pollID)
	if ok {
		storedVotes := val.(map[int]int64)
		for i := 0; i < optionCount; i++ {
			c := storedVotes[i]
			votes[i] = c
			total += c
		}
	} else {
		for i := 0; i < optionCount; i++ {
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

	if r.Client != nil {
		return r.Client.Publish(ctx, r.PubSubChannel(pollID), data).Err()
	}

	// In-memory Pub/Sub distribution
	r.memMu.RLock()
	defer r.memMu.RUnlock()
	val, ok := r.memSubs.Load(pollID)
	if ok {
		subs := val.([]chan string)
		strPayload := string(data)
		for _, ch := range subs {
			select {
			case ch <- strPayload:
			default:
				// non-blocking if subscriber buffer is full
			}
		}
	}
	return nil
}

// Subscribe listens for live poll events via Redis Pub/Sub (or memory channel)
func (r *RedisService) Subscribe(ctx context.Context, pollID string) (<-chan string, func(), error) {
	if r.Client != nil {
		pubsub := r.Client.Subscribe(ctx, r.PubSubChannel(pollID))
		outCh := make(chan string, 100)
		done := make(chan struct{})

		go func() {
			defer close(outCh)
			rch := pubsub.Channel()
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case msg, ok := <-rch:
					if !ok {
						return
					}
					select {
					case outCh <- msg.Payload:
					case <-done:
						return
					}
				}
			}
		}()

		unsub := func() {
			close(done)
			_ = pubsub.Close()
		}
		return outCh, unsub, nil
	}

	// In-memory subscribe
	ch := make(chan string, 100)
	r.memMu.Lock()
	val, _ := r.memSubs.Load(pollID)
	var subs []chan string
	if val != nil {
		subs = val.([]chan string)
	}
	subs = append(subs, ch)
	r.memSubs.Store(pollID, subs)
	r.memMu.Unlock()

	unsub := func() {
		r.memMu.Lock()
		defer r.memMu.Unlock()
		val, _ := r.memSubs.Load(pollID)
		if val != nil {
			var newSubs []chan string
			for _, s := range val.([]chan string) {
				if s != ch {
					newSubs = append(newSubs, s)
				}
			}
			r.memSubs.Store(pollID, newSubs)
		}
	}

	return ch, unsub, nil
}

// Viewer tracking
func (r *RedisService) IncrementViewers(ctx context.Context, pollID string) (int64, error) {
	if r.Client != nil {
		return r.Client.Incr(ctx, r.viewersKey(pollID)).Result()
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	val, _ := r.memViewers.Load(pollID)
	var cur int64 = 0
	if val != nil {
		cur = val.(int64)
	}
	cur++
	r.memViewers.Store(pollID, cur)
	return cur, nil
}

func (r *RedisService) DecrementViewers(ctx context.Context, pollID string) (int64, error) {
	if r.Client != nil {
		count, err := r.Client.Decr(ctx, r.viewersKey(pollID)).Result()
		if count < 0 {
			r.Client.Set(ctx, r.viewersKey(pollID), 0, 0)
			return 0, nil
		}
		return count, err
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	val, _ := r.memViewers.Load(pollID)
	var cur int64 = 0
	if val != nil {
		cur = val.(int64)
	}
	cur--
	if cur < 0 {
		cur = 0
	}
	r.memViewers.Store(pollID, cur)
	return cur, nil
}

func (r *RedisService) GetViewers(ctx context.Context, pollID string) (int64, error) {
	if r.Client != nil {
		val, err := r.Client.Get(ctx, r.viewersKey(pollID)).Result()
		if err == redis.Nil {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		return strconv.ParseInt(val, 10, 64)
	}

	r.memMu.RLock()
	defer r.memMu.RUnlock()
	val, ok := r.memViewers.Load(pollID)
	if !ok {
		return 0, nil
	}
	return val.(int64), nil
}
