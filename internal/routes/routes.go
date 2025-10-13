package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/handlers"
)

func Setup(router *gin.Engine) {
	api := router.Group("/api")

	{
		// Users
		api.GET("/users", handlers.GetUsers)
		api.POST("/users", handlers.CreateUser)

		//Bookmarks
		api.GET("/bookmark", handlers.GetBookmarks)
		api.POST("/bookmark", handlers.CreateBookmark)

	}
}
