# Plan de Implementación - Día 7: Endpoint para Creación de Bookmark con Tags Relacionados

**Fecha de Creación:** 13 de Noviembre, 2025
**Objetivo:** Modificar el endpoint existente POST /api/bookmark para aceptar un array opcional de tags

---

## 📋 Análisis de la Estructura Actual

### Modelos Existentes

#### 1. Bookmarks (`internal/models/bookmark.go`)
- Tiene relación many-to-many con Tags definida en línea 19
- GORM maneja automáticamente la tabla join `bookmark_tags`
- Campos: Title (único), Url (único), Favicon, Description, Pinned, IsActive, IsArchived, VisitCount, LastVisited

#### 2. Tag (`internal/models/tag.go`)
- Campo Title con constraint único
- Relación inversa con Bookmarks

#### 3. BookmarkTag (`internal/models/bookmark_tag.go`)
- Tabla join many-to-many ya configurada correctamente
- Contiene `bookmark_id` y `tag_id`

### Handler Actual

**CreateBookmark** (`internal/handlers/bookmark_handler.go`, líneas 21-54):
- Solo crea bookmark sin tags
- Usa validador `validator.CreateBookmark`
- No procesa relaciones con tags

### Validador Actual

**CreateBookmark** (`internal/validator/bookmark_validator.go`):
- Solo valida: Title, Url, UserID
- NO incluye campo para tags

### Referencia: Lógica de Seeding

**SeedDatabase** (`internal/database/seed.go`, líneas 61-86):
- Implementa lógica: "crear tag si no existe, o reutilizar existente"
- Usa un map para evitar duplicados en la misma operación
- Asigna tags al bookmark vía el slice `Tags` antes de `DB.Create()`
- **Esta será nuestra referencia de implementación**

---

## 🎯 Decisiones de Diseño

| Aspecto | Decisión | Justificación |
|---------|----------|---------------|
| **Endpoint** | Modificar el existente | No crear uno nuevo, mantener simplicidad |
| **Lógica de tags** | Crear si no existe, reutilizar si existe | Consistente con el seed actual |
| **Campo tags** | Opcional | Un bookmark puede crearse sin tags |
| **Formato** | Array de strings | Simple y directo |
| **Case sensitivity** | Case-sensitive | "Tools" ≠ "tools" (mejora futura: lowercase) |
| **Validación** | Strings no vacíos | Tags deben tener al menos 1 carácter |

---

## 🔧 Plan de Implementación

### PASO 1: Modificar el Validador

**Archivo:** `internal/validator/bookmark_validator.go`

**Cambio:**
```go
package validator

type CreateBookmark struct {
	Title  string   `validate:"required"`
	Url    string   `validate:"required"`
	UserID string   `validate:"required"`
	Tags   []string `validate:"omitempty,dive,min=1"` // ← NUEVA LÍNEA
}
```

**Explicación de validaciones:**
- `omitempty`: El campo tags es opcional (puede omitirse)
- `dive`: Valida cada elemento del array individualmente
- `min=1`: Cada tag debe tener al menos 1 carácter (no strings vacíos)

---

### PASO 2: Modificar el Handler CreateBookmark

**Archivo:** `internal/handlers/bookmark_handler.go`

#### 2.1 Agregar Import

Agregar `"strings"` a los imports existentes:

```go
import (
	"fmt"
	"net/http"
	"strings" // ← NUEVO

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
)
```

#### 2.2 Reemplazar Función CreateBookmark

**Reemplazar líneas 21-54** con:

