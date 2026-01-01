package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/services"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
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
	userID := c.Param("user_id")

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

func (ctrl *TagController) CreateTag(c *gin.Context) {
	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})
		return
	}

	tagData := payload.(validator.CreateTag)
	userID := c.GetString("user_id")

	tag, err := ctrl.service.CreateTagService(tagData.Title, userID)

	if err != nil {
		if errors.Is(err, services.ErrTagAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al crear el tag: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tag creado exitosamente",
		"data":    tag,
	})
}

func (ctrl *TagController) UpdateTag(c *gin.Context) {
	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})
		return
	}

	tagData := payload.(validator.UpdateTag)
	tagID := c.Param("id")
	userID := c.GetString("user_id")

	tag, err := ctrl.service.UpdateTagService(tagID, tagData.Title, userID)

	if err != nil {
		if errors.Is(err, services.ErrTagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": err.Error(),
			})
			return
		}

		if errors.Is(err, services.ErrTagUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": err.Error(),
			})
			return
		}

		if errors.Is(err, services.ErrTagAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al actualizar el tag: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tag actualizado exitosamente",
		"data":    tag,
	})
}

func (ctrl *TagController) DeleteTag(c *gin.Context) {
	tagID := c.Param("id")
	userID := c.GetString("user_id")

	err := ctrl.service.DeleteTagService(tagID, userID)

	if err != nil {
		if errors.Is(err, services.ErrTagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": err.Error(),
			})
			return
		}

		if errors.Is(err, services.ErrTagUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": err.Error(),
			})
			return
		}

		if errors.Is(err, services.ErrTagHasBookmarks) {
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al eliminar el tag: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tag eliminado exitosamente",
	})
}
