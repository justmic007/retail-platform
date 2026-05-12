// Package mailer provides a thin Brevo wrapper for sending transactional emails.
// Shared across services — import once, use anywhere.
package mailer

import (
	"context"
	"fmt"

	brevo "github.com/getbrevo/brevo-go/lib"
)

// Mailer sends transactional emails via Brevo.
type Mailer struct {
	client    *brevo.APIClient
	apiKey    string
	fromEmail string
	fromName  string
}

// New creates a new Mailer.
func New(apiKey, fromEmail, fromName string) *Mailer {
	return &Mailer{
		client:    brevo.NewAPIClient(brevo.NewConfiguration()),
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// Send delivers a transactional email via Brevo.
// Returns an error if delivery fails — caller decides whether to log or propagate.
func (m *Mailer) Send(ctx context.Context, to, subject, htmlContent string) error {
	if to == "" {
		return fmt.Errorf("recipient address is empty")
	}

	ctx = context.WithValue(ctx, brevo.ContextAPIKey, brevo.APIKey{Key: m.apiKey})
	_, _, err := m.client.TransactionalEmailsApi.SendTransacEmail(
		ctx,
		brevo.SendSmtpEmail{
			Sender:      &brevo.SendSmtpEmailSender{Email: m.fromEmail, Name: m.fromName},
			To:          []brevo.SendSmtpEmailTo{{Email: to}},
			Subject:     subject,
			HtmlContent: htmlContent,
		},
	)
	if err != nil {
		return fmt.Errorf("brevo send: %w", err)
	}
	return nil
}