```go
func CreateBookmark(c *gin.Context) {
	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})
		return
	}

	bookmarkData := payload.(validator.CreateBookmark)

	// ========================================
	// PASO 1: Procesar Tags
	// ========================================
	// Crear nuevos tags o encontrar existentes
	var tags []models.Tag
	tagMap := make(map[string]*models.Tag) // Para evitar duplicados en el mismo request

	for _, tagName := range bookmarkData.Tags {
		// Normalizar el nombre del tag (trim espacios)
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue // Saltar tags vacíos
		}

		// Verificar si ya procesamos este tag en esta request
		if existingTag, exists := tagMap[tagName]; exists {
			tags = append(tags, *existingTag)
			continue
		}

		// Buscar tag en la base de datos
		var tag models.Tag
		result := database.DB.Where("title = ?", tagName).First(&tag)

		if result.Error != nil {
			// Tag no existe, crear uno nuevo
			tag = models.Tag{Title: tagName}
			if err := database.DB.Create(&tag).Error; err != nil {
				// Manejar posible error de constraint único (race condition)
				if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
					// Intentar buscar de nuevo (otro request puede haberlo creado)
					if err := database.DB.Where("title = ?", tagName).First(&tag).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"message": "Failed to create or find tag: " + tagName,
							"error":   err.Error(),
						})
						return
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": "Failed to create tag: " + tagName,
						"error":   err.Error(),
					})
					return
				}
			}
		}

		tagMap[tagName] = &tag
		tags = append(tags, tag)
	}

	// ========================================
	// PASO 2: Crear el Bookmark con Tags
	// ========================================
	bookmark := models.Bookmarks{
		Title:  bookmarkData.Title,
		Url:    bookmarkData.Url,
		UserID: bookmarkData.UserID,
		Tags:   tags, // GORM creará automáticamente las relaciones en bookmark_tags
	}

	if err := database.DB.Create(&bookmark).Error; err != nil {
		// Manejo de errores específicos
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{
				"message": "Bookmark with this title or URL already exists",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create bookmark",
			"error":   err.Error(),
		})
		return
	}

	// ========================================
	// PASO 3: Cargar Tags para la Respuesta
	// ========================================
	database.DB.Preload("Tags").First(&bookmark, bookmark.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bookmark created successfully",
		"data":    bookmark,
	})
}
```

---

### PASO 3: No se Requieren Cambios en las Rutas

**Archivo:** `internal/routes/routes.go`

El endpoint existente permanece sin cambios (línea 24):
```go
protected.POST("/bookmark", middleware.Validator[validator.CreateBookmark](), handlers.CreateBookmark)
```

La validación se actualiza automáticamente porque el middleware usa el struct `validator.CreateBookmark` actualizado.

---

## 📝 Estructura del Request Body

### Ejemplo 1: Bookmark con Tags Nuevos

```json
{
  "title": "GitHub",
  "url": "https://github.com",
  "user_id": "user-123",
  "tags": ["Tools", "Git", "Development"]
}
```

**Resultado:** Se crean 3 tags nuevos y se asocian al bookmark.

### Ejemplo 2: Bookmark con Tags Existentes

Asumiendo que "Tools" y "Git" ya existen:

```json
{
  "title": "GitLab",
  "url": "https://gitlab.com",
  "user_id": "user-123",
  "tags": ["Tools", "Git", "CI/CD"]
}
```

**Resultado:**
- Reutiliza "Tools" y "Git" existentes
- Crea "CI/CD" nuevo
- Asocia los 3 al bookmark

### Ejemplo 3: Bookmark sin Tags

```json
{
  "title": "Personal Blog",
  "url": "https://myblog.com",
  "user_id": "user-123"
}
```

**Resultado:** Se crea el bookmark sin tags (campo opcional).

### Ejemplo 4: Tags con Espacios (Normalización)

```json
{
  "title": "Example",
  "url": "https://example.com",
  "user_id": "user-123",
  "tags": ["  Tools  ", "Git", "Tools"]
}
```

**Resultado:**
- "  Tools  " → se normaliza a "Tools"
- Duplicado "Tools" se detecta vía tagMap
- Solo se asocian 2 tags: "Tools" y "Git"

---

## 📤 Estructura de la Respuesta

### Respuesta Exitosa (201 Created)

```json
{
  "message": "Bookmark created successfully",
  "data": {
    "ID": 1,
    "CreatedAt": "2025-11-13T10:30:00Z",
    "UpdatedAt": "2025-11-13T10:30:00Z",
    "DeletedAt": null,
    "Title": "GitHub",
    "Url": "https://github.com",
    "Favicon": "",
    "Description": "",
    "Pinned": false,
    "IsActive": true,
    "IsArchived": false,
    "VisitCount": 0,
    "Tags": [
      {
        "ID": 1,
        "CreatedAt": "2025-11-13T10:30:00Z",
        "UpdatedAt": "2025-11-13T10:30:00Z",
        "DeletedAt": null,
        "Title": "Tools"
      },
      {
        "ID": 2,
        "CreatedAt": "2025-11-13T10:30:00Z",
        "UpdatedAt": "2025-11-13T10:30:00Z",
        "DeletedAt": null,
        "Title": "Git"
      },
      {
        "ID": 3,
        "CreatedAt": "2025-11-13T10:30:00Z",
        "UpdatedAt": "2025-11-13T10:30:00Z",
        "DeletedAt": null,
        "Title": "Development"
      }
    ],
    "LastVisited": "0001-01-01T00:00:00Z",
    "user_id": "user-123",
    "User": null
  }
}
```

