package logger

import "go.uber.org/zap"

// New создаёт настроенный логгер Zap.
//
// Методология: Structured Logging — логи в JSON, а не plain text.
// Это позволяет потом парсить их в ELK, Loki или любой другой системе.
//
// Техника: один логгер на всё приложение (Singleton-подобный паттерн).
// Создаётся в main, передаётся по ссылкам — не глобальная переменная,
// а явно инжектируемая зависимость (Dependency Injection).
func New(level string) (*zap.Logger, error) {
	var l *zap.Logger
	var err error

	switch level {
	case "debug":
		l, err = zap.NewDevelopment()
	default:
		l, err = zap.NewProduction()
	}

	if err != nil {
		return nil, err
	}
	return l, nil
}
