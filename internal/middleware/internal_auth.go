package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func InternalAPIAUTH() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Extract client the api Key
		clientKey := c.GetHeader("X-Internal-API-Key")

		// Extract api key
		expectedKey := os.Getenv("INTERNAL_API_KEY")

		if expectedKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal API key not configured",
			})

			c.Abort()
			return
		}

		if subtle.ConstantTimeCompare([]byte(clientKey), []byte(expectedKey)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing internal API key",
			})

			c.Abort()
			return
		}

		c.Next()
	}
}
