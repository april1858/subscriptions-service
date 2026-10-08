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
	// Достаём уже валидированное тело из контекста
	body, ok := c.Get("validatedBody")
	if !ok {
		// Это значит, что middleware не сработал или был пропущен — лучше вернуть 400
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing validated body"})
		return
	}

	req, ok := body.(dto.CreateSubscriptionRequest)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "type assertion failed"})
		return
	}

	// Превращаем DTO в доменную модель (если нужно)
	sub := domain.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
	}

	repo := c.MustGet("subscriptionRepo").(repository.SubscriptionRepository)
	created, err := repo.Create(c.Request.Context(), sub)
	if err != nil {
		// Тут уже могут быть ошибки БД (включая твой CHECK)
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
	id := c.Param("id")
	var input domain.Subscription
	input.ID = id // ID из URL

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "details": err.Error()})
		return
	}

	err := h.repo.Update(c.Request.Context(), input)
	if err == repository.ErrSubscriptionNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription", "details": err.Error()})
		return
	}

	// Возвращаем обновлённую подписку
	updated, _ := h.repo.Get(c.Request.Context(), id) // упрощение: можно вернуть input или сделать отдельный метод
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
