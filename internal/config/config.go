package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func Load() error {
	// In production (like Render), .env file won't exist - environment variables
	// are set directly in the platform. So we ignore errors from godotenv.Load()
	// In development, it will load from .env file as expected
	_ = godotenv.Load()
	return nil
}

func GetPort() string {
	return os.Getenv("PORT")
}

func GetRedisURL() string {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		// Default to localhost for development
		return "redis://localhost:6379/0"
	}
	return url
}

func GetAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		// Default to localhost for development
		return []string{"http://localhost:3000"}
	}
	// Split comma-separated origins
	return strings.Split(origins, ",")
}
