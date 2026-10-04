package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulsepoll-backend/config"
	"pulsepoll-backend/database"
	"pulsepoll-backend/handlers"
	"pulsepoll-backend/middleware"
	"pulsepoll-backend/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	log.Println("⚡ Starting PulsePoll Real-time Backend...")

	// 1. Initialize DataStore (MongoDB with seamless in-memory fallback if mongod isn't running)
	var store database.DataStore
	mongoStore, err := database.NewMongoStore(cfg.MongoURI, cfg.MongoDB)
	if err == nil {
		log.Printf("✔ Connected to MongoDB successfully at: %s", cfg.MongoURI)
		store = mongoStore
	} else {
		log.Printf("⚠️ MongoDB not detected (%v). Running with In-Memory DataStore + Live Redis!", err)
		store = database.NewMemoryStore()
	}
	defer store.Close(context.Background())

	// 2. Initialize Redis (Real-time live engine)
	log.Printf("Connecting to Redis at: %s", cfg.RedisURI)
	redisService, err := database.InitRedis(cfg.RedisURI)
	if err != nil {
		log.Printf("⚠️ Redis server not reached (%v). Running with High-Performance In-Memory Pub/Sub & Atomic Engine fallback!", err)
		redisService = database.NewMemoryRedisService()
	}
	defer redisService.Close()

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(store, cfg)
	pollHandler := handlers.NewPollHandler(store, redisService)

	// API Group
	api := r.Group("/api")
	{
		// Health Check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "healthy",
				"service": "PulsePoll Realtime Backend",
				"time":    time.Now().UTC(),
				"checks": gin.H{
					"redis":         redisService.IsRealRedis(),
					"realtime_mode": func() string { if redisService.IsRealRedis() { return "Redis Pub/Sub + Atomic Counters" } else { return "In-Memory Engine (Start Redis for multi-instance cluster)" } }(),
					"storage":       store.IsConnected(),
				},
			})
		})

		// Public Auth Routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", middleware.JWTAuth(cfg.JWTSecret), authHandler.Me)
		}

		// Public Poll Routes & Voting
		api.GET("/polls", pollHandler.ListPublicPolls)
		api.GET("/polls/:id", pollHandler.GetPoll)
		api.POST("/polls/:id/vote", pollHandler.SubmitVote)
		api.GET("/polls/:id/export", pollHandler.ExportPollResults)

		// Realtime WebSocket Endpoint
		api.GET("/ws/polls/:id", func(c *gin.Context) {
			websocket.HandleWebSocket(c, store, redisService)
		})

		// Protected Poll Routes (Requires JWT)
		protected := api.Group("/polls")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			protected.POST("", pollHandler.CreatePoll)
			protected.GET("/my", pollHandler.ListMyPolls)
			protected.PATCH("/:id/status", pollHandler.TogglePollStatus)
			protected.DELETE("/:id", pollHandler.DeletePoll)
		}
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 PulsePoll backend listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly.")
}
