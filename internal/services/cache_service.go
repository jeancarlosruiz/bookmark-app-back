package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	// Formato de clave para bookmarks archivados: bookmarks:user:{userId}:archived
	// Se mantienen separados de los activos para evitar colisiones
	archivedBookmarkCacheKeyPrefix = "bookmarks:user:"
	archivedBookmarkCacheKeySuffix = ":archived"

	// TTL recomendado: 10 minutos
	// - Suficientemente largo para reducir carga en BD
	// - Suficientemente corto para mantener datos frescos
	// - Ajustar según patrones de uso reales
	bookmarkCacheTTL = 10 * time.Minute

	// Formato de clave para metadata: metadata:url:{hash(url)}
	// Usar hash SHA256 de la URL para evitar problemas con caracteres especiales
	metadataCacheKeyPrefix = "metadata:url:"

	// TTL recomendado para metadata: 24 horas
	// - Los metadatos de URLs raramente cambian
	// - Reduce carga en sitios externos (scraping)
	// - Protege contra rate limiting
	// - Mejora UX con respuestas instantáneas
	metadataCacheTTL = 24 * time.Hour

	// Formato de clave para tags: tags:user:{userId}
	// Similar a bookmarks, incluir userId para aislamiento
	tagsCacheKeyPrefix = "tags:user:"

	// TTL recomendado para tags: 15 minutos
	// - Los tags cambian poco (solo cuando se crean/modifican bookmarks)
	// - Query compleja (JOIN + GROUP BY + COUNT)
	// - Alta tasa de lectura vs escritura
	tagsCacheTTL = 15 * time.Minute
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

// === ARCHIVED BOOKMARKS CACHE ===

// GetArchivedBookmarksFromCache intenta obtener bookmarks archivados desde caché
// Retorna (bookmarks, hit, error)
// - bookmarks: datos si hay cache hit
// - hit: true si hubo cache hit, false si cache miss
// - error: solo si hay error crítico (no es error si la clave no existe)
func (s *CacheService) GetArchivedBookmarksFromCache(ctx context.Context, userID string) ([]models.Bookmarks, bool, error) {
	// Si Redis no está disponible, retornar cache miss sin error
	if !database.IsRedisAvailable() {
		return nil, false, nil
	}

	cacheKey := s.buildArchivedCacheKey(userID)

	// Intentar obtener datos del caché
	data, err := database.RedisClient.Get(ctx, cacheKey).Result()

	// Cache miss: la clave no existe (comportamiento esperado)
	if err == redis.Nil {
		log.Printf("Archived cache MISS for user %s", userID)
		return nil, false, nil
	}

	// Error de conexión u otro problema crítico
	if err != nil {
		log.Printf("Archived cache error for user %s: %v", userID, err)
		// Retornar cache miss para que la app continúe usando la BD
		return nil, false, nil
	}

	// Cache hit: deserializar datos
	var bookmarks []models.Bookmarks
	if err := json.Unmarshal([]byte(data), &bookmarks); err != nil {
		log.Printf("Archived cache deserialization error for user %s: %v", userID, err)
		// Invalidar caché corrupto
		database.RedisClient.Del(ctx, cacheKey)
		return nil, false, nil
	}

	log.Printf("Archived cache HIT for user %s (%d bookmarks)", userID, len(bookmarks))
	return bookmarks, true, nil
}

