package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwks       keyfunc.Keyfunc
	jwksURL    string = getEnv("JWKS_URL", "http://localhost:3000/api/auth/jwks")
	cancelFunc context.CancelFunc
)

// Inicializar JWKS al arrancar la aplicación
func init() {
	var err error

	// Crear contexto con capacidad de cancelación para controlar el background refresh
	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel

	// Crear cliente JWKS con auto-refresh usando Override para configurar opciones
	jwks, err = keyfunc.NewDefaultOverrideCtx(
		ctx,
		[]string{jwksURL},
		keyfunc.Override{
			RefreshInterval: 1 * time.Hour,    // Refrescar claves cada hora
			HTTPTimeout:     10 * time.Second, // Timeout de red
		},
	)

	if err != nil {
		log.Fatalf("❌ Error al inicializar JWKS desde %s: %v", jwksURL, err)
	}

	log.Printf("✅ JWKS inicializado correctamente desde %s", jwksURL)
}

// Claims personalizados que coinciden con el payload de Better Auth
type BetterAuthClaims struct {
	UserID    string `json:"user_id"` // Better Auth usa user_id, no sub
	Email     string `json:"email"`
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func Protect(c *gin.Context) {
	// 1. Extraer token del header Authorization
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Missing Authorization header",
		})
		c.Abort()
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid Authorization format (expected: Bearer <token>)",
		})
		c.Abort()
		return
	}

	// 2. Parsear y verificar JWT con JWKS (clave pública)
	token, err := jwt.ParseWithClaims(
		tokenString,
		&BetterAuthClaims{},
		jwks.Keyfunc,                            // Usa la clave pública obtenida del JWKS endpoint
		jwt.WithValidMethods([]string{"EdDSA"}), // Solo permitir EdDSA
		jwt.WithExpirationRequired(),            // Requiere campo exp
	)

	if err != nil {
		log.Printf("❌ Error al validar JWT: %v", err)

		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Tu sesión ha expirado. Vuelve a iniciar sesión.",
				"code":  "TOKEN_EXPIRED",
			})
		case errors.Is(err, jwt.ErrTokenMalformed),
			errors.Is(err, jwt.ErrTokenSignatureInvalid),
			errors.Is(err, jwt.ErrTokenNotValidYet),
			errors.Is(err, jwt.ErrTokenInvalidClaims):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Tu sesión no es válida. Vuelve a iniciar sesión.",
				"code":  "TOKEN_INVALID",
			})
		default:
			// Errores de infraestructura: JWKS no alcanzable, key no encontrada, etc.
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "No pudimos verificar tu sesión. Intenta de nuevo en unos minutos.",
				"code":  "TOKEN_VERIFICATION_FAILED",
			})
		}
		c.Abort()
		return
	}

	if !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
		})
		c.Abort()
		return
	}

	// 3. Extraer claims personalizados
	claims, ok := token.Claims.(*BetterAuthClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token claims",
		})
		c.Abort()
		return
	}

	// 4. Agregar información del usuario al contexto de Gin para handlers posteriores
	c.Set("user_id", claims.UserID)
	c.Set("email", claims.Email)
	c.Set("name", claims.Name)

	log.Printf("✅ Usuario autenticado: %s (%s)", claims.Name, claims.UserID)

	// 5. Continuar con el siguiente handler
	c.Next()
}

// Limpieza al cerrar la aplicación (opcional, llamar en shutdown)
func CleanupJWKS() {
	if cancelFunc != nil {
		cancelFunc()
		log.Println("🛑 JWKS background refresh detenido")
	}
}

// Helper para obtener variables de entorno con fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
