package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

func GetBookmarks(c *gin.Context) {
	var bookmarks []models.Bookmarks

	database.DB.Find(&bookmarks)

	c.JSON(http.StatusOK, bookmarks)
}

func CreateBookmark(c *gin.Context) {
	var bookmark []models.Bookmarks
	if err := c.ShouldBindJSON(&bookmark); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	database.DB.Create(&bookmark)
	c.JSON(http.StatusOK, bookmark)
}

func GetBookmarkByID(c *gin.Context) {
	var bookmarks []models.Bookmarks

	result := database.DB.First(&bookmarks)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener bookmark" + result.Error.Error(),
		})

		return
	}

	if len(bookmarks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontraron bookmarks",
			"data":    []models.Bookmarks{},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmarks obtenidos exitosamente",
		"data":    bookmarks,
	})
}
