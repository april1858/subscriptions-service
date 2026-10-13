package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/april1858/subscriptions-service/internal/domain"
	"github.com/april1858/subscriptions-service/internal/dto"
	"github.com/april1858/subscriptions-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type SubscriptionHandlers struct {
	repo repository.SubscriptionRepository
	log  *zap.Logger
}

func NewSubscriptionHandlers(repo repository.SubscriptionRepository, l *zap.Logger) *SubscriptionHandlers {
	return &SubscriptionHandlers{repo: repo, log: l}
}

// @Summary Создать подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param body body dto.CreateSubscriptionRequest true "Данные подписки"
// @Router /subscriptions [post]
func (h *SubscriptionHandlers) Create(c *gin.Context) {
	h.log.Info("starting CREATE", zap.String("id", "CREATE")) // r.log — это наш экземпляр
	body, ok := c.Get("validatedBody")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing validated body"})
		return
	}

	req, ok := body.(dto.CreateSubscriptionRequest)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "type assertion failed"})
		return
	}

	sub := domain.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID, // если uuid.UUID, то req.UserID.String()
		StartDate:   req.StartDate,
	}

	// Используем репозиторий, который уже есть у хендлера:
	created, err := h.repo.Create(c.Request.Context(), sub)
	if err != nil {
		// Тут могут быть ошибки БД (включая твой CHECK)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create subscription",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetSubscription
// @Summary      Получить подписку по ID
// @Tags         subscriptions
// @Produce      json
// @Param        id path string true "UUID подписки"
// @Success      200 {object} domain.Subscription
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandlers) Get(c *gin.Context) {
	h.log.Info("starting GET", zap.String("id", "ID"))
	id := c.Param("id")

	sub, err := h.repo.Get(c.Request.Context(), id)
	if err == repository.ErrSubscriptionNotFound {
		h.log.Info("subscription not found", zap.String("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// @Summary Получить список подписок
// @Description Возвращает список подписок с фильтрацией по user_id и пагинацией
// @Tags subscriptions
// @Accept json
// @Produce json
// @QueryParam user_id query int false "ID пользователя"
// @QueryParam page query int false "Страница (по умолчанию 1)"
// @QueryParam per_page query int false "Количество на странице (по умолчанию 20)"
// @Router /subscriptions [get]
func (h *SubscriptionHandlers) List(c *gin.Context) {
	h.log.Info("starting LIST", zap.String("id", "LIST"))
	subs, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

// UpdateSubscription
// @Summary      Обновить подписку по ID
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id path string true "UUID подписки"
// @Param        input body domain.SubscriptionUpdate true "Новые данные подписки"
// @Success      200 {object} domain.Subscription
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandlers) Update(c *gin.Context) {
	h.log.Info("starting update", zap.String("id", "ID UPDATER"))
	// 1. Достаём ID из URL
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing subscription id in path"})
		return
	}

	// 2. Достаём валидированное тело из контекста (от middleware)
	body, ok := c.Get("validatedBody")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing validated body"})
		return
	}

	req, ok := body.(dto.UpdateSubscriptionRequest)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "type assertion failed"})
		return
	}

	// 3. Формируем DTO для репозитория (или маппим в domain.UpdateFields)
	// Здесь ты можешь сделать структуру UpdateFields в domain, где все поля — указатели.
	updateFields := domain.SubscriptionUpdate{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   req.StartDate,
	}

	// 4. Вызываем репозиторий
	sub, err := h.repo.Update(c.Request.Context(), id, updateFields)

	if err != nil {
		/* Сначала проверяем: это «нет такой подписки» или реальная ошибка?
		h.log.Info("DEBUG: inspecting error",
			zap.String("id", id),
			zap.String("errorMsg", err.Error()),
			zap.String("errorType", fmt.Sprintf("%T", err)),
		)
		*/
		if errors.Is(err, pgx.ErrNoRows) {
			// Это ожидаемая ситуация: ресурса нет. Логируем как Warning, а не Error.
			h.log.Warn("subscription not found for update",
				zap.String("id", id),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return // Важно: выходим из функции, дальше не идём!
		}

		// DeleteSubscription
		// @Summary      Удалить подписку по ID
		// @Tags         subscriptions
		// @Param        id path string true "UUID подписки"
		// @Success      204 "No Content"
		// @Router       /api/v1/subscriptions/{id} [delete]
		// Если это НЕ ErrNoRows — значит, случилась настоящая проблема (SQL-ошибка, сеть и т.д.)
		h.log.Error("update failed",
			zap.String("id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)

}

// DeleteSubscription
// @Summary      Удалить подписку по ID
// @Tags         subscriptions
// @Param        id path string true "UUID подписки"
// @Success      204 "No Content"
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandlers) Delete(c *gin.Context) {
	h.log.Info("starting DELETE", zap.String("id", "ID DELETE"))
	id := c.Param("id")

	err := h.repo.Delete(c.Request.Context(), id)
	if err == repository.ErrSubscriptionNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subscription", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// internal/handlers/subscription_handlers.go
// @Summary Подсчёт суммарной стоимости подписок за период
// @Description Считает сумму price_amount по подпискам, пересекающим период [from, to], с опциональной фильтрацией по user_id и service_name.
// @Tags aggregates
// @Accept json
// @Produce json
// @Param request body dto.SubscriptionCostRequest true "Параметры агрегации"
// @Success 200 {object} dto.SubscriptionCostResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /aggregates/total-cost [post]
func (h *SubscriptionHandlers) GetTotalCost(c *gin.Context) {
	var req dto.SubscriptionCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("bad request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Простая валидация периода
	if req.From.After(req.To) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from must be before or equal to to"})
		return
	}

	stats, err := h.repo.GetTotalCostByPeriod(c.Request.Context(), req.UserID, req.ServiceName, req.From, req.To)
	if err != nil {
		h.log.Error("GetTotalCost failed", zap.Error(err), zap.String("from", req.From.Format(time.RFC3339)), zap.String("to", req.To.Format(time.RFC3339)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
