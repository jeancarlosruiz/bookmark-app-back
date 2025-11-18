package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/controllers"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
)

func Setup(router *gin.Engine) {
	protected := router.Group("/api")
	bookmarkCtrl := controllers.NewBookmarkController()
	// Ponerlo al final
	protected.Use(middleware.Protect)

	// Agrupar todas las rutas con el middleware deseado
	{
		// Users
		protected.GET("/users", controllers.GetUsers)

		//Bookmarks
		// Esta ya no existe
		// protected.GET("/bookmark", bookmarkCtrl.GetBookmarks)
		protected.GET("/bookmark/:id", bookmarkCtrl.GetBookmarkByID)
		protected.GET("/bookmark/search", bookmarkCtrl.SearchBookmarkByTitle)
		protected.GET("/bookmark/tags", bookmarkCtrl.SearchBookmarkByTags)
		protected.POST("/bookmark", middleware.Validator[validator.CreateBookmark](), bookmarkCtrl.CreateBookmark)
		protected.GET("/bookmark/user/:user_id", bookmarkCtrl.GetBookmarkByUserID)
		protected.PUT("/bookmark/update/:id", bookmarkCtrl.UpdateBookmark)
		protected.DELETE("/bookmark/:id", bookmarkCtrl.DeleteBookmark)
	}
}
