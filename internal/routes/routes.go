package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/handlers"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
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
		protected.GET("/bookmark/search", handlers.SearchBookmarkByTitle)
		protected.GET("/bookmark/tags", handlers.SearchBookmarkByTags)
		protected.POST("/bookmark", middleware.Validator[validator.CreateBookmark](), handlers.CreateBookmark)
		protected.GET("/bookmark/user/:user_id", handlers.GetBookmarkByUserID)
		protected.PUT("/bookmark/update/:id", handlers.UpdateBookmark)
		protected.DELETE("/bookmark/:id", handlers.DeleteBookmark)

	}
}
