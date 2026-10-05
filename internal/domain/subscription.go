package domain

import "time"

// Subscription — доменная модель подписки.
//
// Пакет domain — сердце приложения. Здесь нет ни БД, ни HTTP, ни логгера.
// Только структуры данных, которые описывают предметную область.
//
// Методология: Clean Architecture / Hexagonal Architecture.
// Домен не зависит от инфраструктуры (БД, фреймворков).
// Инфраструктура зависит от домена, а не наоборот.
// Это позволяет менять БД или HTTP-фреймворк, не трогая бизнес-логику.
//
// Техника: теги db используются репозиторием для маппинга в pgx.
// Теги json используются транспортным слоем для сериализации.
// Домен не знает, как именно эти теги используются —
// он просто их декларирует.
type Subscription struct {
	ID          string    `json:"id" db:"id"`
	ServiceName string    `json:"service_name" db:"service_name"`
	Price       int       `json:"price" db:"price"`
	UserID      string    `json:"user_id" db:"user_id"`
	StartDate   string    `json:"start_date" db:"start_date"` // формат "MM-YYYY"
	EndDate     *string   `json:"end_date,omitempty" db:"end_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// EndDate — указатель (*string), потому что подписка может быть активной
// (без даты окончания) — это nil в Go и NULL в БД.
// omitempty в json-теге: если EndDate == nil, поле не попадает в ответ.
//
// Цена — int, а не float64: в задании сказано «целое число рублей».
// Использование int вместо float64 для денег — осознанное решение:
// avoids floating-point errors (0.1 + 0.2 != 0.3 в IEEE 754).
