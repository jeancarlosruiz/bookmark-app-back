package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/controllers"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/middleware"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
)

func Setup(router *gin.Engine) {
	protectedGroup := router.Group("/api")
	internal := router.Group("/internal")

	bookmarkCtrl := controllers.NewBookmarkController()
	tagCtrl := controllers.NewTagController()
	migrationController := controllers.NewMigrationController()

	protectedGroup.Use(middleware.Protect)
	internal.Use(middleware.InternalAPIAUTH())

	// Agrupar todas las rutas con el middleware deseado
	{
		//Bookmarks
		protectedGroup.GET("/bookmark/user/:user_id", bookmarkCtrl.GetBookmarkByUserID)
		protectedGroup.GET("/bookmark/tags", bookmarkCtrl.SearchBookmarkByTags)
		protectedGroup.GET("/bookmark/preview", bookmarkCtrl.PreviewMetadata)
		protectedGroup.GET("/bookmark/title", bookmarkCtrl.SearchBookmarkByTitle)
		protectedGroup.POST("/bookmark", middleware.Validator[validator.CreateBookmark](), bookmarkCtrl.CreateBookmark)
		protectedGroup.GET("/bookmark/:id", bookmarkCtrl.GetBookmarkByID)
		protectedGroup.GET("/bookmark/user/:user_id/archived", bookmarkCtrl.GetArchivedBookmarkByUserID)
		protectedGroup.PUT("/bookmark/update/:id", middleware.Validator[validator.UpdateBookmark](), bookmarkCtrl.UpdateBookmark)
		protectedGroup.PUT("/bookmark/view-count/:id", bookmarkCtrl.IncrementVisitCountController)
		protectedGroup.PUT("/bookmark/:id/toggle-pinned", bookmarkCtrl.TogglePinnedByIDController)
		protectedGroup.PUT("/bookmark/:id/toggle-is-archived", bookmarkCtrl.ToggleIsArchiveByIDController)
		protectedGroup.DELETE("/bookmark/:id", bookmarkCtrl.DeleteBookmark)

		//Tags
		protectedGroup.POST("/tags", middleware.Validator[validator.CreateTag](), tagCtrl.CreateTag)
		protectedGroup.GET("/tags/:user_id", tagCtrl.FindTagsByUserID)
		protectedGroup.PUT("/tags/:id", middleware.Validator[validator.UpdateTag](), tagCtrl.UpdateTag)
		protectedGroup.DELETE("/tags/:id", tagCtrl.DeleteTag)

		// Migration route
	}

	{
		internal.POST("/migrate", middleware.Protect, migrationController.MigrateUser)
	}

}
