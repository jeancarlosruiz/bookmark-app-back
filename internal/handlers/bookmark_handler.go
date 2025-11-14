package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)

func GetBookmarks(c *gin.Context) {
	var bookmarks []models.Bookmarks

	database.DB.Preload("User").Find(&bookmarks)

	c.JSON(http.StatusOK, bookmarks)
}

func CreateBookmark(c *gin.Context) {

	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})

		return
	}

	bookmarkData := payload.(validator.CreateBookmark)

	var tags []models.Tag
	tagMap := make(map[string]models.Tag)

	for _, tagName := range bookmarkData.Tags {
		// sin espacios
		tagName = strings.TrimSpace(tagName)

		if tagName == "" {
			continue // Saltar tags vacios
		}

		// verificar si ya procesamos este tag en este request

		if existingTag, exists := tagMap[tagName]; exists {
			tags = append(tags, existingTag)
			continue
		}

		// buscar en la base de datos
		var tag models.Tag
		result := database.DB.Where(&models.Tag{Title: tagName, UserID: bookmarkData.UserID}).First(&tag)

		if result.Error == gorm.ErrRecordNotFound {
			tag = models.Tag{Title: tagName, UserID: bookmarkData.UserID}
			if err := database.DB.Create(&tag).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to create tag: " + tagName,
					"error":   err.Error(),
				})
				return
			}
		}

		tagMap[tagName] = tag
		tags = append(tags, tag)
	}

	bookmark := models.Bookmarks{
		Title:  bookmarkData.Title,
		Url:    bookmarkData.Url,
		UserID: bookmarkData.UserID,
		Tags:   tags,
	}

	if err := database.DB.Where(&models.Bookmarks{Title: bookmarkData.Title, Url: bookmarkData.Url, UserID: bookmarkData.UserID}).First(&bookmark); err == nil {

		c.JSON(http.StatusConflict, gin.H{
			"message": "Bookmark with this title or URL already exists",
		})

		return
	}

	if err := database.DB.Create(&bookmark).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create bookmark",
			"data":    err.Error(),
		})

		return
	}

	database.DB.Preload("Tags").First(&bookmark, bookmark.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bookmark created successfully",
		"data":    bookmark,
	})
}

func GetBookmarkByID(c *gin.Context) {
	bookmarkID := c.Param("id")

	fmt.Println("Bookmark ID: ", bookmarkID)

	var bookmark models.Bookmarks
	result := database.DB.Preload("Tags").Where("id = ?", bookmarkID).First(&bookmark)

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
