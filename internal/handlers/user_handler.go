package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
)

func GetUsers(c *gin.Context) {
	var users []models.User
	result := database.DB.Where("deleted_at IS NULL").Order("created_at DESC").Find(&users)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener usuarios " + result.Error.Error(),
		})

		return
	}

	if len(users) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontraron usuarios",
			"data":    []models.User{},
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Usuarios obtenidos exitosamente",
		"data":    users,
	})

}
