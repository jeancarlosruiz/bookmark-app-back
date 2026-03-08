---
title: Health Endpoint Implementation Plan
created: 2026-03-07
updated: 2026-03-08
---

# Plan: Implementar Health Endpoint paso a paso

Guia para implementar `GET /health` en tu API de Go. Cada paso explica **que hacer**, **por que**, y el **codigo exacto**.

---

## Paso 1: Crear el archivo del controller

**Archivo:** `internal/controllers/health_controller.go`

**Por que un controller y no un service/repository?**
El health endpoint solo necesita hacer ping a las conexiones existentes (DB y Redis). No hay logica de negocio ni queries complejas, asi que acceder directo a las variables globales de `database` es lo correcto. Crear un service/repository seria over-engineering.

**Que hacer:** Crea el archivo y empieza con el package, imports, y los structs del response.

```go
package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
)
```

### 1a. Variable de inicio y structs

```go
// startTime se inicializa cuando Go carga este package.
// time.Now() se ejecuta una sola vez al hacer import, no en cada request.
var startTime = time.Now()
```

**Por que `var startTime = time.Now()` a nivel de package?**
Las variables a nivel de package se inicializan una sola vez cuando el programa arranca. Esto te da el momento exacto en que el servidor inicio, sin necesidad de pasarlo desde `main.go`.

Ahora define los structs para el JSON response:

```go
// checkResult representa el estado de una dependencia individual.
// Usamos *string para Error (puntero) porque cuando no hay error
// queremos que aparezca como `null` en el JSON, no como "".
type checkResult struct {
	Status    string  `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	Error     *string `json:"error"`
}

