package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // side-effect import: регистрирует драйвер "pgx"
	"github.com/pressly/goose/v3"
)

// RunMigrations применяет все миграции из указанной папки.
//
// Парадокс, с которым мы столкнулись:
//   - pgxpool (pgx native) — быстрый, с пулом соединений.
//   - goose работает только с *sql.DB (стандартный database/sql).
//   - pgx.Conn НЕ реализует интерфейс *sql.DB.
//
// Решение: bridge pattern через github.com/jackc/pgx/v5/stdlib.
// stdlib — это адаптер, который реализует driver.Driver interface
// для database/sql поверх pgx. Мы открываем *sql.DB с драйвером "pgx",
// goose работает с ним нативно, а под капотом — pgx.
//
// Философия: Right tool for the right job.
// Для миграций (разовый запуск) не нужен пул соединений —
// достаточно одного *sql.DB. Для репозитория (постоянные запросы)
// будем использовать pgxpool.Pool (на Дне 2).
//
// Техника: side-effect import (_ "...")
// Мы не вызываем функции из stdlib напрямую.
// Импорт нужен только для того, чтобы выполнился init(),
// который регистрирует драйвер "pgx" в database/sql.
func RunMigrations(connStr string, dir string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}
