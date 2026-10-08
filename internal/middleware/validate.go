package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var v = validator.New()

// ValidateJSONBody валидирует JSON-тело запроса и сразу возвращает 400 при ошибках.
// T — тип структуры, в которую будем биндить и валидировать.
func ValidateJSONBody[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body T

		// 1. Биндим JSON в структуру
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid JSON",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// 2. Валидируем структуру
		if err := v.Struct(body); err != nil {
			var validationErr validator.ValidationErrors
			if errors.As(err, &validationErr) {
				fields := make(map[string]string)
				for _, e := range validationErr {
					// Человекопонятное сообщение можно сделать кастомным, пока — теги
					fields[e.Field()] = e.Tag() + " failed for field " + e.Field()
				}
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "validation error",
					"details": fields,
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "validation error",
					"details": err.Error(),
				})
			}
			c.Abort()
			return
		}

		// Если всё ок — кладём валидированное тело в контекст, чтобы хендлер мог его достать
		c.Set("validatedBody", body)
		c.Next()
	}
}
