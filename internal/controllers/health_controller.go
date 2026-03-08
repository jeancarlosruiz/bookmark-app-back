package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
)

var starTime = time.Now()

type checkResult struct {
	Status    string  `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	Error     *string `json:"error"`
}

type healthResponse struct {
	Status        string                 `json:"status"`
	Timestamp     string                 `json:"timestamp"`
	Version       string                 `json:"version"`
	UptimeSeconds float64                `json:"uptime_seconds"`
	Checks        map[string]checkResult `json:"checks"`
}

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (ctrl *HealthController) HealthCheck(c *gin.Context) {
	checks := make(map[string]checkResult)

	// Verificar cada dependencia
	checks["database"] = ctrl.checkDatabase()
	checks["redis"] = ctrl.checkRedis()

	status := "healthy"
	httpCode := http.StatusOK

	if checks["database"].Status == "down" {
		status = "unhealthy"
		httpCode = http.StatusServiceUnavailable // 503
	} else if checks["redis"].Status == "down" {
		status = "degraded"
	}

	c.Header("Cache-Control", "no-cache, no-store")

	c.JSON(httpCode, healthResponse{
		Status:        status,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Version:       "1.0.0",
		UptimeSeconds: time.Since(starTime).Seconds(),
		Checks:        checks,
	})
}

func (ctrl *HealthController) checkDatabase() checkResult {
	// Timeout de 3s para que un check lento no bloquee el endopoint
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	// database.DB es *gorm.DB. Necesitamos la conexion SQL
	// subyacente para hacer un ping de bajo nivel.
	sqlDB, err := database.DB.DB()
	if err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	latency := time.Since(start).Milliseconds()

	return checkResult{Status: "up", LatencyMs: latency, Error: nil}
}

func (ctrl *HealthController) checkRedis() checkResult {
	if database.RedisClient == nil {
		return checkResult{Status: "not_configured", LatencyMs: 0, Error: nil}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	// Ping es el comando mas ligero de Redis.
	// Retorna "PONG" si la conexion esta viva.
	if err := database.RedisClient.Ping(ctx).Err(); err != nil {
		errMsg := err.Error()
		return checkResult{Status: "down", LatencyMs: 0, Error: &errMsg}
	}

	latency := time.Since(start).Milliseconds()
	return checkResult{Status: "up", LatencyMs: latency, Error: nil}
}
