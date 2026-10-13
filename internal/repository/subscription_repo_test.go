package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	// замени на свой путь
)

func TestGetTotalCostByPeriod(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/subscriptions"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	testUserID := "550e8400-e29b-41d4-a716-446655440000"

	// Очистка перед тестом
	_, err = pool.Exec(ctx, "DELETE FROM subscriptions WHERE user_id = $1", testUserID)
	if err != nil {
		t.Fatalf("failed to clean test data: %v", err)
	}

	// Вставляем данные
	_, err = pool.Exec(ctx, `
		INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES
		  ($1, 'Spotify', 499, $2, '2024-09', NULL),
		  ($3, 'YouTube Premium', 799, $2, '2024-10', '2024-12'),
		  ($4, 'Netflix', 699, '11111111-1111-1111-1111-111111111111', '2024-11', NULL)
	`,
		"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		testUserID,
		"b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
		"c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
	)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	repo := &subscriptionRepo{db: pool}

	from := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 1, 23, 59, 59, 0, time.UTC)

	t.Run("all_subscriptions_for_user", func(t *testing.T) {
		resp, err := repo.GetTotalCostByPeriod(ctx, &testUserID, nil, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 1298.0, resp.TotalAmount) // 499 + 799
	})

	t.Run("filter_by_service", func(t *testing.T) {
		serviceName := "Spotify"
		resp, err := repo.GetTotalCostByPeriod(ctx, &testUserID, &serviceName, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 499.0, resp.TotalAmount)
	})

	t.Run("no_subscriptions_in_period", func(t *testing.T) {
		// Период до начала всех подписок
		fromOld := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		toOld := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
		resp, err := repo.GetTotalCostByPeriod(ctx, &testUserID, nil, fromOld, toOld)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, resp.TotalAmount)
	})

	t.Run("cross_boundary_subscription", func(t *testing.T) {
		// Подписка пересекает период
		fromMid := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
		toMid := time.Date(2024, 11, 30, 23, 59, 59, 0, time.UTC)
		resp, err := repo.GetTotalCostByPeriod(ctx, &testUserID, nil, fromMid, toMid)
		assert.NoError(t, err)
		// Должна попасть и Spotify (активна), и YouTube (активна в ноябре)
		assert.Equal(t, 1298.0, resp.TotalAmount)
	})
}
