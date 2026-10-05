package config

import (
	"fmt"

	"github.com/caarlos0/env/v7"
)

// Config — корневая структура конфигурации.
// Состоит из подконфигов, каждый со своим префиксом env-переменных.
//
// Методология: Separation of Concerns (разделение ответственностей).
// Серверные настройки не смешиваются с настройками БД —
// каждый блок имеет свою зону ответственности и свой префикс.
type Config struct {
	Server ServerConfig
	DB     DBConfig
	Logger LoggerConfig
}

type ServerConfig struct {
	Port int `env:"SERVER_PORT"`
}

type DBConfig struct {
	Host     string `env:"DB_HOST"`
	Port     int    `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Name     string `env:"DB_NAME"`
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL"`
}

// Load читает переменные окружения и возвращает готовый конфиг.
//
// Почему не используем тег default=...:
// библиотека caarlos0/env не поддерживает defaultValue в тегах.
// Вместо этого задаём дефолты явно в коде — это прозрачнее
// и не зависит от «магии» парсера.
//
// Техника: Fail-Fast — если конфиг невалиден, падаем сразу,
// а не в момент первого запроса.
func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// Дефолты — если переменная не задана в .env,
	// используем безопасное значение.
	// Философия: конфиг должен работать «из коробки» для локальной разработки,
	// но позволять переопределять через .env или переменные окружения в проде.
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.DB.Host == "" {
		c.DB.Host = "localhost"
	}
	if c.DB.Port == 0 {
		c.DB.Port = 5432
	}
	if c.DB.User == "" {
		c.DB.User = "postgres"
	}
	if c.DB.Password == "" {
		c.DB.Password = "postgres"
	}
	if c.DB.Name == "" {
		c.DB.Name = "subscriptions"
	}
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}

	return &c, nil
}

// ConnString возвращает строку подключения к PostgreSQL в формате,
// который понимают и pgxpool, и database/sql (через stdlib).
//
// Абстракция: единая точка формирования строки подключения.
// Меняется формат — меняем только здесь, а не по всему проекту.
func (c *Config) ConnString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name,
	)
}