// SetArchivedBookmarksCache almacena bookmarks archivados en caché para un usuario
func (s *CacheService) SetArchivedBookmarksCache(ctx context.Context, userID string, bookmarks []models.Bookmarks) error {
	// Si Redis no está disponible, no hacer nada (la app continúa sin caché)
	if !database.IsRedisAvailable() {
		return nil
	}

	cacheKey := s.buildArchivedCacheKey(userID)

	// Serializar bookmarks a JSON
	data, err := json.Marshal(bookmarks)
	if err != nil {
		log.Printf("Archived cache serialization error for user %s: %v", userID, err)
		return err
	}

	// Guardar en caché con TTL
	err = database.RedisClient.Set(ctx, cacheKey, data, bookmarkCacheTTL).Err()
	if err != nil {
		log.Printf("Archived cache write error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Archived cache SET for user %s (%d bookmarks, TTL=%v)", userID, len(bookmarks), bookmarkCacheTTL)
	return nil
}

// InvalidateUserCache invalida TODOS los cachés relacionados con bookmarks para un usuario
// (bookmarks activos, archivados y tags)
// CRÍTICO: Llamar esto cuando se crea, actualiza o elimina un bookmark
// NOTA: Invalida todos los cachés porque:
// - Un bookmark puede cambiar entre activo/archivado
// - Los tags se ven afectados (conteo de bookmarks por tag)
func (s *CacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	// Si Redis no está disponible, no hacer nada
	if !database.IsRedisAvailable() {
		return nil
	}

	activeCacheKey := s.buildCacheKey(userID)
	archivedCacheKey := s.buildArchivedCacheKey(userID)
	tagsCacheKey := s.buildTagsCacheKey(userID)

	// Invalidar caché de bookmarks activos
	err := database.RedisClient.Del(ctx, activeCacheKey).Err()
	if err != nil {
		log.Printf("Active cache invalidation error for user %s: %v", userID, err)
		return err
	}

	// Invalidar caché de bookmarks archivados
	err = database.RedisClient.Del(ctx, archivedCacheKey).Err()
	if err != nil {
		log.Printf("Archived cache invalidation error for user %s: %v", userID, err)
		return err
	}

	// Invalidar caché de tags
	err = database.RedisClient.Del(ctx, tagsCacheKey).Err()
	if err != nil {
		log.Printf("Tags cache invalidation error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Cache INVALIDATED for user %s (active, archived & tags)", userID)
	return nil
}

// InvalidateAllBookmarkCaches invalida TODOS los cachés de bookmarks y tags
// (activos, archivados y tags)
// ADVERTENCIA: Usar solo en casos específicos (ej. migración, mantenimiento)
// En producción, es mejor invalidar cachés por usuario
func (s *CacheService) InvalidateAllBookmarkCaches(ctx context.Context) error {
	// Si Redis no está disponible, no hacer nada
	if !database.IsRedisAvailable() {
		return nil
	}

	deletedCount := 0

	// 1. Invalidar todos los cachés de bookmarks (activos y archivados)
	// Patrón: bookmarks:user:*
	bookmarkPattern := bookmarkCacheKeyPrefix + "*"
	bookmarkIter := database.RedisClient.Scan(ctx, 0, bookmarkPattern, 0).Iterator()

	for bookmarkIter.Next(ctx) {
		key := bookmarkIter.Val()
		if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
			log.Printf("Error deleting cache key %s: %v", key, err)
		} else {
			deletedCount++
		}
	}

	if err := bookmarkIter.Err(); err != nil {
		log.Printf("Error scanning bookmark cache keys: %v", err)
		return err
	}

	// 2. Invalidar todos los cachés de tags
	// Patrón: tags:user:*
	tagsPattern := tagsCacheKeyPrefix + "*"
	tagsIter := database.RedisClient.Scan(ctx, 0, tagsPattern, 0).Iterator()

	for tagsIter.Next(ctx) {
		key := tagsIter.Val()
		if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
			log.Printf("Error deleting cache key %s: %v", key, err)
		} else {
			deletedCount++
		}
	}

	if err := tagsIter.Err(); err != nil {
		log.Printf("Error scanning tags cache keys: %v", err)
		return err
	}

	log.Printf("Invalidated %d caches (bookmarks & tags)", deletedCount)
	return nil
}

