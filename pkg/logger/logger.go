package logger

import (
	"go.uber.org/zap"
)

// New создаёт настроенный логгер Zap.
//
// Методология: Structured Logging — логи в JSON, а не plain text.
// Это позволяет потом парсить их в ELK, Loki или любой другой системе.
//
// Техника: один логгер на всё приложение (Singleton-подобный паттерн).
// Создаётся в main, передаётся по ссылкам — не глобальная переменная,
// а явно инжектируемая зависимость (Dependency Injection).

// NewLogger создаёт логгер с заданной конфигурацией
func NewLogger(prod bool) (*zap.Logger, error) {
	if prod {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
