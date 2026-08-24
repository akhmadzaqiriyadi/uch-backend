package mailer

import (
	"context"
	"fmt"
	"log/slog"
)

type Mailer interface {
	SendWelcomeEmail(ctx context.Context, to, name string) error
	SendPasswordResetEmail(ctx context.Context, to, resetURL string) error
}

type LogMailer struct{}

func NewLogMailer() *LogMailer {
	return &LogMailer{}
}

func (m *LogMailer) SendWelcomeEmail(_ context.Context, to, name string) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
  <div style="background: #ffffff; padding: 20px; border-radius: 8px; max-width: 600px; margin: auto;">
    <h2 style="color: #6366f1;">Welcome to Gozaq, %s! 🚀</h2>
    <p>Thank you for registering. Your account is now active and ready to use.</p>
    <p style="color: #666; font-size: 12px;">This is an automated system email.</p>
  </div>
</body>
</html>`, name)

	slog.Info("📧 [MAILER DISPATCH] Welcome Email Sent",
		slog.String("to", to),
		slog.String("subject", "Welcome to Gozaq!"),
		slog.String("preview_html", htmlBody),
	)
	return nil
}

func (m *LogMailer) SendPasswordResetEmail(_ context.Context, to, resetURL string) error {
	slog.Info("📧 [MAILER DISPATCH] Password Reset Email Sent",
		slog.String("to", to),
		slog.String("reset_url", resetURL),
	)
	return nil
}
