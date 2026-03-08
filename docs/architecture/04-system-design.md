---
title: System Design
created: 2025-01-15
updated: 2026-03-08
---

# Backend System Design: Del Curso a la Implementacion

> **Proyecto:** Go-Bookmark API
> **Stack:** Go 1.24 | Gin | GORM | PostgreSQL | Redis
> **Autor:** Jean Ruiz

Este documento demuestra como aplique los conceptos de **Backend System Design** en un proyecto real, conectando la teoria con decisiones de arquitectura concretas.

---

## Tabla de Contenidos

1. [Arquitectura del Sistema](#1-arquitectura-del-sistema)
2. [Protocolo y API Design](#2-protocolo-y-api-design)
3. [CAP Theorem en la Practica](#3-cap-theorem-en-la-practica)
4. [Estrategia de Caching](#4-estrategia-de-caching)
5. [Seguridad y Autenticacion](#5-seguridad-y-autenticacion)
6. [Escalabilidad](#6-escalabilidad)
7. [Data Storage](#7-data-storage)
8. [Decisiones de Diseno](#8-decisiones-de-diseno)

---

## 1. Arquitectura del Sistema

### Concepto del Curso

> *"Un sistema es una coleccion de componentes que trabajan juntos, tiene entradas/salidas y tiene limites definidos."*

### Implementacion

Aplique una **arquitectura de 3 capas** con separacion estricta de responsabilidades:

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT                                │
│                    (Next.js Frontend)                        │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP/REST
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     LOAD BALANCER                            │
│                   (Platform Level)                           │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      GIN ROUTER                              │
│              ┌──────────────────────┐                        │
│              │    MIDDLEWARES       │                        │
│              │  - CORS              │                        │
│              │  - JWT Auth          │                        │
│              │  - Validation        │                        │
│              └──────────────────────┘                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
     ┌─────────────────────┼─────────────────────┐
     │                     │                     │
     ▼                     ▼                     ▼
┌─────────┐         ┌─────────┐          ┌─────────┐
│Controller│        │Controller│         │Controller│
│Bookmark │         │  Tag    │          │  User   │
└────┬────┘         └────┬────┘          └────┬────┘
     │                   │                    │
     ▼                   ▼                    ▼
┌─────────┐         ┌─────────┐          ┌─────────┐
│ Service │         │ Service │          │ Service │
│Bookmark │◄───────►│   Tag   │          │  Cache  │
└────┬────┘         └────┬────┘          └────┬────┘
     │                   │                    │
     └───────────────────┼────────────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │   REPOSITORIES   │
              │  (Data Access)   │
              └────────┬─────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │PostgreSQL│  │  Redis   │  │ External │
   │ (GORM)   │  │ (Cache)  │  │   APIs   │
   └──────────┘  └──────────┘  └──────────┘
```

### Responsabilidades por Capa

| Capa | Responsabilidad | Archivo |
|------|-----------------|---------|
| **Controller** | Extraer parametros HTTP, devolver JSON | `internal/controllers/` |
| **Service** | Logica de negocio, cache invalidation | `internal/services/` |
| **Repository** | Acceso directo a la base de datos | `internal/repositories/` |

### Ejemplo de Flujo: Crear Bookmark

```go
// 1. Controller: Extrae datos validados del contexto
func (bc *BookmarkController) CreateBookmark(c *gin.Context) {
    payload := c.MustGet("payload").(validator.CreateBookmark)
    userID := c.GetString("user_id")

    bookmark, err := bc.bookmarkService.CreateBookmarkService(payload, userID)
    // ...
}

// 2. Service: Logica de negocio + cache invalidation
func (s *BookmarkService) CreateBookmarkService(...) (*models.Bookmark, error) {
    // Normalizar URL para evitar duplicados
    normalizedURL := utils.NormalizeURL(data.Url)

    // Verificar si ya existe
    existing, _ := s.bookmarkRepo.FindByNormalizedURL(userID, normalizedURL)
    if existing != nil {
        return nil, ErrBookmarkAlreadyExists
    }

    // Crear o reusar tags
    tags, _ := s.tagService.FindOrCreateTags(data.Tags, userID)

    // Persistir
    bookmark, _ := s.bookmarkRepo.Create(...)

    // Invalidar cache (CRITICO)
    s.cacheService.InvalidateUserCache(ctx, userID)

    return bookmark, nil
}

// 3. Repository: Solo acceso a datos
func (r *BookmarkRepository) Create(bookmark *models.Bookmark) error {
    return database.DB.Create(bookmark).Error
}
```

**Por que esta arquitectura?**
- Cada capa puede testearse de forma aislada
- Los cambios en la base de datos no afectan los controllers
- Facilita el mantenimiento y escalabilidad

---

## 2. Protocolo y API Design

### Concepto del Curso

> *REST: Multiple endpoints, human readable, stateless. Ideal para CRUD apps y datos facilmente cacheables.*

### Implementacion

Elegi **REST** porque:
- El dominio es CRUD de bookmarks (crear, leer, actualizar, eliminar)
- Los datos son facilmente cacheables por usuario
- Es stateless (compatible con escalamiento horizontal)

### API Design

```
GET    /api/bookmarks              → Lista paginada con filtros
GET    /api/bookmarks/:id          → Detalle de un bookmark
POST   /api/bookmark               → Crear bookmark
PUT    /api/bookmark/:id           → Actualizar bookmark
DELETE /api/bookmark/:id           → Eliminar (soft delete)

POST   /api/bookmark/:id/visit     → Incrementar contador
POST   /api/bookmark/:id/archive   → Toggle archivado
POST   /api/bookmark/:id/pin       → Toggle fijado

GET    /api/tags                   → Lista de tags del usuario
POST   /api/tag                    → Crear tag
PUT    /api/tag/:id                → Actualizar tag
DELETE /api/tag/:id                → Eliminar tag

GET    /api/bookmark/preview       → Scraping de metadata (URL)
GET    /api/bookmark/search        → Busqueda full-text
```

### Entity Modeling

```
┌─────────────────────────────────────────────────────────────┐
│                         User                                 │
│  (External: neon_auth.users_sync - NO migrated by GORM)     │
│  id | email | name | created_at                             │
└─────────────────────────┬───────────────────────────────────┘
                          │ 1:N
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                       Bookmark                               │
│  id | user_id | title | url | normalized_url | description  │
│  favicon | is_archived | is_pinned | visit_count            │
│  created_at | updated_at | deleted_at (soft delete)         │
└─────────────────────────┬───────────────────────────────────┘
                          │ N:M
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                         Tag                                  │
│  id | user_id | name | created_at                           │
│  (User-scoped: cada usuario tiene su propio namespace)      │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. CAP Theorem en la Practica

### Concepto del Curso

> *CAP: Consistency, Availability, Partition tolerance. Solo puedes garantizar 2 de 3.*

### Decision: CP (Consistency + Partition Tolerance)

Para una aplicacion de bookmarks personales, priorice:

| Propiedad | Eleccion | Razon |
|-----------|----------|-------|
| **Consistency** | SI | Un usuario debe ver siempre sus bookmarks actualizados |
| **Partition Tolerance** | SI | El sistema debe manejar fallos de red |
| **Availability** | Parcial | Preferimos datos correctos sobre datos desactualizados |

### Implementacion

```go
// PostgreSQL garantiza ACID (Consistency)
func (r *BookmarkRepository) Create(bookmark *models.Bookmark) error {
    return database.DB.Create(bookmark).Error  // Transaccion atomica
}

// Redis es cache, NO fuente de verdad
func (s *BookmarkService) GetBookmarks(...) {
    // 1. Intentar cache
    cached := s.cacheService.GetBookmarks(userID)
    if cached != nil {
        return cached  // Hit: ~5ms
    }

    // 2. Cache miss: consultar DB (fuente de verdad)
    bookmarks := s.bookmarkRepo.FindByUserID(userID)  // ~150ms

    // 3. Poblar cache para proximas consultas
    s.cacheService.SetBookmarks(userID, bookmarks)

    return bookmarks
}
```

---

## 4. Estrategia de Caching

### Concepto del Curso

> *Cache-Aside (Lazy Loading): cache miss → read from database → update cache*

### Implementacion: Cache-Aside con Graceful Degradation

```go
// internal/services/cache_service.go

type CacheService struct {
    // Verifica si Redis esta disponible antes de cada operacion
}

func (s *CacheService) GetBookmarks(userID string) ([]models.Bookmark, error) {
    // Graceful degradation: si Redis no esta disponible, retorna nil
    if !s.IsRedisAvailable() {
        return nil, nil  // La app continua funcionando sin cache
    }

    key := fmt.Sprintf("bookmarks:user:%s", userID)
    data, err := database.RedisClient.Get(ctx, key).Bytes()
    // ...
}
```

### TTL Strategy

| Recurso | TTL | Razon |
|---------|-----|-------|
| Bookmarks | 10 min | Balance entre frescura y performance |
| Archived | 10 min | Mismo razonamiento |
| Tags | 15 min | Cambian menos frecuentemente |
| Metadata preview | 24 hrs | Contenido externo raramente cambia |

### Cache Invalidation

```go
// REGLA: Toda operacion de escritura DEBE invalidar cache

func (s *BookmarkService) CreateBookmarkService(...) {
    // ... crear bookmark ...

    // CRITICO: Invalidar todas las caches relacionadas
    s.cacheService.InvalidateUserCache(ctx, userID)
    // Invalida: bookmarks, archived, tags (todo del usuario)
}

// EXCEPCION: IncrementVisitCount NO invalida cache
// Razon: Alta frecuencia, bajo impacto en datos mostrados
func (s *BookmarkService) IncrementVisitCount(bookmarkID string) {
    s.bookmarkRepo.IncrementVisitCount(bookmarkID)
    // NO llamamos InvalidateUserCache aqui
}
```

### Cache Keys (User-Scoped)

```go
// Patron: prefijo:scope:identificador
const (
    BookmarksCacheKey     = "bookmarks:user:%s"          // Activos
    ArchivedCacheKey      = "bookmarks:user:%s:archived" // Archivados
    TagsCacheKey          = "tags:user:%s"               // Tags
    MetadataCacheKey      = "metadata:%s"                // SHA256(URL)
)
```

**Por que user-scoped?** Previene data leakage entre usuarios (seguridad multi-tenant).

---

## 5. Seguridad y Autenticacion

### Concepto del Curso

> *Authentication: verifica identidad. Stateless (token-based) hace el escalamiento mas facil.*

### Implementacion: JWKS-Based JWT (EdDSA)

```go
// internal/middleware/protect.go

// JWKS: JSON Web Key Set - el servidor obtiene las claves publicas
// de un endpoint externo y las refresca automaticamente
var jwks keyfunc.Keyfunc

func init() {
    jwks, _ = keyfunc.NewDefaultCtx(ctx, []string{config.GetJWKSURL()})
    // Refresh automatico cada 1 hora
}

func Protect() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extraer token del header
        authHeader := c.GetHeader("Authorization")
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // 2. Validar con clave publica del JWKS
        token, err := jwt.ParseWithClaims(tokenString, &BetterAuthClaims{},
            jwks.Keyfunc,
            jwt.WithValidMethods([]string{"EdDSA"}),  // Solo EdDSA
        )

        // 3. Inyectar claims en contexto
        claims := token.Claims.(*BetterAuthClaims)
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)

        c.Next()
    }
}
```

### Por que JWKS + EdDSA?

| Aspecto | Beneficio |
|---------|-----------|
| **JWKS** | No necesito almacenar secretos en el backend |
| **EdDSA** | Mas seguro y rapido que RSA |
| **Auto-refresh** | Rotacion de claves transparente |
| **Stateless** | Cada request es independiente (escalable) |

### SSL/TLS Termination

Siguiendo el curso:
> *Termination at load balancer: most common approach*

El SSL termina en el load balancer de la plataforma (Railway/Vercel), y la comunicacion interna es HTTP.

---

## 6. Escalabilidad

### Concepto del Curso

> *Horizontal scaling: mas maquinas. Facilmente escala con trafico, alta disponibilidad.*

### Preparacion para Escalar

El sistema esta disenado para escalar horizontalmente:

```
                    ┌─────────────┐
                    │Load Balancer│
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
    │ API #1  │       │ API #2  │       │ API #3  │
    └────┬────┘       └────┬────┘       └────┬────┘
         │                 │                 │
         └─────────────────┼─────────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
         │PostgreSQL│  │  Redis  │  │ Redis   │
         │ (Primary)│  │(Primary)│  │(Replica)│
         └──────────┘  └─────────┘  └─────────┘
```

### Patrones que Permiten Escalar

| Patron | Implementacion | Beneficio |
|--------|---------------|-----------|
| **Stateless Auth** | JWT en cada request | Cualquier instancia puede atender |
| **Shared Cache** | Redis centralizado | Estado compartido entre instancias |
| **Connection Pooling** | GORM pool | Limite de conexiones a la DB |
| **Graceful Degradation** | Redis opcional | Resiliencia ante fallos |

---

## 7. Data Storage

### Concepto del Curso

> *SQL: Structured data with relationships, enforced schema, ACID transactions.*

### Eleccion: PostgreSQL

| Requerimiento | PostgreSQL Feature |
|---------------|-------------------|
| Relaciones User → Bookmarks → Tags | Foreign Keys + JOIN |
| Busqueda de texto | Full-text search (tsvector) |
| Integridad de datos | ACID + Constraints |
| Soft deletes | `deleted_at IS NULL` filter |

### Cross-Schema Design

```go
// internal/database/database.go

func Connect() {
    db, _ := gorm.Open(postgres.Open(config.GetDatabaseURL()))

    // CRITICO: Permite referencias entre schemas
    db.Exec("SET search_path TO public, neon_auth")

    // public.bookmarks tiene FK a neon_auth.users_sync
    // Sin el search_path, las FK constraints fallan
}
```

### Indices Optimizados

```go
// internal/models/bookmark.go

type Bookmark struct {
    // Index compuesto para buscar por usuario y detectar duplicados
    NormalizedURL string `gorm:"index:idx_user_url,unique"`
    UserID        string `gorm:"index:idx_user_url"`

    // Index simple para ordenamiento
    CreatedAt     time.Time `gorm:"index"`
}
```

---

## 8. Decisiones de Diseno

### Trade-offs Conscientes

| Decision | Trade-off | Justificacion |
|----------|-----------|---------------|
| **Cache invalidation agresiva** | Mas writes a Redis | Garantiza consistencia de datos |
| **Visit count no invalida cache** | Datos levemente desactualizados | Alta frecuencia, bajo impacto visual |
| **Soft deletes** | Mas espacio en DB | Permite recuperacion de datos |
| **User-scoped tags** | Tags duplicados entre usuarios | Privacidad y personalizacion |
| **URL normalization** | Procesamiento extra | Previene duplicados semanticos |

### Patrones Aplicados

| Patron | Ubicacion | Proposito |
|--------|-----------|-----------|
| **Repository Pattern** | `internal/repositories/` | Abstraccion de acceso a datos |
| **Service Layer** | `internal/services/` | Encapsulacion de logica de negocio |
| **Cache-Aside** | `cache_service.go` | Performance con consistencia |
| **Middleware Chain** | `routes.go` | Composicion de funcionalidades |
| **Graceful Degradation** | Redis checks | Resiliencia del sistema |
| **Soft Delete** | GORM `DeletedAt` | Recuperabilidad de datos |

---

## Metricas de Calidad del Sistema

Basado en los conceptos del curso:

| Quality Attribute | Implementacion |
|-------------------|---------------|
| **Reliability** | Graceful degradation, soft deletes, transacciones ACID |
| **Observability** | Logging en cada capa, errores descriptivos |
| **Security** | JWT EdDSA, JWKS rotation, user-scoped data |
| **Scalability** | Stateless design, connection pooling, shared cache |
| **Performance** | Redis caching (~5ms hit vs ~150ms miss) |

---

## Conclusiones

Este proyecto demuestra la aplicacion practica de:

1. **Arquitectura limpia** con separacion de responsabilidades
2. **Caching inteligente** con invalidation strategies
3. **Seguridad moderna** con JWKS y EdDSA
4. **Diseno para escalar** con patrones stateless
5. **Trade-offs conscientes** documentados y justificados

Cada decision de arquitectura esta respaldada por los principios aprendidos en el curso de Backend System Design, aplicados a un problema real: una API de gestion de bookmarks.

---

> **Repositorio:** [go-bookmark](https://github.com/username/go-bookmark)
> **Frontend:** [nextjs-bookmark](https://github.com/username/nextjs-bookmark)
> **Curso:** [Backend System Design - Frontend Masters](https://frontendmasters.com/courses/backend-system-design)
