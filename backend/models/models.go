package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered user in MongoDB
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// RegisterRequest payload for user signup
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=60"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// LoginRequest payload for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse payload returned on login/register
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// PollOption represents an individual choice in a poll
type PollOption struct {
	Index int    `bson:"index" json:"index"`
	Text  string `bson:"text" json:"text"`
	Votes int64  `bson:"votes" json:"votes"`
}

// Poll represents a poll entity in MongoDB
type Poll struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Options     []PollOption       `bson:"options" json:"options"`
	CreatorID   primitive.ObjectID `bson:"creator_id" json:"creator_id"`
	CreatorName string             `bson:"creator_name" json:"creator_name"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	TotalVotes  int64              `bson:"total_votes" json:"total_votes"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
}

// CreatePollRequest payload for creating a new poll
type CreatePollRequest struct {
	Title       string   `json:"title" binding:"required,min=3,max=250"`
	Description string   `json:"description" binding:"max=1000"`
	Options     []string `json:"options" binding:"required,min=2,max=10"`
}

// VoteRequest payload when an audience member votes
type VoteRequest struct {
	OptionIndex int    `json:"option_index"`
	VoterID     string `json:"voter_id" binding:"required"`
}

// VoteRecord represents a permanent audit log entry in MongoDB
type VoteRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID      primitive.ObjectID `bson:"poll_id" json:"poll_id"`
	OptionIndex int                `bson:"option_index" json:"option_index"`
	VoterID     string             `bson:"voter_id" json:"voter_id"`
	IP          string             `bson:"ip" json:"ip"`
	UserAgent   string             `bson:"user_agent" json:"user_agent"`
	VotedAt     time.Time          `bson:"voted_at" json:"voted_at"`
}

// WebSocketMessage structure for real-time Redis Pub/Sub and WebSocket events
type WebSocketMessage struct {
	Type        string             `json:"type"` // "initial_state", "vote_update", "status_change", "viewer_count", "error"
	PollID      string             `json:"poll_id"`
	TotalVotes  int64              `json:"total_votes"`
	OptionVotes map[int]int64      `json:"option_votes"`
	Percentages map[int]float64    `json:"percentages"`
	ViewerCount int64              `json:"viewer_count"`
	IsActive    bool               `json:"is_active"`
	Message     string             `json:"message,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

// JWTClaims holds standard and custom token claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}
