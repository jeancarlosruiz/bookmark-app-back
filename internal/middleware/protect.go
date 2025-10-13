package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Protect(c *gin.Context) {

  token := c.GetHeader("Authorization")

  if token == "" {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
    c.Abort()
    return
  }

  fmt.Println("Token encontrado:", token)
  c.Next()
}

