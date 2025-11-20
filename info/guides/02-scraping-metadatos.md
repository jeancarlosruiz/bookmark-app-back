# Día 13: Scraping de Metadatos para Bookmarks

Esta guía explica cómo implementar funcionalidad de web scraping para extraer automáticamente el título, favicon y descripción de las URLs cuando se crean bookmarks.

## Descripción General

Implementaremos un servicio de scraping que:

- Extrae el título de la página desde el tag `<title>` o Open Graph
- Obtiene el favicon desde múltiples fuentes posibles
- Extrae la descripción desde meta tags (description, og:description, twitter:description)
- Se integra automáticamente en el flujo de creación de bookmarks
- Maneja errores de forma elegante (campos opcionales si el scraping falla)
- Aplica timeouts para evitar bloqueos en sitios lentos

## Beneficios

- **Experiencia de usuario mejorada**: Los usuarios no tienen que ingresar manualmente el título y descripción
- **Datos consistentes**: Información estandarizada extraída directamente de las páginas web
- **Ahorro de tiempo**: Creación rápida de bookmarks con solo proporcionar la URL
- **Fallback robusto**: Si el scraping falla, el usuario aún puede crear el bookmark manualmente

## Dependencias Requeridas

### Instalar goquery

```bash
go get github.com/PuerkitoBio/goquery
```

Esta biblioteca proporciona una API similar a jQuery para parsear y manipular documentos HTML en Go.

### Actualizar go.mod

Después de instalar, ejecuta:

```bash
go mod tidy
```

## Arquitectura de la Implementación

Seguiremos el patrón de capas del proyecto:

```
1. Scraper Service (nueva capa de utilidad)
   ↓
2. Bookmark Service (integración del scraper)
   ↓
3. Bookmark Controller (manejo de datos scraped)
   ↓
4. Validator (actualización de validaciones)
```

## Paso 1: Crear el Servicio de Scraping

### 1.1 Crear el archivo del servicio

Crear el archivo: `internal/services/scraper_service.go`

```go
package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// MetadataResult contiene los metadatos extraídos de una URL
type MetadataResult struct {
	Title       string
	Description string
	Favicon     string
	Error       error
}

// ScraperService maneja la extracción de metadatos de URLs
type ScraperService struct {
	client  *http.Client
	timeout time.Duration
}

// NewScraperService crea una nueva instancia del servicio de scraping
func NewScraperService() *ScraperService {
	return &ScraperService{
		client: &http.Client{
			Timeout: 10 * time.Second, // Timeout de 10 segundos
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Limitar a 10 redirects
				if len(via) >= 10 {
					return errors.New("demasiados redirects")
				}
				return nil
			},
		},
		timeout: 10 * time.Second,
	}
}

// ScrapeMetadata extrae título, descripción y favicon de una URL
func (s *ScraperService) ScrapeMetadata(targetURL string) *MetadataResult {
	result := &MetadataResult{}

	// Validar la URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.Error = fmt.Errorf("URL inválida: %w", err)
		return result
	}

	// Asegurar que la URL tenga un esquema
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		targetURL = parsedURL.String()
	}

	// Hacer la petición HTTP
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("error creando petición: %w", err)
		return result
	}

	// Agregar headers para parecer un navegador real
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	// Ejecutar la petición
	resp, err := s.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("error haciendo petición: %w", err)
		return result
	}
	defer resp.Body.Close()

	// Verificar el status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("status code inválido: %d", resp.StatusCode)
		return result
	}

	// Parsear el HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("error parseando HTML: %w", err)
		return result
	}

	// Extraer metadatos
	result.Title = s.extractTitle(doc)
	result.Description = s.extractDescription(doc)
	result.Favicon = s.extractFavicon(doc, parsedURL)

	return result
}

// extractTitle extrae el título de la página
// Prioridad: og:title > twitter:title > <title>
func (s *ScraperService) extractTitle(doc *goquery.Document) string {
	// Intentar Open Graph title
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists && title != "" {
		return strings.TrimSpace(title)
	}

	// Intentar Twitter title
	if title, exists := doc.Find("meta[name='twitter:title']").Attr("content"); exists && title != "" {
		return strings.TrimSpace(title)
	}

	// Intentar tag <title>
	if title := doc.Find("title").First().Text(); title != "" {
		return strings.TrimSpace(title)
	}

	return ""
}

// extractDescription extrae la descripción de la página
// Prioridad: og:description > twitter:description > meta description
func (s *ScraperService) extractDescription(doc *goquery.Document) string {
	// Intentar Open Graph description
	if desc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	// Intentar Twitter description
	if desc, exists := doc.Find("meta[name='twitter:description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	// Intentar meta description estándar
	if desc, exists := doc.Find("meta[name='description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	return ""
}

// extractFavicon extrae la URL del favicon
// Busca en múltiples ubicaciones posibles
func (s *ScraperService) extractFavicon(doc *goquery.Document, baseURL *url.URL) string {
	var faviconPath string

	// Lista de selectores de favicon en orden de preferencia
	selectors := []struct {
		selector string
		attr     string
	}{
		// Favicon de alta resolución
		{"link[rel='apple-touch-icon']", "href"},
		{"link[rel='icon'][sizes='192x192']", "href"},
		{"link[rel='icon'][sizes='180x180']", "href"},
		// Favicon estándar
		{"link[rel='icon']", "href"},
		{"link[rel='shortcut icon']", "href"},
		// Open Graph image como fallback
		{"meta[property='og:image']", "content"},
	}

	// Buscar favicon usando los selectores
	for _, sel := range selectors {
		if path, exists := doc.Find(sel.selector).First().Attr(sel.attr); exists && path != "" {
			faviconPath = path
			break
		}
	}

	// Si no se encontró favicon, usar el favicon por defecto
	if faviconPath == "" {
		faviconPath = "/favicon.ico"
	}

	// Convertir a URL absoluta
	faviconURL, err := url.Parse(faviconPath)
	if err != nil {
		return ""
	}

	// Si es una URL relativa, resolverla
	if !faviconURL.IsAbs() {
		faviconURL = baseURL.ResolveReference(faviconURL)
	}

	return faviconURL.String()
}

// ScrapeMetadataAsync ejecuta el scraping de forma asíncrona con timeout
func (s *ScraperService) ScrapeMetadataAsync(targetURL string, timeout time.Duration) *MetadataResult {
	resultChan := make(chan *MetadataResult, 1)

	go func() {
		resultChan <- s.ScrapeMetadata(targetURL)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-time.After(timeout):
		return &MetadataResult{
			Error: errors.New("timeout: el scraping tardó demasiado tiempo"),
		}
	}
}
```

