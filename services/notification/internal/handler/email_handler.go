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
	body := fmt.Sprintf(
		"Hi,<br><br>Your order <strong>%s</strong> has been confirmed.<br>Total: <strong>R%.2f</strong><br><br>Thank you for shopping with us.",
		event.OrderID, event.Total,
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
