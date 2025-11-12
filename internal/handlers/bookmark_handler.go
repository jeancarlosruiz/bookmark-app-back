package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

func GetBookmarks(c *gin.Context) {
	var bookmarks []models.Bookmarks

	database.DB.Preload("User").Find(&bookmarks)

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
	bookmarkID := c.Param("id")

	fmt.Println("Bookmark ID: ", bookmarkID)

	var bookmark models.Bookmarks
	result := database.DB.Where("id = ?", bookmarkID).First(&bookmark)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener bookmark" + result.Error.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark obtenido exitosamente",
		"data":    bookmark,
	})
}

func GetBookmarkByUserID(c *gin.Context) {
	userID := c.Param("user_id")

	fmt.Println("User ID: ", userID)

	var bookmarks []models.Bookmarks
	result := database.DB.Preload("Tags").Where("user_id = ?", userID).Where("is_active = ?", true).Find(&bookmarks)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener bookmarks para este usuario: " + result.Error.Error(),
		})
		return
	}

	if len(bookmarks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontraron bookmarks para este usuario",
			"data":    []models.Bookmarks{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmarks obtenidos exitosamente",
		"data":    bookmarks,
	})
}

func DeleteBookmark(c *gin.Context) {
	bookmarkID := c.Param("id")

	var bookmark models.Bookmarks
	result := database.DB.Where("id = ?", bookmarkID).Find(&bookmark)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro al encontrar el bookmark" + result.Error.Error(),
		})

		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontro bookmark con este id" + bookmarkID,
			"data":    bookmark,
		})

		return
	}

	database.DB.Model(&bookmark).Update("is_active", false)

	c.JSON(http.StatusOK, gin.H{
		"message": "El bookmark fue eliminado correctamente " + bookmarkID,
	})

}

func UpdateBookmark(c *gin.Context) {
	bookmarkID := c.Param("id")

	var bookmark models.Bookmarks
	result := database.DB.Where("id = ?", bookmarkID).Find(&bookmark)

	fmt.Println("Bookmark id:", bookmarkID)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al encontrar el bookmark" + result.Error.Error(),
		})

		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontro bookmark con este id" + bookmarkID,
			"data":    bookmark,
		})

		return
	}

}
