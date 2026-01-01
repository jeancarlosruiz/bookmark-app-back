package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/controllers"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
)

func Setup(router *gin.Engine) {
	protectedGroup := router.Group("/api")
	bookmarkCtrl := controllers.NewBookmarkController()
	tagCtrl := controllers.NewTagController()
	// Ponerlo al final
	protectedGroup.Use(middleware.Protect)

	// Agrupar todas las rutas con el middleware deseado
	{
		// Users
		protectedGroup.GET("/users", controllers.GetUsers)

		//Bookmarks
		protectedGroup.GET("/bookmark/:id", bookmarkCtrl.GetBookmarkByID)
		protectedGroup.GET("/bookmark/title", bookmarkCtrl.SearchBookmarkByTitle)
		protectedGroup.GET("/bookmark/tags", bookmarkCtrl.SearchBookmarkByTags)
		protectedGroup.GET("/bookmark/preview", bookmarkCtrl.PreviewMetadata)
		protectedGroup.POST("/bookmark", middleware.Validator[validator.CreateBookmark](), bookmarkCtrl.CreateBookmark)
		protectedGroup.GET("/bookmark/user/:user_id", bookmarkCtrl.GetBookmarkByUserID)
		protectedGroup.GET("/bookmark/user/:user_id/archived", bookmarkCtrl.GetArchivedBookmarkByUserID)
		protectedGroup.PUT("/bookmark/update/:id", bookmarkCtrl.UpdateBookmark)
		protectedGroup.PUT("/bookmark/view-count/:id", bookmarkCtrl.IncrementVisitCountController)
		protectedGroup.PUT("/bookmark/:id/toggle-pinned", bookmarkCtrl.TogglePinnedByIDController)
		protectedGroup.PUT("/bookmark/:id/toggle-is-archived", bookmarkCtrl.ToggleIsArchiveByIDController)
		protectedGroup.DELETE("/bookmark/:id", bookmarkCtrl.DeleteBookmark)

		//Tags
		protectedGroup.GET("/tags/:user_id", tagCtrl.FindTagsByUserID)
		protectedGroup.POST("/tags", middleware.Validator[validator.CreateTag](), tagCtrl.CreateTag)
		protectedGroup.PUT("/tags/:id", middleware.Validator[validator.UpdateTag](), tagCtrl.UpdateTag)
	}
}