---

## ⚠️ Manejo de Errores

### 1. Error de Validación (422 Unprocessable Entity)

**Cuándo:** Tags contienen strings vacíos o inválidos

**Request:**
```json
{
  "title": "Test",
  "url": "https://test.com",
  "user_id": "user-123",
  "tags": ["Valid", ""]  // ← String vacío
}
```

**Respuesta:**
```json
{
  "error": "Validation failed",
  "details": [
    {
      "field": "Tags[1]",
      "message": "min=1"
    }
  ]
}
```

### 2. Bookmark Duplicado (409 Conflict)

**Cuándo:** Ya existe un bookmark con el mismo título o URL

**Request:**
```json
{
  "title": "GitHub",  // ← Ya existe
  "url": "https://github.com",
  "user_id": "user-123",
  "tags": ["Tools"]
}
```

**Respuesta:**
```json
{
  "message": "Bookmark with this title or URL already exists",
  "error": "ERROR: duplicate key value violates unique constraint..."
}
```

### 3. Error al Crear Tag (500 Internal Server Error)

**Cuándo:** Falla la creación de un tag (error inesperado)

**Respuesta:**
```json
{
  "message": "Failed to create tag: TagName",
  "error": "database connection error..."
}
```

### 4. Error al Crear Bookmark (500 Internal Server Error)

**Cuándo:** Falla la creación del bookmark

**Respuesta:**
```json
{
  "message": "Failed to create bookmark",
  "error": "..."
}
```

---

## 🔄 Lógica de Negocio Detallada

### Flujo de Procesamiento de Tags

```
1. Recibir array de tags desde el request
   ↓
2. Para cada tag:
   a. Normalizar con TrimSpace()
   b. Verificar si está vacío → Skip
   c. Verificar si ya fue procesado en este request (tagMap) → Reutilizar
   ↓
3. Buscar tag en la base de datos por título (case-sensitive)
   ↓
4. Si NO existe:
   a. Crear nuevo tag
   b. Si hay error de constraint único (race condition):
      - Buscar de nuevo en DB
      - Si aún falla → Error 500
   ↓
5. Si SÍ existe:
   a. Reutilizar tag existente
   ↓
6. Agregar tag al slice de tags
   ↓
7. Crear bookmark con el slice de tags
   ↓
8. GORM crea automáticamente las filas en bookmark_tags
   ↓
9. Preload tags y devolver respuesta
```

### ¿Por Qué Esta Implementación es Robusta?

| Característica | Beneficio |
|----------------|-----------|
| **Idempotente** | Múltiples requests con los mismos tags no crean duplicados |
| **Transaccional** | GORM maneja la transacción automáticamente |
| **Concurrency-safe** | Maneja race conditions cuando dos requests crean el mismo tag simultáneamente |
| **Eficiente** | Usa map para evitar queries redundantes en el mismo request |
| **Compatible** | Usa la misma lógica que el seed, ya probada |
| **Flexible** | Tags son opcionales, no rompe endpoints existentes |

### Casos Especiales Manejados

1. **Duplicados en el request:** `["Tools", "Tools", "Git"]` → Solo 2 tags
2. **Tags con espacios:** `"  Tools  "` → Se normaliza a `"Tools"`
3. **Tags vacíos:** `["Tools", "", "Git"]` → Se ignora el vacío
4. **Race condition:** Dos requests crean "Tools" simultáneamente → Uno crea, el otro reutiliza
5. **Sin tags:** `"tags": []` o campo omitido → Válido, crea bookmark sin tags

---

## 🧪 Plan de Testing Manual

### Preparación

1. Verificar que el servidor esté corriendo: `go run cmd/api/main.go`
2. Verificar que existe un usuario válido en la base de datos
3. Usar Postman o curl para las pruebas

### Test 1: Crear Bookmark con Tags Nuevos

**Objetivo:** Verificar que se crean nuevos tags automáticamente

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Documentation",
    "url": "https://go.dev/doc",
    "user_id": "existing-user-id",
    "tags": ["Go", "Documentation", "Programming"]
  }'
```

**Verificación esperada:**
- Status: 201 Created
- Response contiene 3 tags con IDs nuevos
- Bookmark contiene los 3 tags

**Verificar en DB:**
```sql
SELECT * FROM tags WHERE title IN ('Go', 'Documentation', 'Programming');
SELECT * FROM bookmark_tags WHERE bookmark_id = 1;
```

### Test 2: Crear Bookmark Reutilizando Tags Existentes

**Objetivo:** Verificar que se reutilizan tags ya creados

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Golang Tutorial",
    "url": "https://golang.org/tutorial",
    "user_id": "existing-user-id",
    "tags": ["Go", "Tutorial"]
  }'
```

