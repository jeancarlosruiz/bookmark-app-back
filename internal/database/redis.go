package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient es la instancia global del cliente Redis
var RedisClient *redis.Client

// ConnectRedis inicializa y prueba la conexión con Redis
func ConnectRedis() error {
	ctx := context.Background()

	// Parsear la URL de Redis desde configuración
	redisURL := config.GetRedisURL()
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("error parsing Redis URL: %w", err)
	}

	// Configurar timeouts y pool size para producción
	opts.DialTimeout = 10 * time.Second
	opts.ReadTimeout = 30 * time.Second
	opts.WriteTimeout = 30 * time.Second
	opts.PoolSize = 10
	opts.PoolTimeout = 30 * time.Second

	// Crear cliente Redis
	RedisClient = redis.NewClient(opts)

	// Probar conexión con timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(pingCtx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		log.Println("The application will continue without caching")
		// No retornamos error para que la aplicación pueda funcionar sin Redis
		return nil
	}

	log.Println("Redis connected successfully")
	return nil
}

// CloseRedis cierra la conexión con Redis de forma segura
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// IsRedisAvailable verifica si Redis está disponible y funcionando
func IsRedisAvailable() bool {
	if RedisClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := RedisClient.Ping(ctx).Err()
	return err == nil
}
