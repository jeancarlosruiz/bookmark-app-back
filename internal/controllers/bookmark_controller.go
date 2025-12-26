package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/services"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)

type BookmarkController struct {
	service *services.BookmarkService
}

func NewBookmarkController() *BookmarkController {
	return &BookmarkController{
		service: services.NewBookmarkService(),
	}
}

func (ctrl *BookmarkController) CreateBookmark(c *gin.Context) {

	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})

		return
	}

	bookmarkData := payload.(validator.CreateBookmark)

	bookmark, err := ctrl.service.CreateBookmarkService(bookmarkData)

	if err != nil {
		if err == services.ErrBookmarkAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"message": err.Error(),
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark created successfully",
		"data":    bookmark,
	})

}

func (ctrl *BookmarkController) GetBookmarkByID(c *gin.Context) {
	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
		})

		return
	}

	bookmark, err := ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark obtenido exitosamente",
		"data":    bookmark,
	})
}

func (ctrl *BookmarkController) GetBookmarkByUserID(c *gin.Context) {
	userID := c.Param("user_id")

	bookmarks, err := ctrl.service.FindByUserIDWithTagService(userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener bookmarks para este usuario: " + err.Error(),
		})
		return
	}

	if len(bookmarks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontraron bookmarks para este usuario",
			"data":    bookmarks,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmarks obtenidos exitosamente",
		"data":    bookmarks,
	})
}

// Mejorar
func (ctrl *BookmarkController) SearchBookmarkByTags(c *gin.Context) {
	query := c.Query("q")
	userID := c.GetString("user_id")

	var tagList = strings.Split(query, ",")

	var searchList []string

	for _, tag := range tagList {

		tag = strings.TrimSpace(strings.ToLower(tag))

		if tag == "" {
			continue
		}

		searchList = append(searchList, tag)

	}

	bookmarks, err := ctrl.service.FindBookmarksByTagsService(searchList, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark " + err.Error(),
			"error":   err,
		})

		return
	}

	if len(bookmarks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No bookmarks encontrados",
			"data":    []models.Bookmarks{},
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark encontrados satifactoriamente",
		"data":    bookmarks,
	})

}

func (ctrl *BookmarkController) SearchBookmarkByTitle(c *gin.Context) {
	bookmarkTitle := c.Query("q")
	// Aqui deberia ir tambien el UserID
	userID := c.GetString("user_id")

	bookmarks, err := ctrl.service.FindBookmarkByTitleService(bookmarkTitle, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark " + err.Error(),
			"error":   err.Error(),
		})

		return
	}

	if len(bookmarks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se encontro bookmark con este title: " + bookmarkTitle,
			"data":    []models.Bookmarks{},
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark encontrados satifactoriamente",
		"data":    bookmarks,
	})
}

func (ctrl *BookmarkController) DeleteBookmark(c *gin.Context) {
	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
			"error":   err.Error(),
		})
		return
	}

	_, err = ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro al encontrar el bookmark" + err.Error(),
		})

		return
	}

	_, err = ctrl.service.SoftDeleteBookmarkByIDService(uint(bookmarkID), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al eliminar bookmark",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "El bookmark fue eliminado correctamente ",
	})

}

func (ctrl *BookmarkController) UpdateBookmark(c *gin.Context) {
	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
			"error":   err.Error(),
		})
		return
	}

	_, err = ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este id",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al encontrar el bookmark" + err.Error(),
		})

		return
	}

	payload, exist := c.Get("payload")

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Validation payload not found",
		})

		return
	}

	bookmarkData := payload.(validator.UpdateBookmark)

	bookmarkUpdated, err := ctrl.service.UpdateBookmarkService(uint(bookmarkID), userID, bookmarkData)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark created successfully",
		"data":    bookmarkUpdated,
	})

}

func (ctrl *BookmarkController) IncrementVisitCountController(c *gin.Context) {

	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
			"error":   err.Error(),
		})
		return
	}

	_, err = ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este id",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al encontrar el bookmark" + err.Error(),
		})

		return
	}

	bookmarkUpdated, err := ctrl.service.IncrementVisitCountService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark actualizado successfully",
		"data":    bookmarkUpdated,
	})

}

func (ctrl *BookmarkController) TogglePinnedByIDController(c *gin.Context) {

	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
			"error":   err.Error(),
		})
		return
	}

	_, err = ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este id",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al encontrar el bookmark" + err.Error(),
		})

		return
	}

	bookmarkUpdated, err := ctrl.service.TogglePinnedByIDService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark actualizado successfully",
		"data":    bookmarkUpdated,
	})

}

func (ctrl *BookmarkController) ToggleIsArchiveByIDController(c *gin.Context) {

	bookmarkIDStr := c.Param("id")
	userID := c.GetString("user_id")

	bookmarkID, err := strconv.Atoi(bookmarkIDStr)

	if err != nil || bookmarkID <= 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Formato de ID invalido",
			"error":   err.Error(),
		})
		return
	}

	_, err = ctrl.service.FindByIDWithTagsService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este id",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al encontrar el bookmark" + err.Error(),
		})

		return
	}

	bookmarkUpdated, err := ctrl.service.ToggleIsArchiveByIDService(uint(bookmarkID), userID)

	if err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "No se encontro bookmark con este ID",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al buscar bookmark",
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bookmark actualizado successfully",
		"data":    bookmarkUpdated,
	})

}

func (ctrl *BookmarkController) PreviewMetadata(c *gin.Context) {
	url := c.Query("url")
	userID := c.GetString("user_id")

	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "URL es requerida",
		})

		return
	}

	// Verificar si la URL ya existe para este usuario
	err := ctrl.service.CheckURLExists(url, userID)

	if err != nil {
		if err == services.ErrBookmarkURLAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"message": "Ya existe un bookmark con esta URL",
				"error":   err.Error(),
			})

			return
		}

		// Otro tipo de error de base de datos
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error al verificar la URL",
			"error":   err.Error(),
		})

		return
	}

	// Crear servicios
	cacheService := &services.CacheService{}
	ctx := c.Request.Context()

	// 1. CACHE HIT PATH: Intentar obtener metadata desde caché
	if cachedMetadata, hit, _ := cacheService.GetMetadataFromCache(ctx, url); hit {
		// Respuesta con datos cacheados
		responseData := gin.H{
			"title":       cachedMetadata.Title,
			"description": cachedMetadata.Description,
			"favicon":     cachedMetadata.Favicon,
			"cached":      true,
		}

		if cachedMetadata.Error != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "Metadatos obtenidos desde caché (parciales)",
				"data":    responseData,
				"error":   cachedMetadata.Error.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Metadatos obtenidos desde caché",
			"data":    responseData,
		})
		return
	}

	// 2. CACHE MISS PATH: Hacer scraping de la URL
	scraperService := services.NewScraperService()
	metadata := scraperService.ScrapeMetadataAsync(url, 8*time.Second)

	// 3. Guardar en caché (incluso metadata parcial)
	cacheService.SetMetadataCache(ctx, url, metadata)

	// 4. Responder al cliente
	responseData := gin.H{
		"title":       metadata.Title,
		"description": metadata.Description,
		"favicon":     metadata.Favicon,
		"cached":      false,
	}

	if metadata.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "No se pudieron obtener metadatos completos",
			"data":    responseData,
			"error":   metadata.Error.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Metadatos obtenidos exitosamente",
		"data":    responseData,
	})
}
