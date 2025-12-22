package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/redis/go-redis/v9"
)

// CacheService maneja todas las operaciones de caché para bookmarks
type CacheService struct{}

const (
	// Formato de clave: bookmarks:user:{userId}
	// IMPORTANTE: Siempre incluir userId para evitar fugas de datos entre usuarios
	bookmarkCacheKeyPrefix = "bookmarks:user:"

	// TTL recomendado: 10 minutos
	// - Suficientemente largo para reducir carga en BD
	// - Suficientemente corto para mantener datos frescos
	// - Ajustar según patrones de uso reales
	bookmarkCacheTTL = 10 * time.Minute
)

// GetBookmarksFromCache intenta obtener bookmarks desde caché
// Retorna (bookmarks, hit, error)
// - bookmarks: datos si hay cache hit
// - hit: true si hubo cache hit, false si cache miss
// - error: solo si hay error crítico (no es error si la clave no existe)
func (s *CacheService) GetBookmarksFromCache(ctx context.Context, userID string) ([]models.Bookmarks, bool, error) {
	// Si Redis no está disponible, retornar cache miss sin error
	if !database.IsRedisAvailable() {
		return nil, false, nil
	}

	cacheKey := s.buildCacheKey(userID)

	// Intentar obtener datos del caché
	data, err := database.RedisClient.Get(ctx, cacheKey).Result()

	// Cache miss: la clave no existe (comportamiento esperado)
	if err == redis.Nil {
		log.Printf("Cache MISS for user %s", userID)
		return nil, false, nil
	}

	// Error de conexión u otro problema crítico
	if err != nil {
		log.Printf("Cache error for user %s: %v", userID, err)
		// Retornar cache miss para que la app continúe usando la BD
		return nil, false, nil
	}

	// Cache hit: deserializar datos
	var bookmarks []models.Bookmarks
	if err := json.Unmarshal([]byte(data), &bookmarks); err != nil {
		log.Printf("Cache deserialization error for user %s: %v", userID, err)
		// Invalidar caché corrupto
		s.InvalidateUserCache(ctx, userID)
		return nil, false, nil
	}

	log.Printf("Cache HIT for user %s (%d bookmarks)", userID, len(bookmarks))
	return bookmarks, true, nil
}

// SetBookmarksCache almacena bookmarks en caché para un usuario
func (s *CacheService) SetBookmarksCache(ctx context.Context, userID string, bookmarks []models.Bookmarks) error {
	// Si Redis no está disponible, no hacer nada (la app continúa sin caché)
	if !database.IsRedisAvailable() {
		return nil
	}

	cacheKey := s.buildCacheKey(userID)

	// Serializar bookmarks a JSON
	data, err := json.Marshal(bookmarks)
	if err != nil {
		log.Printf("Cache serialization error for user %s: %v", userID, err)
		return err
	}

	// Guardar en caché con TTL
	err = database.RedisClient.Set(ctx, cacheKey, data, bookmarkCacheTTL).Err()
	if err != nil {
		log.Printf("Cache write error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Cache SET for user %s (%d bookmarks, TTL=%v)", userID, len(bookmarks), bookmarkCacheTTL)
	return nil
}

// InvalidateUserCache invalida el caché de bookmarks para un usuario específico
// CRÍTICO: Llamar esto cuando se crea, actualiza o elimina un bookmark
func (s *CacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	// Si Redis no está disponible, no hacer nada
	if !database.IsRedisAvailable() {
		return nil
	}

	cacheKey := s.buildCacheKey(userID)

	err := database.RedisClient.Del(ctx, cacheKey).Err()
	if err != nil {
		log.Printf("Cache invalidation error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Cache INVALIDATED for user %s", userID)
	return nil
}

// InvalidateAllBookmarkCaches invalida TODOS los cachés de bookmarks
// ADVERTENCIA: Usar solo en casos específicos (ej. migración, mantenimiento)
// En producción, es mejor invalidar cachés por usuario
func (s *CacheService) InvalidateAllBookmarkCaches(ctx context.Context) error {
	// Si Redis no está disponible, no hacer nada
	if !database.IsRedisAvailable() {
		return nil
	}

	// Buscar todas las claves que coincidan con el patrón
	pattern := bookmarkCacheKeyPrefix + "*"
	iter := database.RedisClient.Scan(ctx, 0, pattern, 0).Iterator()

	deletedCount := 0
	for iter.Next(ctx) {
		key := iter.Val()
		if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
			log.Printf("Error deleting cache key %s: %v", key, err)
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error scanning cache keys: %v", err)
		return err
	}

	log.Printf("Invalidated %d bookmark caches", deletedCount)
	return nil
}

// buildCacheKey construye la clave de caché para un usuario
// Formato: bookmarks:user:{userId}
// IMPORTANTE: Siempre incluir userId para prevenir fugas de datos
func (s *CacheService) buildCacheKey(userID string) string {
	return fmt.Sprintf("%s%s", bookmarkCacheKeyPrefix, userID)
}

// GetCacheStats retorna estadísticas del pool de conexiones de Redis
// Útil para monitoreo y debugging
func (s *CacheService) GetCacheStats() *redis.PoolStats {
	if !database.IsRedisAvailable() {
		return nil
	}
	return database.RedisClient.PoolStats()
}
