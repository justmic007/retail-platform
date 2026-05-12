// Package handler contains the HTTP handlers for the Inventory Service.
// Thin layer — binds JSON, validates, calls service, maps errors to HTTP.
package handler

import (
	"net/http"

	"retail-platform/inventory/internal/domain"
	"retail-platform/inventory/internal/service"
	"retail-platform/pkg/errors"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/validator"

	"github.com/gin-gonic/gin"
)

// InventoryHandler holds dependencies for all inventory HTTP handlers.
type InventoryHandler struct {
	service   *service.InventoryService
	validator *validator.Validator
	log       *logger.Logger
}

// NewInventoryHandler creates a new InventoryHandler.
func NewInventoryHandler(
	svc *service.InventoryService,
	v *validator.Validator,
	log *logger.Logger,
) *InventoryHandler {
	return &InventoryHandler{
		service:   svc,
		validator: v,
		log:       log,
	}
}

// ListProducts handles GET /products
// Returns all active products with their stock levels.
func (h *InventoryHandler) ListProducts(c *gin.Context) {
	products, err := h.service.ListProducts(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert domain types to API response types
	response := make([]*domain.ProductResponse, 0, len(products))
	for _, p := range products {
		response = append(response, p.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"products": response,
		"total":    len(response),
	})
}

// GetProduct handles GET /products/:id
// Returns a single product with its stock level.
func (h *InventoryHandler) GetProduct(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "product id is required",
			"code":  "BAD_REQUEST",
		})
		return
	}

	product, err := h.service.GetProduct(c.Request.Context(), productID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product.ToResponse(),
	})
}

// GetStockLevel handles GET /products/:id/stock
// Returns just the stock level for a product.
// Called by Order Service before attempting a reservation.
func (h *InventoryHandler) GetStockLevel(c *gin.Context) {
	productID := c.Param("id")

	stock, err := h.service.GetStockLevel(c.Request.Context(), productID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, &domain.StockResponse{
		ProductID: stock.ProductID,
		Quantity:  stock.Quantity,
		Reserved:  stock.Reserved,
		Available: stock.Available(),
	})
}

// Reserve handles POST /inventory/reserve
// Locks N units of a product for an order.
// Called by Order Service when processing an order.
func (h *InventoryHandler) Reserve(c *gin.Context) {
	var req domain.ReserveRequest

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

	if err := h.service.Reserve(c.Request.Context(), req); err != nil {
		h.handleError(c, err)
		return
	}

	// Fetch updated stock to include in response
	stock, _ := h.service.GetStockLevel(c.Request.Context(), req.ProductID)
	available := 0
	reserved := 0
	if stock != nil {
		available = stock.Available()
		reserved = stock.Reserved
	}

	c.JSON(http.StatusOK, &domain.ReserveResponse{
		ProductID: req.ProductID,
		Reserved:  reserved,
		Available: available,
		Message:   "stock reserved successfully",
	})
}

// Release handles POST /inventory/release
// Returns previously reserved units back to available stock.
func (h *InventoryHandler) Release(c *gin.Context) {
	var req domain.ReleaseRequest

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

	if err := h.service.Release(c.Request.Context(), req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "stock released successfully",
	})
}

// AdjustStock handles PATCH /products/:id/stock
// Manually sets the total quantity for a product.
// Admin only — enforced by RequireRole middleware in router.
func (h *InventoryHandler) AdjustStock(c *gin.Context) {
	productID := c.Param("id")

	var req domain.StockAdjustRequest
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

	if err := h.service.AdjustStock(c.Request.Context(), productID, req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "stock adjusted successfully",
	})
}

// Health handles GET /health
func (h *InventoryHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "inventory",
	})
}

// Ready handles GET /ready
func (h *InventoryHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "inventory",
	})
}

// handleError maps domain errors to HTTP responses.
func (h *InventoryHandler) handleError(c *gin.Context, err error) {
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
