package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"pulsepoll-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DataStore interface {
	// Users
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)

	// Polls
	CreatePoll(ctx context.Context, poll *models.Poll) error
	GetPollByID(ctx context.Context, id string) (*models.Poll, error)
	GetPollsByCreator(ctx context.Context, creatorID string) ([]models.Poll, error)
	ListActivePolls(ctx context.Context, limit int) ([]models.Poll, error)
	UpdatePollStatus(ctx context.Context, id string, creatorID string, isActive bool) error
	IncrementPollVotes(ctx context.Context, id string, optionIndex int) error
	DeletePoll(ctx context.Context, id string, creatorID string) error

	// Votes Audit
	RecordVote(ctx context.Context, vote *models.VoteRecord) error
	GetVotesByPoll(ctx context.Context, pollID string) ([]models.VoteRecord, error)

	// Health
	IsConnected() bool
	Close(ctx context.Context) error
}

// ----------------------------------------------------
// MongoDB Implementation
// ----------------------------------------------------

type MongoStore struct {
	Client   *mongo.Client
	Database *mongo.Database
	Users    *mongo.Collection
	Polls    *mongo.Collection
	Votes    *mongo.Collection
}

func NewMongoStore(uri, dbName string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	usersCol := db.Collection("users")
	pollsCol := db.Collection("polls")
	votesCol := db.Collection("votes")

	_, _ = usersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	_, _ = pollsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "creator_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})

	return &MongoStore{
		Client:   client,
		Database: db,
		Users:    usersCol,
		Polls:    pollsCol,
		Votes:    votesCol,
	}, nil
}

func (m *MongoStore) CreateUser(ctx context.Context, user *models.User) error {
	_, err := m.Users.InsertOne(ctx, user)
	return err
}

func (m *MongoStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := m.Users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *MongoStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := m.Users.FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *MongoStore) EmailExists(ctx context.Context, email string) (bool, error) {
	count, err := m.Users.CountDocuments(ctx, bson.M{"email": email})
	return count > 0, err
}

func (m *MongoStore) CreatePoll(ctx context.Context, poll *models.Poll) error {
	_, err := m.Polls.InsertOne(ctx, poll)
	return err
}

func (m *MongoStore) GetPollByID(ctx context.Context, id string) (*models.Poll, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var poll models.Poll
	if err := m.Polls.FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
		return nil, err
	}
	return &poll, nil
}

