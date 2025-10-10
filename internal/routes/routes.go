package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/handlers"
)

func Setup(router *gin.Engine) {
	api := router.Group("/api")

	{
		api.GET("/users", handlers.GetUsers)
		api.POST("/users", handlers.CreateUser)
	}
}
