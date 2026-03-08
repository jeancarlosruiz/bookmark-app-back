---
title: Project Overview
created: 2026-03-08
updated: 2026-03-08
---

# Go-Bookmark API: Vision General del Proyecto

## Que es Go-Bookmark

API REST para gestion de bookmarks personales. Permite a los usuarios guardar, organizar, buscar y archivar enlaces web con metadata automatica (titulo, descripcion, favicon) extraida mediante scraping. Cada bookmark puede etiquetarse con tags definidos por el usuario.

**Repositorio:** `github.com/jeancarlosruiz/bookmark-app-back`

## Objetivos del Proyecto

- Proveer un backend completo para una aplicacion de bookmarks con autenticacion delegada
- Demostrar patrones de arquitectura backend: capas, caching, autenticacion stateless
- Funcionar como API independiente consumida por un frontend Next.js (`nextjs-bookmark`)

## Funcionalidades Principales

| Funcionalidad | Descripcion |
|---------------|-------------|
| **CRUD de bookmarks** | Crear, leer, actualizar y eliminar (soft delete) bookmarks por usuario |
| **Sistema de tags** | Tags con scope por usuario, relacion N:M con bookmarks, find-or-create automatico |
| **Metadata scraping** | Extraccion automatica de titulo, descripcion y favicon desde la URL via `goquery` |
| **Archivado y pinning** | Toggle de estado archivado y fijado por bookmark |
| **Contador de visitas** | Tracking de accesos por bookmark con timestamp de ultima visita |
| **Busqueda** | Busqueda por titulo (`ILIKE`) y filtrado por tags |
| **Cache con degradacion** | Redis como cache-aside, la API continua operando si Redis no esta disponible |
| **Health check** | Endpoint `/health` publico que verifica estado de PostgreSQL y Redis |

## Estructura del Proyecto

```
go-bookmark/
  cmd/
    api/main.go              # Entry point del servidor
    seed/main.go             # Seeder de datos de prueba
  internal/
    config/                  # Carga de variables de entorno
    controllers/             # Handlers HTTP (extraen params, devuelven JSON)
    database/                # Conexion a PostgreSQL y Redis
    middleware/              # CORS, JWT/JWKS auth, validacion generica
    models/                  # Structs GORM (Bookmarks, Tag, BookmarkTag, User)
    repositories/            # Acceso a datos via GORM
    routes/                  # Registro de rutas y middlewares
    services/                # Logica de negocio, cache, scraping
    utils/                   # Utilidades (normalizacion de URL)
    validator/               # Schemas de validacion con struct tags
```

## Arquitectura de Capas

El proyecto sigue un patron de tres capas con separacion estricta de responsabilidades:

```
HTTP Request
    |
    v
┌─────────────────────────────────────────┐
│            Gin Router                   │
│  Middlewares: CORS -> Protect -> Validator │
└──────────────────┬──────────────────────┘
                   |
                   v
┌─────────────────────────────────────────┐
│            Controllers                  │
│  Extraen parametros del contexto Gin    │
│  Delegan a services, devuelven JSON     │
└──────────────────┬──────────────────────┘
                   |
                   v
┌─────────────────────────────────────────┐
│             Services                    │
│  Logica de negocio                      │
│  Invalidacion de cache despues de writes│
│  Coordinacion entre repositorios        │
└──────────────────┬──────────────────────┘
                   |
                   v
┌─────────────────────────────────────────┐
│            Repositories                 │
│  Queries GORM contra PostgreSQL         │
│  Preload de relaciones (Tags)           │
└──────────────────┬──────────────────────┘
                   |
                   v
        PostgreSQL / Redis
```

### Responsabilidades por Capa

**Controllers** (`internal/controllers/`):
- Extraen `user_id` del contexto JWT via `c.GetString("user_id")`
- Obtienen payload validado via `c.MustGet("payload")`
- Parsean query params y path params
- Devuelven respuestas JSON con codigos HTTP apropiados
- No contienen logica de negocio

**Services** (`internal/services/`):
- Contienen toda la logica de negocio
- Coordinan entre multiples repositorios (e.g., `BookmarkService` usa `TagService`)
- Ejecutan invalidacion de cache despues de operaciones de escritura
- Definen errores de dominio (`ErrBookmarkAlreadyExists`, `ErrTagNotFound`)
- `CacheService` encapsula la interaccion con Redis con degradacion graceful
- `ScraperService` extrae metadata de URLs externas

**Repositories** (`internal/repositories/`):
- Acceso exclusivo a la base de datos via GORM
- Usan `Preload("Tags")` en todas las consultas de bookmarks
- No contienen logica de negocio ni interactuan con cache
- Reciben referencia a `database.DB` en el constructor

### Flujo de un Request

1. El request llega al router Gin
2. Middleware `CORS` aplica headers de cross-origin
3. Middleware `Protect` valida el JWT contra las claves publicas JWKS y extrae claims (`user_id`, `email`, `name`) al contexto
4. Middleware `Validator[T]` (si aplica) deserializa el body, valida con struct tags, y almacena el payload en contexto bajo la key `"payload"`
5. El controller extrae datos del contexto y delega al service
6. El service ejecuta logica de negocio, consulta cache, e invoca repositorios
7. El repository ejecuta la query GORM y retorna el resultado
8. El controller serializa la respuesta como JSON

## Modelo de Autenticacion

La API no gestiona usuarios directamente. Los usuarios se crean y autentican en el frontend via Better Auth. La API solo verifica tokens JWT:

- El frontend obtiene un JWT firmado con EdDSA desde Better Auth
- La API descarga las claves publicas desde el endpoint JWKS (`/api/auth/jwks`) y las refresca cada hora
- Cada request autenticado incluye `Authorization: Bearer <token>`
- El middleware `Protect` verifica la firma, extrae claims, y los inyecta en el contexto Gin

La tabla `user` en PostgreSQL es de solo lectura para esta API. Existe en el schema de Better Auth y se referencia via FK desde `bookmarks` y `tags`.

## Grupos de Rutas

| Prefijo | Autenticacion | Proposito |
|---------|---------------|-----------|
| `/health` | Ninguna | Health check publico |
| `/api/*` | JWT via `Protect` | Endpoints de la API (bookmarks, tags, preview, search) |
| `/internal/*` | API Key via `InternalAPIAUTH` + JWT | Operaciones internas (migracion de datos) |

## Modelo de Datos (Resumen)

- **User**: tabla externa (Better Auth), solo se referencia `id` como FK
- **Bookmarks**: entidad principal, scoped por `user_id`, soft delete via `DeletedAt`
- **Tag**: etiquetas con scope por usuario, relacion N:M con bookmarks
- **BookmarkTag**: tabla de union para la relacion muchos a muchos

Indice unico compuesto `idx_user_url` sobre `(user_id, url)` con condicion `WHERE deleted_at IS NULL` para prevenir duplicados semanticos por usuario.

## Documentos Relacionados

- [02-stack.md](./02-stack.md) - Stack tecnologico y dependencias
- [04-system-design.md](./04-system-design.md) - Diseno del sistema, caching, seguridad, escalabilidad
- [05-diagrams.md](./05-diagrams.md) - Diagramas de arquitectura en Mermaid