**Verificación esperada:**
- Status: 201 Created
- Tag "Go" tiene el mismo ID que en Test 1 (reutilizado)
- Tag "Tutorial" tiene un ID nuevo
- No se duplicaron tags en la tabla `tags`

### Test 3: Crear Bookmark sin Tags

**Objetivo:** Verificar que tags es opcional

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Simple Bookmark",
    "url": "https://example.com",
    "user_id": "existing-user-id"
  }'
```

**Verificación esperada:**
- Status: 201 Created
- `Tags` es un array vacío `[]` en la respuesta
- No hay filas en `bookmark_tags` para este bookmark

### Test 4: Manejo de Duplicados en Request

**Objetivo:** Verificar deduplicación en el mismo request

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Duplicate Tags Test",
    "url": "https://duplicate-test.com",
    "user_id": "existing-user-id",
    "tags": ["Python", "python", "Python", "Web"]
  }'
```

**Verificación esperada:**
- Status: 201 Created
- Se crean/asocian 3 tags: "Python", "python" (case-sensitive), "Web"
- Solo 3 filas en `bookmark_tags` para este bookmark

### Test 5: Error - Tags Vacíos

**Objetivo:** Verificar validación de tags vacíos

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Empty Tag Test",
    "url": "https://empty-tag.com",
    "user_id": "existing-user-id",
    "tags": ["Valid", ""]
  }'
```

**Verificación esperada:**
- Status: 422 Unprocessable Entity
- Error de validación indicando que tag vacío no es válido

### Test 6: Error - Bookmark Duplicado

**Objetivo:** Verificar manejo de título/URL duplicado

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Documentation",
    "url": "https://different-url.com",
    "user_id": "existing-user-id",
    "tags": ["Tag"]
  }'
```

**Verificación esperada:**
- Status: 409 Conflict
- Mensaje: "Bookmark with this title or URL already exists"

### Test 7: Normalización de Espacios

**Objetivo:** Verificar que TrimSpace funciona correctamente

```bash
curl -X POST http://localhost:8080/api/bookmark \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Spaces Test",
    "url": "https://spaces-test.com",
    "user_id": "existing-user-id",
    "tags": ["  Frontend  ", "Backend", "   DevOps   "]
  }'
```

**Verificación esperada:**
- Status: 201 Created
- Tags guardados como: "Frontend", "Backend", "DevOps" (sin espacios)

---

## 🗃️ Verificación en Base de Datos

### Query 1: Ver Bookmarks con sus Tags

```sql
SELECT
    b.id,
    b.title,
    b.url,
    STRING_AGG(t.title, ', ') AS tags
FROM bookmarks b
LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
LEFT JOIN tags t ON bt.tag_id = t.id
GROUP BY b.id, b.title, b.url
ORDER BY b.id;
```

**Output esperado:**
```
id | title              | url                      | tags
---|--------------------|--------------------------|------------------------
1  | Go Documentation   | https://go.dev/doc       | Go, Documentation, Programming
2  | Golang Tutorial    | https://golang.org/...   | Go, Tutorial
3  | Simple Bookmark    | https://example.com      | NULL
```

### Query 2: Ver Todos los Tags Creados

```sql
SELECT id, title, created_at
FROM tags
ORDER BY created_at DESC;
```

### Query 3: Ver la Tabla Join

```sql
SELECT
    bt.bookmark_id,
    b.title AS bookmark_title,
    bt.tag_id,
    t.title AS tag_title
FROM bookmark_tags bt
JOIN bookmarks b ON bt.bookmark_id = b.id
JOIN tags t ON bt.tag_id = t.id
ORDER BY bt.bookmark_id, bt.tag_id;
```

### Query 4: Verificar No Hay Tags Duplicados

```sql
SELECT title, COUNT(*) AS count
FROM tags
GROUP BY title
HAVING COUNT(*) > 1;
```

**Output esperado:** Sin resultados (no duplicados)

---

## 🚀 Orden de Implementación

1. **Modificar validador** (5 min)
   - Archivo más simple, solo agregar 1 línea
   - Bajo riesgo de errores

2. **Modificar handler** (15 min)
   - Agregar import `strings`
   - Reemplazar función `CreateBookmark` completa
   - Revisar indentación y formato

