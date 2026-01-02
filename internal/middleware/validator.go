package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validator[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload T

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid JSON format",
				"data":    err.Error(),
			})

			c.Abort()
			return
		}

		if err := validate.Struct(payload); err != nil {
			errors := formatValidationErrors(err)
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Validation failed",
				"data":    errors,
			})

			c.Abort()
			return
		}

		c.Set("payload", payload)
		c.Next()
	}
}

func formatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	validationErrors, ok := err.(validator.ValidationErrors)

	if !ok {
		errors["_error"] = "Error de validacion inesperado"
	}

	for _, err := range validationErrors {
		fieldName := err.Field()

		switch err.Tag() {
		case "required":
			errors[fieldName] = fieldName + " is required"
		case "url":
			errors[fieldName] = fieldName + " must be a valid URL"
		default:
			errors[fieldName] = fieldName + " is invalid"
		}
	}

	return errors
}
