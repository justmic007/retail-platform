// Package handler contains the notification handlers.
package handler

import (
	"context"
	"fmt"

	brevo "github.com/getbrevo/brevo-go/lib"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// EmailHandler sends transactional emails to customers via Brevo.
type EmailHandler struct {
	client    *brevo.APIClient
	apiKey    string
	fromEmail string
	fromName  string
	log       *logger.Logger
}

// NewEmailHandler creates an EmailHandler wired to Brevo.
func NewEmailHandler(apiKey, fromEmail, fromName string, log *logger.Logger) *EmailHandler {
	return &EmailHandler{
		client:    brevo.NewAPIClient(brevo.NewConfiguration()),
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		log:       log,
	}
}

// SendOrderConfirmation sends an order confirmation email to the customer.
func (h *EmailHandler) SendOrderConfirmation(ctx context.Context, event events.OrderEvent) {
	subject := "Your order has been confirmed"

	itemRows := ""
	for _, item := range event.Items {
		itemRows += fmt.Sprintf(
			"<tr><td style='padding:4px 12px;border:1px solid #e5e7eb'>%s</td><td style='padding:4px 12px;text-align:center;border:1px solid #e5e7eb'>%d</td><td style='padding:4px 12px;text-align:right;border:1px solid #e5e7eb'>R%.2f</td><td style='padding:4px 12px;text-align:right;border:1px solid #e5e7eb'>R%.2f</td></tr>",
			item.ProductName, item.Quantity, item.UnitPrice, item.TotalPrice,
		)
	}

	var itemsTable string
	if itemRows != "" {
		itemsTable = fmt.Sprintf(`
			<table style="border-collapse:collapse;margin:16px 0;font-size:14px">
				<thead>
					<tr style="background:#f3f4f6">
						<th style="padding:6px 12px;text-align:left;border:1px solid #e5e7eb">Item</th>
						<th style="padding:6px 12px;text-align:center;border:1px solid #e5e7eb">Qty</th>
						<th style="padding:6px 12px;text-align:right;border:1px solid #e5e7eb">Unit Price</th>
						<th style="padding:6px 12px;text-align:right;border:1px solid #e5e7eb">Total</th>
					</tr>
				</thead>
				<tbody>%s</tbody>
			</table>`, itemRows)
	}

	body := fmt.Sprintf(
		"Hi,<br><br>Your order <strong>#%s</strong> has been confirmed.%s<br><strong>Total: R%.2f</strong><br><br>Thank you for shopping with us.",
		event.OrderID[:8], itemsTable, event.Total,
	)
	h.send(ctx, event.UserEmail, subject, body)
	h.log.Info().
		Str("type", "ORDER_CONFIRMATION").
		Str("order_id", event.OrderID).
		Str("to", event.UserEmail).
		Float64("total", event.Total).
		Msg("[EMAIL] order confirmation sent")
}

// SendOrderFailed sends an order failure notification to the customer.
func (h *EmailHandler) SendOrderFailed(ctx context.Context, event events.OrderEvent) {
	subject := "Your order could not be processed"
	body := fmt.Sprintf(
		"Hi,<br><br>Unfortunately your order <strong>%s</strong> could not be processed.<br>No payment has been taken. Please try again.",
		event.OrderID,
	)
	h.send(ctx, event.UserEmail, subject, body)
	h.log.Info().
		Str("type", "ORDER_FAILED").
		Str("order_id", event.OrderID).
		Str("to", event.UserEmail).
		Msg("[EMAIL] order failed notification sent")
}

// SendOrderCancelled sends an order cancellation confirmation to the customer.
func (h *EmailHandler) SendOrderCancelled(ctx context.Context, event events.OrderEvent) {
	subject := "Your order has been cancelled"
	body := fmt.Sprintf(
		"Hi,<br><br>Your order <strong>%s</strong> has been cancelled as requested.",
		event.OrderID,
	)
	h.send(ctx, event.UserEmail, subject, body)
	h.log.Info().
		Str("type", "ORDER_CANCELLED").
		Str("order_id", event.OrderID).
		Str("to", event.UserEmail).
		Msg("[EMAIL] order cancelled notification sent")
}

// send is the shared Brevo API call used by all email methods.
func (h *EmailHandler) send(ctx context.Context, to, subject, htmlContent string) {
	if to == "" {
		h.log.Warn().Str("subject", subject).Msg("skipping email — recipient address is empty")
		return
	}

	ctx = context.WithValue(ctx, brevo.ContextAPIKey, brevo.APIKey{Key: h.apiKey})
	_, _, err := h.client.TransactionalEmailsApi.SendTransacEmail(
		ctx,
		brevo.SendSmtpEmail{
			Sender:      &brevo.SendSmtpEmailSender{Email: h.fromEmail, Name: h.fromName},
			To:          []brevo.SendSmtpEmailTo{{Email: to}},
			Subject:     subject,
			HtmlContent: htmlContent,
		},
	)
	if err != nil {
		h.log.Error().Err(err).Str("to", to).Str("subject", subject).Msg("failed to send email via Brevo")
	}
}