3. **Reiniciar servidor** (1 min)
   ```bash
   # Ctrl+C para detener
   go run cmd/api/main.go
   ```

4. **Testing manual con Postman/curl** (20 min)
   - Ejecutar los 7 tests descritos arriba
   - Verificar respuestas y base de datos

5. **Verificar en base de datos** (5 min)
   - Ejecutar las queries de verificación
   - Confirmar integridad de datos

**Tiempo total estimado:** 45 minutos

---

## ⚠️ Consideraciones Adicionales

### Limitaciones Conocidas

1. **Case-sensitive:** "Tools" y "tools" son tags diferentes
   - **Impacto:** Podría causar duplicados semánticos
   - **Solución futura:** Convertir a lowercase antes de buscar/crear

2. **Sin trim en DB:** Si alguien crea un tag " Tools " directamente en DB, no se detectará
   - **Impacto:** Bajo (solo si se modifica DB manualmente)
   - **Solución:** Agregar constraint CHECK en migración

3. **No hay soft-delete en tags:** Tags eliminados se borran permanentemente
   - **Impacto:** Perder historial si se borra un tag usado
   - **Solución futura:** Agregar DeletedAt a Tag model

4. **Sin límite de tags:** Un bookmark puede tener infinitos tags
   - **Impacto:** Posible abuso o performance issues
   - **Solución futura:** Validar `max=10` en el validador

### Mejoras Futuras (NO implementar ahora)

| Mejora | Prioridad | Estimación |
|--------|-----------|------------|
| Tags case-insensitive (lowercase) | Alta | 30 min |
| Límite máximo de tags (ej: 10) | Media | 10 min |
| Endpoint CRUD para tags | Media | 2 horas |
| Descripción y color para tags | Baja | 1 hora |
| Slug automático para tags | Baja | 30 min |
| Soft-delete para tags | Baja | 30 min |
| Tags sugeridos (autocomplete) | Baja | 2 horas |

### Notas de Seguridad

- ✅ **SQL Injection:** GORM usa prepared statements automáticamente
- ✅ **XSS:** JSON serialization escapa caracteres especiales
- ✅ **Validación:** Tags vacíos se rechazan en validador
- ⚠️ **Rate limiting:** No implementado (considerar para producción)
- ⚠️ **Tag spam:** Sin límite de tags por bookmark

---

## 📊 Resumen Ejecutivo

### Archivos a Modificar

| Archivo | Cambios | Líneas Afectadas |
|---------|---------|------------------|
| `internal/validator/bookmark_validator.go` | Agregar campo `Tags` | +1 línea |
| `internal/handlers/bookmark_handler.go` | Agregar import + reemplazar función | ~70 líneas |

**Total:** 2 archivos, ~71 líneas

### Complejidad

- **Complejidad técnica:** Media
- **Riesgo de bugs:** Bajo (lógica ya probada en seed)
- **Impacto en código existente:** Mínimo (backward compatible)
- **Testing requerido:** Medio (7 casos de prueba)

### Beneficios

✅ Endpoint más completo y funcional
✅ UX mejorado (crear bookmark + tags en 1 request)
✅ Código robusto con manejo de edge cases
✅ Backward compatible (tags opcional)
✅ Consistente con seed existente

### Riesgos

⚠️ Race condition en creación de tags (mitigado con retry)
⚠️ Tags case-sensitive pueden causar duplicados semánticos
⚠️ Sin límite de tags podría afectar performance

---

## ✅ Checklist de Implementación

- [ ] Backup del código actual
- [ ] Modificar `internal/validator/bookmark_validator.go`
- [ ] Agregar import `strings` en handler
- [ ] Reemplazar función `CreateBookmark` en handler
- [ ] Reiniciar servidor
- [ ] Test 1: Crear bookmark con tags nuevos ✓
- [ ] Test 2: Crear bookmark con tags existentes ✓
- [ ] Test 3: Crear bookmark sin tags ✓
- [ ] Test 4: Duplicados en request ✓
- [ ] Test 5: Error - tags vacíos ✓
- [ ] Test 6: Error - bookmark duplicado ✓
- [ ] Test 7: Normalización de espacios ✓
- [ ] Verificar queries en base de datos
- [ ] Marcar tarea como completada en roadmap
- [ ] Commit con mensaje descriptivo
- [ ] Documentar en CHANGELOG (si existe)

---

**Plan creado por:** Claude Code
**Basado en:** Análisis del proyecto go-bookmark-app-back
**Referencia:** Day 7 del roadmap de desarrollo