## Paso 2: Actualizar el Validador de Bookmarks

### 2.1 Modificar `internal/validator/bookmark_validator.go`

Actualizar la estructura `CreateBookmark` para hacer el título opcional (ya que ahora se puede extraer automáticamente):

```go
package validator

type CreateBookmark struct {
	Title       string   `json:"title" validate:"omitempty,min=1,max=200"`
	Url         string   `json:"url" validate:"required,url"`
	Description string   `json:"description" validate:"omitempty,max=1000"`
	Favicon     string   `json:"favicon" validate:"omitempty,url"`
	UserID      string   `json:"user_id" validate:"required"`
	Tags        []string `json:"tags" validate:"omitempty,dive,min=1"`
}

type UpdateBookmark struct {
	Title       *string  `json:"title" validate:"omitempty,min=1,max=200"`
	Url         *string  `json:"url" validate:"omitempty,url,http_url"`
	Description *string  `json:"description" validate:"omitempty,max=1000"`
	Favicon     *string  `json:"favicon" validate:"omitempty,url"`
	Pinned      *bool    `json:"pinned"`
	IsArchived  *bool    `json:"is_archived"`
	Tags        []string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
}
```

**Cambios importantes:**

- `Title` ahora es `omitempty` en lugar de `required`
- Se agregaron campos `Description` y `Favicon` al validador `CreateBookmark`

## Paso 3: Integrar el Scraper en el Servicio de Bookmarks

### 3.1 Actualizar `internal/services/bookmark_service.go`

Modificar el servicio para incluir el scraper:

```go
package services

import (
	"errors"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)

type BookmarkService struct {
	bookmarkRepo   *repositories.BookmarkRepository
	tagService     *TagService
	scraperService *ScraperService // ← NUEVO
}

func NewBookmarkService() *BookmarkService {
	return &BookmarkService{
		bookmarkRepo:   repositories.NewBookmarkRepository(),
		tagService:     NewTagService(),
		scraperService: NewScraperService(), // ← NUEVO
	}
}

func (s *BookmarkService) CreateBookmarkService(data validator.CreateBookmark) (*models.Bookmarks, error) {
	// Verificar duplicados
	existing, _ := s.bookmarkRepo.FindByTitleOrURL(data.Title, data.Url, data.UserID)
	if existing != nil {
		return nil, ErrBookmarkAlreadyExists
	}

	// ← NUEVO: Scraping de metadatos si no se proporcionaron
	metadata := &MetadataResult{}
	shouldScrape := data.Title == "" || data.Description == "" || data.Favicon == ""

	if shouldScrape {
		// Scraping con timeout de 8 segundos
		metadata = s.scraperService.ScrapeMetadataAsync(data.Url, 8*time.Second)

		// Log del error de scraping pero no fallar la creación
		if metadata.Error != nil {
			// Podrías usar un logger aquí
			fmt.Printf("Warning: Error al scrapear metadatos: %v\n", metadata.Error)
		}
	}

	// Usar valores scraped como fallback
	title := data.Title
	if title == "" && metadata.Title != "" {
		title = metadata.Title
	}
	// Si aún no hay título, usar la URL como fallback
	if title == "" {
		title = data.Url
	}

	description := data.Description
	if description == "" && metadata.Description != "" {
		description = metadata.Description
	}

	favicon := data.Favicon
	if favicon == "" && metadata.Favicon != "" {
		favicon = metadata.Favicon
	}

	// Procesar tags
	tags, err := s.tagService.FindOrCreateTags(data.Tags, data.UserID)
	if err != nil {
		return nil, err
	}

	// Crear el bookmark
	bookmark := &models.Bookmarks{
		Title:       title,
		Url:         data.Url,
		Description: description,
		Favicon:     favicon,
		UserID:      data.UserID,
		Tags:        tags,
	}

	if err := s.bookmarkRepo.Create(bookmark); err != nil {
		return nil, err
	}

	// Recuperar el bookmark con tags
	bookmark, err = s.bookmarkRepo.FindByIDWithTags(bookmark.ID, bookmark.UserID)
	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

// ... resto de métodos sin cambios ...

var (
	ErrBookmarkAlreadyExists = errors.New("bookmark with this title or URL already exists")
	ErrBookmarksNotFound     = errors.New("Bookmarks not found")
)
```

**Nota:** No olvides agregar los imports necesarios:

```go
import (
	"errors"
	"fmt"      // ← NUEVO
	"time"     // ← NUEVO

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)
```

## Paso 4: Actualizar el Controlador (Opcional)

El controlador no necesita cambios significativos ya que todo el scraping ocurre en la capa de servicio. Sin embargo, puedes agregar información adicional en la respuesta:

### 4.1 Modificar `internal/controllers/bookmark_controller.go`

```go
func (ctrl *BookmarkController) CreateBookmark(c *gin.Context) {
	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})
		return
	}

	bookmarkData := payload.(validator.CreateBookmark)

	bookmark, err := ctrl.service.CreateBookmarkService(bookmarkData)

	if err != nil {
		if err == services.ErrBookmarkAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create bookmark",
			"error":   err.Error(),
		})
		return
	}

	// Respuesta mejorada indicando si se usó scraping
	wasScraped := bookmarkData.Title == "" || bookmarkData.Description == "" || bookmarkData.Favicon == ""

	c.JSON(http.StatusOK, gin.H{
		"message":     "Bookmark created successfully",
		"data":        bookmark,
		"scraped":     wasScraped, // ← NUEVO: Indica si se usó scraping
	})
}
```

## Paso 5: Manejo de Errores y Casos Especiales

### 5.1 Casos a considerar:

1. **URLs sin esquema**: El scraper automáticamente agrega `https://` si falta
2. **Timeouts**: Se usa un timeout de 8 segundos para evitar bloqueos
3. **Redirects**: Se permiten hasta 10 redirects
4. **Sitios lentos**: El scraping es asíncrono con timeout
5. **Errores de scraping**: No bloquean la creación del bookmark, se usan valores por defecto

### 5.2 Valores de Fallback:

```
Si el scraping falla:
- Title: Usa la URL como título
- Description: Queda vacío (campo opcional)
- Favicon: Queda vacío (campo opcional)
```

## Paso 6: Testing

### 6.1 Crear archivo de pruebas manuales

Crear `test_scraping.http` (para usar con extensiones como REST Client):