func (m *MongoStore) GetPollsByCreator(ctx context.Context, creatorID string) ([]models.Poll, error) {
	cID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := m.Polls.Find(ctx, bson.M{"creator_id": cID}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

func (m *MongoStore) ListActivePolls(ctx context.Context, limit int) ([]models.Poll, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}
	cursor, err := m.Polls.Find(ctx, bson.M{"is_active": true}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

func (m *MongoStore) UpdatePollStatus(ctx context.Context, id string, creatorID string, isActive bool) error {
	pID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	cID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return err
	}

	res, err := m.Polls.UpdateOne(ctx, bson.M{"_id": pID, "creator_id": cID}, bson.M{"$set": bson.M{"is_active": isActive}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("poll not found or unauthorized")
	}
	return nil
}

func (m *MongoStore) IncrementPollVotes(ctx context.Context, id string, optionIndex int) error {
	pID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	update := bson.M{
		"$inc": bson.M{
			fmt.Sprintf("options.%d.votes", optionIndex): 1,
			"total_votes":                                1,
		},
	}
	_, err = m.Polls.UpdateOne(ctx, bson.M{"_id": pID}, update)
	return err
}

func (m *MongoStore) DeletePoll(ctx context.Context, id string, creatorID string) error {
	pID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	cID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return err
	}
	res, err := m.Polls.DeleteOne(ctx, bson.M{"_id": pID, "creator_id": cID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("poll not found or unauthorized")
	}
	_, _ = m.Votes.DeleteMany(ctx, bson.M{"poll_id": pID})
	return nil
}

func (m *MongoStore) RecordVote(ctx context.Context, vote *models.VoteRecord) error {
	_, err := m.Votes.InsertOne(ctx, vote)
	return err
}

func (m *MongoStore) GetVotesByPoll(ctx context.Context, pollID string) ([]models.VoteRecord, error) {
	pID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return nil, err
	}
	cursor, err := m.Votes.Find(ctx, bson.M{"poll_id": pID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var votes []models.VoteRecord
	_ = cursor.All(ctx, &votes)
	return votes, nil
}

func (m *MongoStore) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return m.Client.Ping(ctx, nil) == nil
}

func (m *MongoStore) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

// ----------------------------------------------------
// In-Memory Fallback Implementation (when MongoDB isn't running)
// ----------------------------------------------------

type MemoryStore struct {
	mu    sync.RWMutex
	users map[string]*models.User // keyed by email
	polls map[string]*models.Poll // keyed by hex ID
	votes map[string][]models.VoteRecord
}

func NewMemoryStore() *MemoryStore {
	log.Println("ℹ️  Running with Thread-Safe In-Memory Store (MongoDB local fallback)")
	return &MemoryStore{
		users: make(map[string]*models.User),
		polls: make(map[string]*models.Poll),
		votes: make(map[string][]models.VoteRecord),
	}
}

func (s *MemoryStore) CreateUser(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.Email] = user
	return nil
}

func (s *MemoryStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *MemoryStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.ID.Hex() == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (s *MemoryStore) EmailExists(ctx context.Context, email string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.users[email]
	return exists, nil
}

func (s *MemoryStore) CreatePoll(ctx context.Context, poll *models.Poll) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.polls[poll.ID.Hex()] = poll
	return nil
}

func (s *MemoryStore) GetPollByID(ctx context.Context, id string) (*models.Poll, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	poll, exists := s.polls[id]
	if !exists {
		return nil, errors.New("poll not found")
	}
	return poll, nil
}

func (s *MemoryStore) GetPollsByCreator(ctx context.Context, creatorID string) ([]models.Poll, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []models.Poll
	for _, p := range s.polls {
		if p.CreatorID.Hex() == creatorID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (s *MemoryStore) ListActivePolls(ctx context.Context, limit int) ([]models.Poll, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []models.Poll
	for _, p := range s.polls {
		if p.IsActive {
			list = append(list, *p)
		}
	}
	// Sort newest first
	for i := 0; i < len(list)-1; i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i].CreatedAt.Before(list[j].CreatedAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (s *MemoryStore) UpdatePollStatus(ctx context.Context, id string, creatorID string, isActive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	poll, exists := s.polls[id]
	if !exists || poll.CreatorID.Hex() != creatorID {
		return errors.New("poll not found or unauthorized")
	}
	poll.IsActive = isActive
	return nil
}

func (s *MemoryStore) IncrementPollVotes(ctx context.Context, id string, optionIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	poll, exists := s.polls[id]
	if !exists {
		return errors.New("poll not found")
	}
	if optionIndex >= 0 && optionIndex < len(poll.Options) {
		poll.Options[optionIndex].Votes++
		poll.TotalVotes++
	}
	return nil
}

func (s *MemoryStore) DeletePoll(ctx context.Context, id string, creatorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	poll, exists := s.polls[id]
	if !exists || poll.CreatorID.Hex() != creatorID {
		return errors.New("poll not found or unauthorized")
	}
	delete(s.polls, id)
	delete(s.votes, id)
	return nil
}

func (s *MemoryStore) RecordVote(ctx context.Context, vote *models.VoteRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idHex := vote.PollID.Hex()
	s.votes[idHex] = append(s.votes[idHex], *vote)
	return nil
}

func (s *MemoryStore) GetVotesByPoll(ctx context.Context, pollID string) ([]models.VoteRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.votes[pollID], nil
}

func (s *MemoryStore) IsConnected() bool {
	return true
}

func (s *MemoryStore) Close(ctx context.Context) error {
	return nil
}
