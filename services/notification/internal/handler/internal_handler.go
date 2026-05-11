// Package handler — internal notifications (warehouse staff, ops team).
package handler

import (
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// InternalHandler sends internal alerts to warehouse staff and ops.
type InternalHandler struct {
	log *logger.Logger
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(log *logger.Logger) *InternalHandler {
	return &InternalHandler{log: log}
}

// SendLowStockAlert sends a low stock alert to warehouse staff.
func (h *InternalHandler) SendLowStockAlert(event events.StockEvent) {
	h.log.Warn().
		Str("type", "LOW_STOCK_ALERT").
		Str("product_id", event.ProductID).
		Str("product_name", event.ProductName).
		Int("stock_level", event.StockLevel).
		Msg("[SLACK] low stock alert sent to warehouse")
}