```http
### Test 1: Crear bookmark con scraping automático (sin título)
POST http://localhost:8080/api/bookmark
Content-Type: application/json

{
  "url": "https://go.dev",
  "user_id": "test-user-123",
  "tags": ["Programming", "Go"]
}

### Test 2: Crear bookmark con título manual (sin scraping)
POST http://localhost:8080/api/bookmark
Content-Type: application/json

{
  "title": "Mi Título Personalizado",
  "url": "https://golang.org",
  "user_id": "test-user-123",
  "tags": ["Go"]
}

### Test 3: Crear bookmark con scraping parcial
POST http://localhost:8080/api/bookmark
Content-Type: application/json

{
  "title": "Go Documentation",
  "url": "https://pkg.go.dev",
  "user_id": "test-user-123",
  "tags": ["Docs"]
}

### Test 4: Verificar que funcionan URLs conocidas
POST http://localhost:8080/api/bookmark
Content-Type: application/json

{
  "url": "https://github.com",
  "user_id": "test-user-123",
  "tags": ["Tools"]
}
```

### 6.2 Testing con cURL

```bash
# Test con scraping automático
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://go.dev",
    "user_id": "test-user-123",
    "tags": ["Programming"]
  }'

# Test con título manual
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Website",
    "url": "https://go.dev",
    "user_id": "test-user-123",
    "tags": ["Programming"]
  }'
```

### 6.3 Verificar resultados

```bash
# Obtener todos los bookmarks del usuario
curl http://localhost:8080/api/bookmark/user/test-user-123
```

## Paso 7: Optimizaciones Adicionales (Opcional)

### 7.1 Agregar cache para metadatos

Crear `internal/services/metadata_cache.go`:

```go
package services

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Metadata  *MetadataResult
	Timestamp time.Time
}

type MetadataCache struct {
	cache map[string]*CacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

func NewMetadataCache(ttl time.Duration) *MetadataCache {
	return &MetadataCache{
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}
}

func (c *MetadataCache) Get(url string) (*MetadataResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[url]
	if !exists {
		return nil, false
	}

	// Verificar si la entrada expiró
	if time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}

	return entry.Metadata, true
}

func (c *MetadataCache) Set(url string, metadata *MetadataResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[url] = &CacheEntry{
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
}

func (c *MetadataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
}
```

### 7.2 Integrar el cache en ScraperService

```go
type ScraperService struct {
	client  *http.Client
	timeout time.Duration
	cache   *MetadataCache // ← NUEVO
}

func NewScraperService() *ScraperService {
	return &ScraperService{
		client: &http.Client{
			Timeout: 10 * time.Second,
			// ...
		},
		timeout: 10 * time.Second,
		cache:   NewMetadataCache(1 * time.Hour), // Cache de 1 hora
	}
}

func (s *ScraperService) ScrapeMetadata(targetURL string) *MetadataResult {
	// Intentar obtener del cache
	if cached, found := s.cache.Get(targetURL); found {
		return cached
	}

	// Realizar scraping normal
	result := &MetadataResult{}

	// ... código de scraping existente ...

	// Guardar en cache si fue exitoso
	if result.Error == nil {
		s.cache.Set(targetURL, result)
	}

	return result
}
```

### 7.3 Agregar endpoint para scrapear metadatos sin crear bookmark

En `internal/controllers/bookmark_controller.go`:

```go
func (ctrl *BookmarkController) PreviewMetadata(c *gin.Context) {
	url := c.Query("url")

	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "URL es requerida",
		})
		return
	}

	scraperService := services.NewScraperService()
	metadata := scraperService.ScrapeMetadataAsync(url, 8*time.Second)

	if metadata.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se pudieron obtener metadatos completos",
			"data": gin.H{
				"title":       metadata.Title,
				"description": metadata.Description,
				"favicon":     metadata.Favicon,
			},
			"error": metadata.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Metadatos obtenidos exitosamente",
		"data": gin.H{
			"title":       metadata.Title,
			"description": metadata.Description,
			"favicon":     metadata.Favicon,
		},
	})
}
```

Registrar la ruta en `internal/routes/routes.go`:

```go
// Preview de metadatos antes de crear bookmark
apiGroup.GET("/bookmark/preview", bookmarkController.PreviewMetadata)
```

## Resumen de Archivos Modificados y Creados

### Archivos Nuevos:

1. `internal/services/scraper_service.go` - Servicio principal de scraping
2. `internal/services/metadata_cache.go` - Cache de metadatos (opcional)
3. `test_scraping.http` - Pruebas manuales (opcional)

