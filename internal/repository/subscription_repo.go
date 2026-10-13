package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/april1858/subscriptions-service/internal/domain"
	"github.com/april1858/subscriptions-service/internal/dto"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error)
	Get(ctx context.Context, id string) (*domain.Subscription, error)
	List(ctx context.Context) ([]domain.Subscription, error)
	Update(ctx context.Context, id string, sub domain.SubscriptionUpdate) (domain.Subscription, error)
	Delete(ctx context.Context, id string) error
	GetTotalCostByPeriod(ctx context.Context, userID *string, serviceName *string, from time.Time, to time.Time) (*dto.SubscriptionCostResponse, error)
}

type subscriptionRepo struct {
	db  *pgxpool.Pool
	log *zap.Logger // вот откуда берётся log: это поле структуры
}

func NewSubscriptionRepository(pool *pgxpool.Pool, l *zap.Logger) *subscriptionRepo {
	return &subscriptionRepo{db: pool, log: l}
}

// Create вставляет подписку и возвращает её с ID (если он генерируется в БД)
func (r *subscriptionRepo) Create(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error) {
	r.log.Info("starting CREATE", zap.String("id", "CREATE")) // r.log — это наш экземпляр
	var created domain.Subscription
	err := r.db.QueryRow(ctx, `
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
	r.log.Info("starting GET", zap.String("id", id)) // r.log — это наш экземпляр
	var sub domain.Subscription
	err := r.db.QueryRow(ctx, "SELECT id, service_name, price, user_id, start_date FROM subscriptions WHERE id = $1", id).Scan(
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
	r.log.Info("starting LIST", zap.String("id", "LIST")) // r.log — это наш экземпляр
	rows, err := r.db.Query(ctx, "SELECT id, service_name, price, user_id, start_date FROM subscriptions")
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
// internal/repository/subscription_repository.go

func (r *subscriptionRepo) Update(ctx context.Context, id string, fields domain.SubscriptionUpdate) (domain.Subscription, error) {
	// Если вообще ничего не обновляем — можно сразу вернуть ErrNoRows как «не нашли»
	if fields.ServiceName == nil && fields.Price == nil && fields.StartDate == nil {
		return domain.Subscription{}, pgx.ErrNoRows
	}

	var sets []string
	var args []any
	i := 1

	if fields.ServiceName != nil {
		sets = append(sets, fmt.Sprintf("service_name = $%d", i))
		args = append(args, *fields.ServiceName)
		i++
	}
	if fields.Price != nil {
		sets = append(sets, fmt.Sprintf("price = $%d", i))
		args = append(args, *fields.Price)
		i++
	}
	if fields.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", i))
		args = append(args, *fields.StartDate)
		i++
	}

	query := fmt.Sprintf(`
		UPDATE subscriptions
		SET %s
		WHERE id = $%d
		RETURNING id, service_name, price, user_id, start_date
	`, strings.Join(sets, ", "), i)

	args = append(args, id)

	row := r.db.QueryRow(ctx, query, args...)
	var sub domain.Subscription
	err := row.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate)

	// ✅ Самое важное: возвращаем err как есть. Если это pgx.ErrNoRows — он долетит до хендлера.
	return sub, err
}

// Delete удаляет подписку по ID
func (r *subscriptionRepo) Delete(ctx context.Context, id string) error {
	r.log.Info("starting DELETE", zap.String("id", id)) // r.log — это наш экземпляр
	result, err := r.db.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

// internal/repository/subscription_repo.go
func (r *subscriptionRepo) GetTotalCostByPeriod(
	ctx context.Context,
	userID *string,
	serviceName *string,
	from time.Time,
	to time.Time,
) (*dto.SubscriptionCostResponse, error) {
	query := `
		SELECT COALESCE(SUM(price), 0) AS total_amount
		FROM subscriptions
		WHERE start_date <= $2
		  AND (end_date IS NULL OR end_date >= $1)
	`
	args := []any{from, to}
	nextParam := 3

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", nextParam)
		args = append(args, *userID)
		nextParam++
	}

	if serviceName != nil {
		query += fmt.Sprintf(" AND service_name = $%d", nextParam)
		args = append(args, *serviceName)
		nextParam++
	}

	var total int
	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	resp := &dto.SubscriptionCostResponse{
		TotalAmount: total,
		Currency:    "RUB",
		PeriodFrom:  from.Format(time.RFC3339),
		PeriodTo:    to.Format(time.RFC3339),
	}

	if userID != nil {
		resp.Filters.UserID = *userID
	}
	if serviceName != nil {
		resp.Filters.ServiceName = *serviceName
	}

	return resp, nil
}
