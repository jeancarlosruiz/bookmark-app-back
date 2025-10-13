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

func CreateBookmark(c *gin.Context){
  var bookmark []models.Bookmarks
  if err := c.ShouldBindJSON(&bookmark); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
  }

  database.DB.Create(&bookmark)
  c.JSON(http.StatusOK, bookmark)
}
