package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validator[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload T

		fmt.Println("Este son los datos", payload)

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

	for _, err := range err.(validator.ValidationErrors) {
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
