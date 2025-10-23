package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/handlers"
	// "github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
)

func Setup(router *gin.Engine) {
	protected := router.Group("/api")
	// Ponerlo al final
	// protected.Use(middleware.Protect)

	// Agrupar todas las rutas con el middleware deseado
	{
		// Users
		protected.GET("/users", handlers.GetUsers)

		//Bookmarks
		protected.GET("/bookmark", handlers.GetBookmarks)
		protected.GET("/bookmark/:id", handlers.GetBookmarkByID)
		protected.POST("/bookmark", handlers.CreateBookmark)
		protected.GET("/bookmark/user/:user_id", handlers.GetBookmarkByUserID)

	}
}
