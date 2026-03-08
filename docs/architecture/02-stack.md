---
title: Technology Stack
created: 2026-03-08
updated: 2026-03-08
---

# Go-Bookmark API: Stack Tecnologico

## Lenguaje y Runtime

| Componente | Version | Notas |
|------------|---------|-------|
| **Go** | 1.24.0 | Toolchain `go1.24.2` |

El proyecto usa generics de Go (1.18+) en el middleware de validacion (`Validator[T]`).

## Framework Web

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/gin-gonic/gin` | v1.11.0 | Router HTTP, middleware chain, context management |
| `github.com/gin-contrib/cors` | v1.7.6 | Middleware CORS configurable |
| `github.com/gin-contrib/sse` | v1.1.0 | Server-sent events (dependencia de Gin) |

Gin se configura con los middlewares por defecto (`Logger`, `Recovery`) mas middlewares custom (`CORS`, `Protect`, `Validator[T]`).

## Base de Datos

| Componente | Detalle |
|------------|---------|
| **PostgreSQL** | Base de datos principal (Neon en produccion) |
| `gorm.io/gorm` | v1.31.0 - ORM con auto-migracion, soft deletes, preloading |
| `gorm.io/driver/postgres` | v1.6.0 - Driver PostgreSQL para GORM |
| `github.com/jackc/pgx/v5` | v5.7.6 - Driver SQL subyacente (via GORM) |

GORM se configura con `PrepareStmt: true` para prepared statements. Las tablas `bookmarks`, `tags` y `bookmark_tags` se auto-migran al inicio. La tabla `user` no se migra porque es gestionada externamente por Better Auth.

## Cache

| Componente | Detalle |
|------------|---------|
| **Redis** | Cache opcional (Upstash en produccion) |
| `github.com/redis/go-redis/v9` | v9.17.2 - Cliente Redis con connection pooling |

Configuracion del cliente Redis:

| Parametro | Valor |
|-----------|-------|
| `DialTimeout` | 10s |
| `ReadTimeout` | 30s |
| `WriteTimeout` | 30s |
| `PoolSize` | 10 conexiones |
| `PoolTimeout` | 30s |

Redis es opcional. Si no esta disponible o la conexion falla, la aplicacion continua operando sin cache (graceful degradation). La funcion `IsRedisAvailable()` verifica disponibilidad antes de cada operacion de cache.

## Autenticacion y Seguridad

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | Parsing y validacion de JWT |
| `github.com/MicahParks/keyfunc/v3` | v3.7.0 | Cliente JWKS con auto-refresh de claves publicas |
| `github.com/MicahParks/jwkset` | v0.11.0 | Dependencia de keyfunc para manejo de JWK sets |

Flujo de autenticacion:
1. Al iniciar, el middleware descarga las claves publicas EdDSA desde el endpoint JWKS configurado en `JWKS_URL`
2. Las claves se refrescan automaticamente cada 1 hora
3. Solo se aceptan tokens firmados con el algoritmo **EdDSA**
4. Claims requeridos: `user_id`, `email`, `name`, `session_id`

La autenticacion de rutas internas (`/internal/*`) usa comparacion constant-time de API key via el header `X-Internal-API-Key`.

## Validacion

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/go-playground/validator/v10` | v10.28.0 | Validacion de structs con tags |

El middleware generico `Validator[T]()` deserializa el body JSON, valida con struct tags, y almacena el payload validado en el contexto Gin bajo la key `"payload"`. Tags de validacion usados: `required`, `min`, `max`, `url`, `http_url`, `omitempty`, `dive`.

## Web Scraping

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/PuerkitoBio/goquery` | v1.11.0 | Parsing HTML, extraccion de metadata (titulo, descripcion, favicon) |
| `github.com/andybalholm/cascadia` | v1.3.3 | Selectores CSS (dependencia de goquery) |

El `ScraperService` hace requests HTTP con User-Agent de navegador, sigue hasta 10 redirects, y extrae metadata con la siguiente prioridad:
- **Titulo**: `og:title` > `twitter:title` > `<title>`
- **Descripcion**: `og:description` > `twitter:description` > `meta[name=description]`
- **Favicon**: `apple-touch-icon` > `icon[192x192]` > `icon[180x180]` > `icon` > `shortcut icon` > `og:image` > `/favicon.ico`

## Configuracion

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/joho/godotenv` | v1.5.1 | Carga de `.env` en desarrollo; en produccion las variables se configuran en la plataforma |

## Testing

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/stretchr/testify` | v1.11.1 | Assertions y test suites |

Comando: `go test ./...`

## Serializacion JSON

| Libreria | Version | Rol |
|----------|---------|-----|
| `github.com/bytedance/sonic` | v1.14.1 | Serializacion JSON de alto rendimiento (usado por Gin) |
| `github.com/goccy/go-json` | v0.10.5 | Alternativa JSON (dependencia de Gin) |
| `google.golang.org/protobuf` | v1.36.10 | Protocol Buffers (dependencia de Gin) |

## Herramientas de Desarrollo

| Herramienta | Archivo | Proposito |
|-------------|---------|-----------|
| **Air** | `.air.toml` | Hot reload en desarrollo |
| **Docker** | `Dockerfile`, `Dockerfile.dev` | Contenedores para produccion y desarrollo |

### Comandos de Desarrollo

```bash
# Servidor con hot reload (requiere air instalado)
air

# Servidor directo
go run cmd/api/main.go

# Build binario
go build -o bin/api cmd/api/main.go

# Tests
go test ./...

# Seed de datos (requiere usuario existente en Better Auth)
go run cmd/seed/main.go --user=<user-id>
go run cmd/seed/main.go --user=<user-id> --reset
go run cmd/seed/main.go --clear
```

## Infraestructura de Produccion

| Servicio | Plataforma | Rol |
|----------|------------|-----|
| **API** | Railway | Hosting del binario Go, auto-scaling |
| **PostgreSQL** | Neon | Base de datos relacional con connection pooling |
| **Redis** | Upstash | Cache serverless con replicacion global |
| **Auth** | Better Auth (en frontend Next.js) | Gestion de usuarios y emision de JWT |
| **SSL** | Plataforma (Railway) | Terminacion SSL en el load balancer |

### Configuracion de Despliegue

Archivo `railway.toml` define la configuracion de build y deploy en Railway.

## Variables de Entorno

| Variable | Requerida | Descripcion |
|----------|-----------|-------------|
| `DATABASE_URL` | Si | Connection string PostgreSQL |
| `PORT` | Si | Puerto del servidor HTTP |
| `JWKS_URL` | Si | URL del endpoint JWKS para validacion de JWT |
| `REDIS_URL` | No | Connection string Redis (default: `redis://localhost:6379/0`) |
| `ALLOWED_ORIGINS` | No | Origenes CORS separados por coma (default: `http://localhost:3000`) |
| `INTERNAL_API_KEY` | Si* | API key para rutas internas (*requerida si se usan rutas `/internal`) |

## Documentos Relacionados

- [01-overview.md](./01-overview.md) - Vision general del proyecto y arquitectura de capas
- [04-system-design.md](./04-system-design.md) - Decisiones de diseno, caching, seguridad
- [05-diagrams.md](./05-diagrams.md) - Diagramas de arquitectura en Mermaid
