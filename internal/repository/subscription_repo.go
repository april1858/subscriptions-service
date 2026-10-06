package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/april1858/subscriptions-service/internal/domain"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error)
	Get(ctx context.Context, id string) (*domain.Subscription, error)
	List(ctx context.Context) ([]domain.Subscription, error)
	Update(ctx context.Context, sub domain.Subscription) error
	Delete(ctx context.Context, id string) error
}

type subscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) SubscriptionRepository {
	return &subscriptionRepo{pool: pool}
}

// Create вставляет подписку и возвращает её с ID (если он генерируется в БД)
func (r *subscriptionRepo) Create(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error) {
	var created domain.Subscription
	err := r.pool.QueryRow(ctx, `
		INSERT INTO subscriptions (service_name, price, user_id, start_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, service_name, price, user_id, start_date
	`, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate).Scan(
		&created.ID,
		&created.ServiceName,
		&created.Price,
		&created.UserID,
		&created.StartDate,
	)
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return &created, nil
}

// Get ищет по ID
func (r *subscriptionRepo) Get(ctx context.Context, id string) (*domain.Subscription, error) {
	var sub domain.Subscription
	err := r.pool.QueryRow(ctx, "SELECT id, service_name, price, user_id, start_date FROM subscriptions WHERE id = $1", id).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return &sub, nil
}

// List возвращает все подписки
func (r *subscriptionRepo) List(ctx context.Context) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, service_name, price, user_id, start_date FROM subscriptions")
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate); err != nil {
			return nil, fmt.Errorf("scan subscription row: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// Update обновляет существующую подписку (по ID)
func (r *subscriptionRepo) Update(ctx context.Context, sub domain.Subscription) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE subscriptions
		SET service_name = $1, price = $2, user_id = $3, start_date = $4
		WHERE id = $5
	`, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.ID)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

// Delete удаляет подписку по ID
func (r *subscriptionRepo) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}
