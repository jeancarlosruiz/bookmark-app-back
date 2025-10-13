package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

func GetUsers(c *gin.Context) {
	var users []models.User
	database.DB.Find(&users)

  fmt.Println("Depues de el middleware")
	c.JSON(http.StatusOK, users)
}

func CreateUser(c *gin.Context) {
	var user []models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	database.DB.Create(&user)
	c.JSON(http.StatusOK, user)
}
