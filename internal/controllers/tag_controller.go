package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/services"
)

type TagController struct {
	service *services.TagService
}

func NewTagController() *TagController {
	return &TagController{
		service: services.NewTagService(),
	}
}

func (ctrl *TagController) FindTagsByUserID(c *gin.Context) {
	userID := c.Params("user_id")

	tags, err := ctrl.service.FindByUserIDService(userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener los tags para este usuario: " + err.Error(),
		})

		return
	}

	if len(tags) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontraron tags para este usuario",
			"data":    tags,
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tags obtenidos exitosamente",
		"data":    tags,
	})
}
