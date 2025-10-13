package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/handlers"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
)

func Setup(router *gin.Engine) {
	api := router.Group("/api")

  // Agrupar todas las rutas con el middleware deseado
	{
		// Users
		api.GET("/users", middleware.Protect, handlers.GetUsers)
		api.POST("/users", handlers.CreateUser)

		//Bookmarks
		api.GET("/bookmark", handlers.GetBookmarks)
		api.POST("/bookmark", handlers.CreateBookmark)

	}
}

