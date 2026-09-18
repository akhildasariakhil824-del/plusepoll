package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	MongoURI   string
	MongoDB    string
	RedisURI   string
	JWTSecret  string
	CORSOrigin string
}

func LoadConfig() *Config {
	// Try loading .env if present
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, using system environment variables")
	}

	return &Config{
		Port:       getEnv("PORT", "8080"),
		MongoURI:   getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:    getEnv("MONGO_DB", "pulsepoll"),
		RedisURI:   getEnv("REDIS_URI", "redis://localhost:6379"),
		JWTSecret:  getEnv("JWT_SECRET", "pulsepoll-ultra-secure-jwt-secret-key-32chars"),
		CORSOrigin: getEnv("CORS_ORIGIN", "*"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}
