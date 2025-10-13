package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/config"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/routes"
)

func main() {
	if err := config.Load(); err != nil {
		panic("Failed to load config" + err.Error())
	}

	if err := database.Connect(); err != nil {
		panic("Failed to connect to DB" + err.Error())
	}

	database.DB.AutoMigrate(&models.User{}, &models.Bookmarks{}, &models.Tag{}, &models.BookmarkTag{})

	router := gin.Default()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	routes.Setup(router)

	port := config.GetPort()
	fmt.Printf("Server running on :%s\n", port)
	router.Run(":" + port)
}