// buildCacheKey construye la clave de caché para un usuario
// Formato: bookmarks:user:{userId}
// IMPORTANTE: Siempre incluir userId para prevenir fugas de datos
func (s *CacheService) buildCacheKey(userID string) string {
	return fmt.Sprintf("%s%s", bookmarkCacheKeyPrefix, userID)
}

// buildArchivedCacheKey construye la clave de caché para bookmarks archivados de un usuario
// Formato: bookmarks:user:{userId}:archived
// IMPORTANTE: Mantener separado del caché de bookmarks activos
func (s *CacheService) buildArchivedCacheKey(userID string) string {
	return fmt.Sprintf("%s%s%s", archivedBookmarkCacheKeyPrefix, userID, archivedBookmarkCacheKeySuffix)
}

// GetCacheStats retorna estadísticas del pool de conexiones de Redis
// Útil para monitoreo y debugging
func (s *CacheService) GetCacheStats() *redis.PoolStats {
	if !database.IsRedisAvailable() {
		return nil
	}
	return database.RedisClient.PoolStats()
}

// === METADATA CACHING ===

// GetMetadataFromCache intenta obtener metadata de una URL desde caché
// Retorna (metadata, hit, error)
// - metadata: datos si hay cache hit
// - hit: true si hubo cache hit, false si cache miss
// - error: solo si hay error crítico (no es error si la clave no existe)
func (s *CacheService) GetMetadataFromCache(ctx context.Context, url string) (*MetadataResult, bool, error) {
	// Si Redis no está disponible, retornar cache miss sin error
	if !database.IsRedisAvailable() {
		return nil, false, nil
	}

	cacheKey := s.buildMetadataCacheKey(url)

	// Intentar obtener datos del caché
	data, err := database.RedisClient.Get(ctx, cacheKey).Result()

	// Cache miss: la clave no existe (comportamiento esperado)
	if err == redis.Nil {
		log.Printf("Metadata cache MISS for URL: %s", url)
		return nil, false, nil
	}

	// Error de conexión u otro problema crítico
	if err != nil {
		log.Printf("Metadata cache error for URL %s: %v", url, err)
		// Retornar cache miss para que la app continúe con scraping
		return nil, false, nil
	}

	// Cache hit: deserializar datos
	var metadata MetadataResult
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		log.Printf("Metadata cache deserialization error for URL %s: %v", url, err)
		// Invalidar caché corrupto
		database.RedisClient.Del(ctx, cacheKey)
		return nil, false, nil
	}

	log.Printf("Metadata cache HIT for URL: %s", url)
	return &metadata, true, nil
}

// SetMetadataCache almacena metadata de una URL en caché
// Solo cachea si no hay errores críticos (permite cachear metadata parcial)
func (s *CacheService) SetMetadataCache(ctx context.Context, url string, metadata *MetadataResult) error {
	// Si Redis no está disponible, no hacer nada (la app continúa sin caché)
	if !database.IsRedisAvailable() {
		return nil
	}

	// No cachear errores de timeout/red (pueden ser temporales)
	if metadata.Error != nil {
		errorMsg := metadata.Error.Error()
		// Solo cachear si hay datos parciales útiles
		if metadata.Title == "" && metadata.Description == "" && metadata.Favicon == "" {
			log.Printf("Skipping cache for URL %s due to complete failure: %v", url, metadata.Error)
			return nil
		}
		log.Printf("Caching partial metadata for URL %s (error: %s)", url, errorMsg)
	}

	cacheKey := s.buildMetadataCacheKey(url)

	// Serializar metadata a JSON
	data, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("Metadata cache serialization error for URL %s: %v", url, err)
		return err
	}

	// Guardar en caché con TTL
	err = database.RedisClient.Set(ctx, cacheKey, data, metadataCacheTTL).Err()
	if err != nil {
		log.Printf("Metadata cache write error for URL %s: %v", url, err)
		return err
	}

	log.Printf("Metadata cache SET for URL: %s (TTL=%v)", url, metadataCacheTTL)
	return nil
}

