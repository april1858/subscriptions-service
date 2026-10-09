package handlers

import (
	"net/http"

	"github.com/april1858/subscriptions-service/internal/domain"
	"github.com/april1858/subscriptions-service/internal/dto"
	"github.com/april1858/subscriptions-service/internal/repository"
	"github.com/gin-gonic/gin"
)

type SubscriptionHandlers struct {
	repo repository.SubscriptionRepository
}

func NewSubscriptionHandlers(repo repository.SubscriptionRepository) *SubscriptionHandlers {
	return &SubscriptionHandlers{repo: repo}
}

func (h *SubscriptionHandlers) Create(c *gin.Context) {
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

func (h *SubscriptionHandlers) Get(c *gin.Context) {
	id := c.Param("id")

	sub, err := h.repo.Get(c.Request.Context(), id)
	if err == repository.ErrSubscriptionNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

func (h *SubscriptionHandlers) List(c *gin.Context) {
	subs, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *SubscriptionHandlers) Update(c *gin.Context) {
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
	updated, err := h.repo.Update(c.Request.Context(), id, updateFields)
	if err != nil {
		// Тут могут быть ошибки БД: не найден, конфликт, нарушение CHECK и т.д.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update subscription",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *SubscriptionHandlers) Delete(c *gin.Context) {
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
