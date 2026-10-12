package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogRequests возвращает Gin-хендлер, который логирует все HTTP-запросы.
// Это чистая абстракция: он ничего не знает про бизнес-логику, только про HTTP.
func LogRequests(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Сначала запускаем остальные хендлеры (валидацию, сам эндпоинт и т.д.)
		c.Next()

		latency := time.Since(start)

		log.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		)
	}
}
