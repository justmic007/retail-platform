// Package handler contains the HTTP handlers for the Order Service.
// Thin layer — binds JSON, validates, calls service, maps errors to HTTP.
package handler

import (
	"net/http"

	"retail-platform/order/internal/domain"
	"retail-platform/order/internal/service"
	"retail-platform/pkg/errors"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/middleware"
	"retail-platform/pkg/validator"

	"github.com/gin-gonic/gin"
)

// OrderHandler holds dependencies for all order HTTP handlers.
type OrderHandler struct {
	service   *service.OrderService
	validator *validator.Validator
	log       *logger.Logger
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(svc *service.OrderService, v *validator.Validator, log *logger.Logger) *OrderHandler {
	return &OrderHandler{
		service:   svc,
		validator: v,
		log:       log,
	}
}

// CreateOrder handles POST /orders
// Saves the order as PENDING and submits it to the worker pool.
// Returns 202 Accepted immediately — processing happens asynchronously.
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	var req domain.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}
	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	order, err := h.service.CreateOrder(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// 202 Accepted — order received, processing in background
	c.JSON(http.StatusAccepted, gin.H{
		"order":   order.ToResponse(),
		"message": "order received and is being processed",
	})
}

// GetOrder handles GET /orders/:id
// Returns a single order. Ownership enforced in service layer.
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	orderID := c.Param("id")

	order, err := h.service.GetOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": order.ToResponse(),
	})
}

// ListOrders handles GET /orders
// Returns all orders for the authenticated user.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	orders, err := h.service.ListOrders(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response := make([]*domain.OrderResponse, 0, len(orders))
	for _, o := range orders {
		response = append(response, o.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": response,
		"total":  len(response),
	})
}

// CancelOrder handles PATCH /orders/:id/cancel
// Cancels a PENDING order. Only the order owner can cancel.
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	orderID := c.Param("id")

	if err := h.service.CancelOrder(c.Request.Context(), orderID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "order cancelled successfully",
	})
}

// Health handles GET /health
func (h *OrderHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "order",
	})
}

// Ready handles GET /ready
func (h *OrderHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "order",
	})
}

// handleError maps domain errors to HTTP responses.
func (h *OrderHandler) handleError(c *gin.Context, err error) {
	if appErr, ok := errors.IsAppError(err); ok {
		c.JSON(appErr.HTTPStatus(), gin.H{
			"error": appErr.Message,
			"code":  string(appErr.Code),
		})
		return
	}

	h.log.Error().Err(err).Msg("unhandled internal error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "an internal error occurred",
		"code":  "INTERNAL_ERROR",
	})
}
