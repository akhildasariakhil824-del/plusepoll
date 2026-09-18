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
	log.Printf("Connecting to MongoDB at: %s", cfg.MongoURI)
	mongoDB, err := database.InitMongoDB(cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoDB.Client.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	log.Printf("Connecting to Redis at: %s", cfg.RedisURI)
	redisService, err := database.InitRedis(cfg.RedisURI)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to Redis: %v", err)
	}
	defer redisService.Client.Close()

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(mongoDB, cfg)
	pollHandler := handlers.NewPollHandler(mongoDB, redisService)

	// API Group
	api := r.Group("/api")
	{
		// Health Check
		api.GET("/health", func(c *gin.Context) {
			redisErr := redisService.Client.Ping(c.Request.Context()).Err()
			mongoErr := mongoDB.Client.Ping(c.Request.Context(), nil)

			status := "healthy"
			if redisErr != nil || mongoErr != nil {
				status = "degraded"
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  status,
				"service": "PulsePoll Realtime Backend",
				"time":    time.Now().UTC(),
				"checks": gin.H{
					"mongodb": mongoErr == nil,
					"redis":   redisErr == nil,
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
		api.GET("/polls/:id", pollHandler.GetPoll)
		api.POST("/polls/:id/vote", pollHandler.SubmitVote)
		api.GET("/polls/:id/export", pollHandler.ExportPollResults)

		// Realtime WebSocket Endpoint
		api.GET("/ws/polls/:id", func(c *gin.Context) {
			websocket.HandleWebSocket(c, mongoDB, redisService)
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