// healthResponse es el response completo del endpoint.
type healthResponse struct {
	Status        string                 `json:"status"`
	Timestamp     string                 `json:"timestamp"`
	Version       string                 `json:"version"`
	UptimeSeconds float64                `json:"uptime_seconds"`
	Checks        map[string]checkResult `json:"checks"`
}
```

**Por que `*string` y no `string` para Error?**
En Go, el zero value de `string` es `""`. Si usas `string`, cuando no hay error el JSON mostraria `"error": ""` en vez de `"error": null`. Con un puntero `*string`, el zero value es `nil`, que se serializa como `null`. Esto es un patron comun para campos opcionales en JSON.

---

## Paso 2: Constructor y handler principal

```go
type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}
```

**Por que un struct vacio?** Para mantener consistencia con el resto del codebase (`NewBookmarkController()`, `NewTagController()`). Todos los controllers siguen este patron. Ademas, si en el futuro necesitas inyectar dependencias, ya tienes el struct listo.

Ahora el handler principal:

```go
func (ctrl *HealthController) HealthCheck(c *gin.Context) {
	checks := make(map[string]checkResult)

	// Verificar cada dependencia
	checks["database"] = ctrl.checkDatabase()
	checks["redis"] = ctrl.checkRedis()

	// Determinar status general basado en los checks
	status := "healthy"
	httpCode := http.StatusOK

	if checks["database"].Status == "down" {
		status = "unhealthy"
		httpCode = http.StatusServiceUnavailable // 503
	} else if checks["redis"].Status == "down" {
		status = "degraded"
		// Se mantiene 200 porque el servicio funciona sin cache
	}

	// Evitar que proxies/CDNs cacheen el resultado
	c.Header("Cache-Control", "no-cache, no-store")

	c.JSON(httpCode, healthResponse{
		Status:        status,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Version:       "1.0.0",
		UptimeSeconds: time.Since(startTime).Seconds(),
		Checks:        checks,
	})
}
```

**Logica de status — por que estos 3 niveles?**
- **healthy (200):** Todo funciona. PulseGuard muestra verde.
- **degraded (200):** Redis esta caido pero la API sigue funcionando (solo sin cache). Es 200 porque el servicio puede atender requests. PulseGuard puede mostrar amarillo.
- **unhealthy (503):** La DB esta caida. Sin DB no hay servicio. 503 le dice a PulseGuard que algo esta roto.

**Por que `Cache-Control: no-cache, no-store`?**
Si un proxy o CDN cachea el health response, PulseGuard podria recibir un "healthy" viejo mientras el servicio ya esta caido. Este header previene ese escenario.

---

## Paso 3: Check de PostgreSQL

```go
func (ctrl *HealthController) checkDatabase() checkResult {
	// Timeout de 3s para que un check lento no bloquee el endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	// database.DB es *gorm.DB (el ORM). Necesitamos la conexion SQL
	// subyacente para hacer un ping de bajo nivel.
	sqlDB, err := database.DB.DB()
	if err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	// PingContext verifica que la conexion esta viva.
	// Usa el context con timeout para no esperar indefinidamente.
	if err := sqlDB.PingContext(ctx); err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	latency := time.Since(start).Milliseconds()
	return checkResult{Status: "up", LatencyMs: latency, Error: nil}
}
```

**Conceptos clave:**

1. **`context.WithTimeout`** — Crea un context que se cancela automaticamente despues de 3 segundos. Si la DB tarda mas en responder, el ping falla en vez de bloquear. Siempre llama `defer cancel()` para liberar recursos.

2. **`database.DB.DB()`** — GORM envuelve la conexion SQL estandar de Go. `.DB()` te devuelve el `*sql.DB` subyacente, que tiene el metodo `Ping`. No queremos hacer un query de GORM (como `DB.Raw("SELECT 1")`), queremos el ping mas ligero posible.

3. **`&errMsg`** — Tomamos la direccion de la variable local. No podemos hacer `&err.Error()` directamente porque Go no permite tomar la direccion de un valor de retorno. Por eso primero guardamos en `errMsg`.

---

## Paso 4: Check de Redis

```go
func (ctrl *HealthController) checkRedis() checkResult {
	// Si Redis no esta configurado, no es un error — simplemente reportamos
	if database.RedisClient == nil {
		return checkResult{Status: "not_configured", LatencyMs: 0, Error: nil}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	// PING es el comando mas ligero de Redis.
	// Retorna "PONG" si la conexion esta viva.
	if err := database.RedisClient.Ping(ctx).Err(); err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	latency := time.Since(start).Milliseconds()
	return checkResult{Status: "up", LatencyMs: latency, Error: nil}
}
```

**Por que checar `RedisClient == nil`?**
En tu codebase, Redis es opcional. Si `REDIS_URL` no esta configurado, `RedisClient` queda como `nil`. Intentar llamar `.Ping()` en un nil pointer causaria un panic. El status `"not_configured"` le dice a PulseGuard que no es un error, simplemente no esta habilitado.

---

## Paso 5: Registrar la ruta

**Archivo:** `internal/routes/routes.go`

Agrega estas 3 lineas al inicio de la funcion `Setup`, **antes** de los grupos protegidos:

```go
func Setup(router *gin.Engine) {
	// Health check - publico, sin autenticacion
	healthCtrl := controllers.NewHealthController()
	router.GET("/health", healthCtrl.HealthCheck)

	protectedGroup := router.Group("/api")
	// ... resto del codigo existente
```

**Por que fuera de los grupos protegidos?**
- Las rutas en `protectedGroup` pasan por `middleware.Protect` (JWT auth).
- Las rutas en `internal` pasan por `middleware.InternalAPIAUTH()` (API key).
- El health endpoint es publico. PulseGuard no tiene credenciales de tu sistema, solo hace un `GET /health` directo. Al registrarlo con `router.GET()` directamente (sin grupo), no tiene middleware de auth.

---

## Paso 6: Verificar

```bash
# 1. Compilar — debe pasar sin errores
go build ./...

# 2. Correr el servidor
go run cmd/api/main.go

# 3. En otra terminal, probar el endpoint
curl -s http://localhost:8080/health | jq .
```

Deberias ver algo como:

```json
{
  "status": "healthy",
  "timestamp": "2026-03-07T14:23:00Z",
  "version": "1.0.0",
  "uptime_seconds": 5.2,
  "checks": {
    "database": { "status": "up", "latency_ms": 2, "error": null },
    "redis": { "status": "up", "latency_ms": 1, "error": null }
  }
}
```

---

## Resumen de archivos

| Archivo | Accion |
|---|---|
| `internal/controllers/health_controller.go` | **Crear** (Pasos 1-4) |
| `internal/routes/routes.go` | **Modificar** (Paso 5) — agregar 3 lineas |

## Conceptos que practicaste

- **Package-level variables** para estado persistente (`startTime`)
- **`context.WithTimeout`** para evitar bloqueos
- **Punteros para JSON opcionales** (`*string` → `null`)
- **GORM `.DB()`** para acceder a la conexion SQL subyacente
- **Registro de rutas** fuera de grupos para endpoints publicos
- **Health check pattern** (healthy/degraded/unhealthy) comun en microservicios
