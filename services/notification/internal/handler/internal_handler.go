// Package handler — internal notifications (warehouse staff, ops team).
package handler

import (
	"context"
	"fmt"

	brevo "github.com/getbrevo/brevo-go/lib"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// InternalHandler sends internal alerts to warehouse staff via Brevo.
type InternalHandler struct {
	client         *brevo.APIClient
	apiKey         string
	fromEmail      string
	fromName       string
	warehouseEmail string
	log            *logger.Logger
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(apiKey, fromEmail, fromName, warehouseEmail string, log *logger.Logger) *InternalHandler {
	return &InternalHandler{
		client:         brevo.NewAPIClient(brevo.NewConfiguration()),
		apiKey:         apiKey,
		fromEmail:      fromEmail,
		fromName:       fromName,
		warehouseEmail: warehouseEmail,
		log:            log,
	}
}

// SendLowStockAlert sends a low stock alert email to warehouse staff.
func (h *InternalHandler) SendLowStockAlert(ctx context.Context, event events.StockEvent) {
	subject := fmt.Sprintf("Low stock alert: %s", event.ProductName)
	body := fmt.Sprintf(
		"<strong>Low Stock Alert</strong><br><br>Product: <strong>%s</strong><br>Product ID: %s<br>Current stock: <strong>%d units</strong><br><br>Please restock as soon as possible.",
		event.ProductName, event.ProductID, event.StockLevel,
	)

	ctx = context.WithValue(ctx, brevo.ContextAPIKey, brevo.APIKey{Key: h.apiKey})
	_, _, err := h.client.TransactionalEmailsApi.SendTransacEmail(
		ctx,
		brevo.SendSmtpEmail{
			Sender:      &brevo.SendSmtpEmailSender{Email: h.fromEmail, Name: h.fromName},
			To:          []brevo.SendSmtpEmailTo{{Email: h.warehouseEmail}},
			Subject:     subject,
			HtmlContent: body,
		},
	)
	if err != nil {
		h.log.Error().Err(err).Str("to", h.warehouseEmail).Msg("failed to send low stock alert via Brevo")
	}

	h.log.Warn().
		Str("type", "LOW_STOCK_ALERT").
		Str("to", h.warehouseEmail).
		Str("product_id", event.ProductID).
		Str("product_name", event.ProductName).
		Int("stock_level", event.StockLevel).
		Msg("[EMAIL] low stock alert sent to warehouse")
}
