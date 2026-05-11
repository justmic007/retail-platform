// Package handler contains the notification handlers.
// Each handler is responsible for one notification channel (email, SMS, Slack).
// Handlers contain structured log stubs — swapping in a real provider like
// SendGrid requires changing one function body, not the architecture.

package handler

import (
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// EmailHandler sends the email notifications to customers
type EmailHandler struct {
	log *logger.Logger
}

// NewEmailHandler creates an email handler
func NewEmailHandler(log *logger.Logger) *EmailHandler {
	return &EmailHandler{log: log}
}

// SendOrderConfirmation sends an order confirmation email to the customer.
func (h *EmailHandler) SendOrderConfirmation(event events.OrderEvent) {
	h.log.Info().
		Str("type", "ORDER_CONFIRMATION").
		Str("order_id", event.OrderID).
		Str("user_id", event.UserID).
		Str("to", event.UserEmail).
		Float64("total", event.Total).
		Msg("[EMAIL] order confirmation sent")
}

// SendOrderFailed sends an order failure notification to the customer.
func (h *EmailHandler) SendOrderFailed(event events.OrderEvent) {
	h.log.Info().
		Str("type", "ORDER_FAILED").
		Str("order_id", event.OrderID).
		Str("user_id", event.UserID).
		Str("to", event.UserEmail).
		Msg("[EMAIL] order failed notification sent")
}

// SendOrderCancelled sends an order cancellation confirmation to the customer.
func (h *EmailHandler) SendOrderCancelled(event events.OrderEvent) {
	h.log.Info().
		Str("type", "ORDER_CANCELLED").
		Str("order_id", event.OrderID).
		Str("user_id", event.UserID).
		Str("to", event.UserEmail).
		Msg("[EMAIL] order cancelled notification sent")
}
