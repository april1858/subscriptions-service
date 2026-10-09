package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	Update(ctx context.Context, id string, sub domain.SubscriptionUpdate) error
	Delete(ctx context.Context, id string) error
}

type subscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *subscriptionRepo {
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
func (r *subscriptionRepo) Update(ctx context.Context, id string, fields domain.SubscriptionUpdate) (domain.Subscription, error) {
	// 1. Сначала читаем, чтобы вернуть актуальную запись (или проверить существование)
	sub, err := r.Get(ctx, id)
	if err != nil {
		return domain.Subscription{}, err
	}

	// Если ничего не обновляем — возвращаем текущее состояние
	if fields.ServiceName == nil && fields.Price == nil && fields.StartDate == nil {
		return *sub, nil
	}

	var sets []string
	var args []interface{}
	argIndex := 1

	if fields.ServiceName != nil {
		sets = append(sets, fmt.Sprintf("service_name = $%d", argIndex))
		args = append(args, *fields.ServiceName)
		argIndex++
	}
	if fields.Price != nil {
		sets = append(sets, fmt.Sprintf("price = $%d", argIndex))
		args = append(args, *fields.Price)
		argIndex++
	}
	if fields.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", argIndex))
		args = append(args, *fields.StartDate)
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE subscriptions
		SET %s
		WHERE id = $%d
		RETURNING id, service_name, price, user_id, start_date
	`, strings.Join(sets, ", "), argIndex)

	args = append(args, id) // последний аргумент — id

	// ВАЖНО: для pgxpool используем QueryRowContext (с контекстом!)

	row := r.pool.QueryRow(ctx, query, args...)

	err = row.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate)
	if err != nil {
		// Если строки не нашлось — будет pgx.ErrNoRows
		return domain.Subscription{}, err
	}

	return *sub, nil
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
