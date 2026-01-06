package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/services"
)

// handles migration related http request
type MigrationController struct {
	migrationService *services.MigrationService
}

// create a new migration controller
func NewMigrationController() *MigrationController {
	return &MigrationController{
		migrationService: services.NewMigrationService(),
	}
}

// defines the expected request body structure
type MigrateUserRequest struct {
	AnonymousUserID string `json:"anonymous_user_id" binding:"required"`
	NewUserID       string `json:"new_user_id" binding:"required"`
}

func (mc *MigrationController) MigrateUser(c *gin.Context) {
	var req MigrateUserRequest

	// validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: anonymous_user_id is required",
		})

		return
	}

	// authenticatedUserID := c.GetString("user_id")
	// if authenticatedUserID == "" {
	// 	c.JSON(http.StatusUnauthorized, gin.H{
	// 		"error": "User Id not found in token context",
	// 	})
	//
	// 	return
	// }

	result, err := mc.migrationService.MigrateUser(c.Request.Context(), req.AnonymousUserID, req.NewUserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Migration failed",
			"detailes": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User data migrated successfully",
		"data":    result,
	})
}
