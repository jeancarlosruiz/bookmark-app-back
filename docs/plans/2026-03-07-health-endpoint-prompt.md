---
title: Health Endpoint Implementation Prompt
created: 2026-03-07
updated: 2026-03-08
---

# Prompt — Implementar Health Endpoint en go-bookmark

Copia y pega este prompt en Claude Code estando en el directorio `/Users/jeanruiz/Desktop/projects/go-bookmark`:

---

## Prompt

Implementar un **health check endpoint** en esta API de Go (Gin + GORM + Redis). El endpoint sera consumido por una plataforma externa de monitoreo (PulseGuard) que hace requests cada 30 segundos para verificar el estado del servicio.

### Requisitos

#### 1. Endpoint `GET /health` — Publico, sin autenticacion

Registrar la ruta **fuera** de los grupos protegidos en `internal/routes/routes.go`. NO debe pasar por `middleware.Protect` ni por `middleware.InternalAPIAUTH()`. Debe estar al mismo nivel que los middleware globales (Logger, Recovery, CORS).

#### 2. Crear `internal/controllers/health_controller.go`

El controller debe verificar cada dependencia y retornar un JSON estructurado:

```json
{
  "status": "healthy | degraded | unhealthy",
  "timestamp": "2026-03-07T14:23:00Z",
  "version": "1.0.0",
  "uptime_seconds": 12345,
  "checks": {
    "database": {
      "status": "up | down",
      "latency_ms": 2,
      "error": null
    },
    "redis": {
      "status": "up | down | not_configured",
      "latency_ms": 1,
      "error": null
    }
  }
}
```

#### 3. Logica de status

- **healthy** (HTTP 200): Todas las dependencias estan UP
- **degraded** (HTTP 200): Redis esta DOWN pero DB esta UP (el servicio funciona sin cache, no es critico)
- **unhealthy** (HTTP 503): La base de datos esta DOWN (el servicio no puede operar)

#### 4. Verificacion de dependencias

**PostgreSQL** — Usar `database.DB` (variable global en `internal/database/database.go`). Obtener la conexion SQL subyacente con `DB.DB()` y llamar a `sqlDB.PingContext(ctx)` con un timeout de 3 segundos. Medir latencia en milisegundos.

**Redis** — Usar `database.RedisClient` (variable global en `internal/database/redis.go`). Ya existe la funcion `database.IsRedisAvailable()` que retorna un bool — usarla como base pero ademas medir latencia con `RedisClient.Ping(ctx)`. Si `RedisClient` es nil, reportar status como `"not_configured"`.

#### 5. Calcular uptime

Guardar el timestamp de inicio del servidor. Opciones:

- Crear una variable en el health controller que se inicializa al cargar el package
- O recibir el start time desde `main.go` al registrar las rutas

Calcular `uptime_seconds` como `time.Since(startTime).Seconds()`.

#### 6. Estructura de archivos

```
internal/
├── controllers/
│   ├── bookmark_controller.go  (existente)
│   ├── tag_controller.go       (existente)
│   └── health_controller.go    (NUEVO)
└── routes/
    └── routes.go               (MODIFICAR — agregar ruta /health)
```

### Contexto del codebase

- **Framework:** Gin (`github.com/gin-gonic/gin`)
- **ORM:** GORM con driver PostgreSQL (`gorm.io/gorm`)
- **Redis:** go-redis v9 (`github.com/redis/go-redis/v9`)
- **DB global:** `database.DB` tipo `*gorm.DB` (se inicializa en `database.Connect()`)
- **Redis global:** `database.RedisClient` tipo `*redis.Client` (se inicializa en `database.ConnectRedis()`)
- **Patron:** Controller → Service → Repository (pero el health controller NO necesita service ni repository — accede directo a las conexiones de DB)
- **Auth:** JWT via JWKS (Better Auth) en `middleware.Protect` — el health endpoint NO usa esto
- **Rutas existentes:** `/api/*` (protegidas), `/internal/*` (API key). El health va en la raiz: `/health`

### Consideraciones

- El health endpoint debe ser **rapido** (< 100ms). No hacer queries complejas — solo ping a las conexiones.
- Usar `context.WithTimeout` de 3 segundos para cada check para evitar que un health check lento bloquee el worker.
- NO loggear cada health check request (se ejecuta cada 30s, llenaria los logs). Considerar usar `gin.LoggerWithConfig` con un skip path, o simplemente no agregar logging extra.
- El response debe incluir headers `Cache-Control: no-cache, no-store` para que ningun proxy cachee el resultado.
- Seguir el patron existente del codebase: archivo en `controllers/`, registro en `routes/routes.go`, naming consistente.

### Ejemplo de uso esperado

```bash
# Servicio saludable
$ curl http://localhost:8080/health
HTTP/1.1 200 OK
{
  "status": "healthy",
  "timestamp": "2026-03-07T14:23:00Z",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "checks": {
    "database": { "status": "up", "latency_ms": 2, "error": null },
    "redis": { "status": "up", "latency_ms": 1, "error": null }
  }
}

# Redis caido (servicio degradado pero funcional)
$ curl http://localhost:8080/health
HTTP/1.1 200 OK
{
  "status": "degraded",
  "timestamp": "2026-03-07T14:23:00Z",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "checks": {
    "database": { "status": "up", "latency_ms": 2, "error": null },
    "redis": { "status": "down", "latency_ms": 0, "error": "connection refused" }
  }
}

# DB caida (servicio no operacional)
$ curl http://localhost:8080/health
HTTP/1.1 503 Service Unavailable
{
  "status": "unhealthy",
  "timestamp": "2026-03-07T14:23:00Z",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "checks": {
    "database": { "status": "down", "latency_ms": 0, "error": "connection refused" },
    "redis": { "status": "up", "latency_ms": 1, "error": null }
  }
}
```