// buildMetadataCacheKey construye la clave de caché para metadata de una URL
// Formato: metadata:url:{hash(url)}
// Usa hash SHA256 para evitar problemas con caracteres especiales en URLs
func (s *CacheService) buildMetadataCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s%s", metadataCacheKeyPrefix, hashStr)
}

// === TAGS CACHING ===

// GetTagsFromCache intenta obtener tags desde caché
// Retorna (tags, hit, error)
// - tags: datos si hay cache hit
// - hit: true si hubo cache hit, false si cache miss
// - error: solo si hay error crítico (no es error si la clave no existe)
func (s *CacheService) GetTagsFromCache(ctx context.Context, userID string) ([]models.TagWithCount, bool, error) {
	// Si Redis no está disponible, retornar cache miss sin error
	if !database.IsRedisAvailable() {
		return nil, false, nil
	}

	cacheKey := s.buildTagsCacheKey(userID)

	// Intentar obtener datos del caché
	data, err := database.RedisClient.Get(ctx, cacheKey).Result()

	// Cache miss: la clave no existe (comportamiento esperado)
	if err == redis.Nil {
		log.Printf("Tags cache MISS for user %s", userID)
		return nil, false, nil
	}

	// Error de conexión u otro problema crítico
	if err != nil {
		log.Printf("Tags cache error for user %s: %v", userID, err)
		// Retornar cache miss para que la app continúe usando la BD
		return nil, false, nil
	}

	// Cache hit: deserializar datos
	var tags []models.TagWithCount
	if err := json.Unmarshal([]byte(data), &tags); err != nil {
		log.Printf("Tags cache deserialization error for user %s: %v", userID, err)
		// Invalidar caché corrupto
		database.RedisClient.Del(ctx, cacheKey)
		return nil, false, nil
	}

	log.Printf("Tags cache HIT for user %s (%d tags)", userID, len(tags))
	return tags, true, nil
}

// SetTagsCache almacena tags en caché para un usuario
func (s *CacheService) SetTagsCache(ctx context.Context, userID string, tags []models.TagWithCount) error {
	// Si Redis no está disponible, no hacer nada (la app continúa sin caché)
	if !database.IsRedisAvailable() {
		return nil
	}

	cacheKey := s.buildTagsCacheKey(userID)

	// Serializar tags a JSON
	data, err := json.Marshal(tags)
	if err != nil {
		log.Printf("Tags cache serialization error for user %s: %v", userID, err)
		return err
	}

	// Guardar en caché con TTL
	err = database.RedisClient.Set(ctx, cacheKey, data, tagsCacheTTL).Err()
	if err != nil {
		log.Printf("Tags cache write error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Tags cache SET for user %s (%d tags, TTL=%v)", userID, len(tags), tagsCacheTTL)
	return nil
}

// InvalidateTagsCache invalida el caché de tags para un usuario específico
// CRÍTICO: Llamar esto cuando se crean, actualizan o eliminan bookmarks con tags
func (s *CacheService) InvalidateTagsCache(ctx context.Context, userID string) error {
	// Si Redis no está disponible, no hacer nada
	if !database.IsRedisAvailable() {
		return nil
	}

	cacheKey := s.buildTagsCacheKey(userID)

	err := database.RedisClient.Del(ctx, cacheKey).Err()
	if err != nil {
		log.Printf("Tags cache invalidation error for user %s: %v", userID, err)
		return err
	}

	log.Printf("Tags cache INVALIDATED for user %s", userID)
	return nil
}

// buildTagsCacheKey construye la clave de caché para tags de un usuario
// Formato: tags:user:{userId}
// IMPORTANTE: Siempre incluir userId para prevenir fugas de datos
func (s *CacheService) buildTagsCacheKey(userID string) string {
	return fmt.Sprintf("%s%s", tagsCacheKeyPrefix, userID)
}