### Archivos Modificados:

1. `internal/validator/bookmark_validator.go` - Hacer título opcional
2. `internal/services/bookmark_service.go` - Integrar scraping en creación
3. `internal/controllers/bookmark_controller.go` - Respuesta mejorada (opcional)
4. `internal/routes/routes.go` - Agregar endpoint de preview (opcional)
5. `go.mod` / `go.sum` - Dependencia de goquery

## Ejemplo de Flujo Completo

### 1. Usuario crea bookmark con solo URL:

```json
POST /api/bookmark
{
  "url": "https://go.dev",
  "user_id": "user-123",
  "tags": ["Programming"]
}
```

### 2. El sistema automáticamente:

1. Valida la URL
2. Detecta que no hay título
3. Ejecuta el scraping con timeout de 8s
4. Extrae:
   - Title: "Go Programming Language"
   - Description: "Build simple, secure, scalable systems with Go"
   - Favicon: "https://go.dev/images/favicon.ico"
5. Crea el bookmark con los datos scraped
6. Responde con el bookmark creado

### 3. Respuesta:

```json
{
  "message": "Bookmark created successfully",
  "scraped": true,
  "data": {
    "id": 1,
    "title": "Go Programming Language",
    "url": "https://go.dev",
    "favicon": "https://go.dev/images/favicon.ico",
    "description": "Build simple, secure, scalable systems with Go",
    "pinned": false,
    "is_archived": false,
    "visit_count": 0,
    "tags": [
      {
        "id": 5,
        "title": "Programming"
      }
    ],
    "user_id": "user-123",
    "created_at": "2024-11-19T10:30:00Z"
  }
}
```

## Consideraciones de Producción

### Seguridad:

1. **Rate Limiting**: Implementar límite de requests de scraping por usuario
2. **Whitelist/Blacklist**: Considerar bloquear ciertos dominios
3. **Validación de contenido**: Sanitizar metadatos extraídos
4. **HTTPS preferido**: Siempre intentar HTTPS primero

### Performance:

1. **Cache**: Implementar cache de metadatos (ver sección 7.1)
2. **Queue**: Para producción, considerar una cola de scraping asíncrona
3. **Timeout adecuado**: Ajustar según necesidades (actualmente 8s)
4. **Retry logic**: Agregar reintentos con backoff exponencial

### Mantenimiento:

1. **Logging**: Implementar logging robusto de errores de scraping
2. **Metrics**: Trackear tasa de éxito/fallo del scraping
3. **Monitoring**: Alertas si la tasa de fallo supera un umbral

## Troubleshooting

### Error: "timeout: el scraping tardó demasiado tiempo"

- El sitio objetivo es muy lento
- Aumentar el timeout en `ScrapeMetadataAsync`
- Verificar conectividad de red

### Error: "status code inválido: 403"

- El sitio bloquea bots/scrapers
- Algunos sitios requieren headers específicos
- Considerar agregar el sitio a una blacklist

### No se extrae el favicon correctamente

- Algunos sitios usan rutas no estándar
- Verificar manualmente en el HTML del sitio
- Considerar agregar más selectores en `extractFavicon`

### Títulos muy largos o con caracteres especiales

- Agregar límite de caracteres en el scraper
- Sanitizar caracteres especiales
- Implementar truncado inteligente

## Próximos Pasos

Después de implementar el scraping básico, considera:

1. **Día 14**: Implementar actualización automática de metadatos
2. **Día 15**: Agregar verificación periódica de enlaces rotos
3. **Día 16**: Implementar preview de bookmarks con screenshots
4. **Día 17**: Agregar extracción de tags automática basada en contenido

## Recursos Adicionales

- [Documentación de goquery](https://github.com/PuerkitoBio/goquery)
- [Open Graph Protocol](https://ogp.me/)
- [Twitter Cards](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)
- [HTML Meta Tags](https://www.w3schools.com/tags/tag_meta.asp)

## Conclusión

Con esta implementación, tu API de bookmarks ahora puede:

- Extraer automáticamente metadatos de URLs
- Proporcionar una mejor experiencia de usuario
- Manejar errores de forma elegante
- Ser extendida fácilmente con más funcionalidades

El scraping de metadatos es una feature fundamental que hace que crear bookmarks sea rápido y conveniente, mientras mantiene la flexibilidad de permitir entrada manual cuando sea necesario.
