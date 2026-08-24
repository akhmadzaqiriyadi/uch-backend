package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"log/slog"
	"net/mail"
	"net/smtp"
	"strings"

	"gozaq/config"
)

var (
	ErrHeaderInjection = errors.New("potential email header injection detected (CRLF forbidden)")
	ErrInvalidEmail    = errors.New("invalid email address format")
)

type Mailer interface {
	SendWelcomeEmail(ctx context.Context, to, name string) error
	SendPasswordResetEmail(ctx context.Context, to, resetURL string) error
	SendVerificationEmail(ctx context.Context, to, verifyURL string) error
}

// SanitizeHeader prevents SMTP Header Injection / CRLF attacks
func SanitizeHeader(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") || strings.Contains(value, "%0d") || strings.Contains(value, "%0a") {
		return "", ErrHeaderInjection
	}
	return strings.TrimSpace(value), nil
}

// ValidateRecipient validates email format and ensures no CRLF characters
func ValidateRecipient(email string) error {
	if _, err := SanitizeHeader(email); err != nil {
		return err
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEmail, err)
	}
	return nil
}

// SMTPMailer sends emails via authenticated SMTP server with TLS
type SMTPMailer struct {
	cfg *config.Config
}

func NewSMTPMailer(cfg *config.Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// NewMailer creates an SMTPMailer or LogMailer based on driver configuration
func NewMailer(cfg *config.Config) Mailer {
	if cfg.SMTP.Driver == "log" || cfg.SMTP.Driver == "mock" {
		slog.Info("📧 LogMailer mock driver active (stdout preview mode)")
		return NewLogMailer()
	}

	if cfg.SMTP.Host != "" {
		slog.Info("📧 Production SMTP Mailer initialized", slog.String("host", cfg.SMTP.Host), slog.Int("port", cfg.SMTP.Port))
		return NewSMTPMailer(cfg)
	}

	slog.Info("📧 SMTP host not configured, falling back to LogMailer mock driver")
	return NewLogMailer()
}

var welcomeTemplate = htmlTemplate.Must(htmlTemplate.New("welcome").Parse(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; color: #333;">
  <div style="background: #ffffff; padding: 30px; border-radius: 8px; max-width: 600px; margin: auto; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
    <h2 style="color: #6366f1; margin-top: 0;">Welcome to {{.AppName}}, {{.Name}}! 🚀</h2>
    <p>Thank you for joining. Your account is now fully active and ready to use.</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;" />
    <p style="color: #6b7280; font-size: 12px;">This is an automated system email from {{.AppName}}.</p>
  </div>
</body>
</html>`))

var resetTemplate = htmlTemplate.Must(htmlTemplate.New("reset").Parse(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; color: #333;">
  <div style="background: #ffffff; padding: 30px; border-radius: 8px; max-width: 600px; margin: auto; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
    <h2 style="color: #dc2626; margin-top: 0;">Password Reset Request 🔐</h2>
    <p>We received a request to reset your password for your {{.AppName}} account.</p>
    <p>Click the button below to set a new password. This secure link is valid for <strong>15 minutes</strong>:</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.ResetURL}}" style="background-color: #6366f1; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">Reset Password</a>
    </div>
    <p style="font-size: 13px; color: #4b5563;">Or copy this URL into your browser:<br/><a href="{{.ResetURL}}" style="color: #6366f1; word-break: break-all;">{{.ResetURL}}</a></p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;" />
    <p style="color: #9ca3af; font-size: 12px;">If you did not request a password reset, please ignore this email. Your password will remain unchanged.</p>
  </div>
</body>
</html>`))

var verifyTemplate = htmlTemplate.Must(htmlTemplate.New("verify").Parse(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; color: #333;">
  <div style="background: #ffffff; padding: 30px; border-radius: 8px; max-width: 600px; margin: auto; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
    <h2 style="color: #10b981; margin-top: 0;">Verify Your Email Address ✉️</h2>
    <p>Thank you for registering at {{.AppName}}. Please confirm your email address to activate all account features.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.VerifyURL}}" style="background-color: #10b981; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">Verify Email</a>
    </div>
    <p style="font-size: 13px; color: #4b5563;">Or copy this link:<br/><a href="{{.VerifyURL}}" style="color: #10b981; word-break: break-all;">{{.VerifyURL}}</a></p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;" />
    <p style="color: #9ca3af; font-size: 12px;">This link is valid for 24 hours.</p>
  </div>
</body>
</html>`))

func (m *SMTPMailer) SendWelcomeEmail(ctx context.Context, to, name string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	sanitizedName, _ := SanitizeHeader(name)

	var body bytes.Buffer
	err := welcomeTemplate.Execute(&body, map[string]string{
		"AppName": m.cfg.App.Name,
		"Name":    sanitizedName,
	})
	if err != nil {
		return fmt.Errorf("failed to render welcome email template: %w", err)
	}

	return m.sendEmail(ctx, to, fmt.Sprintf("Welcome to %s! 🚀", m.cfg.App.Name), body.String())
}

func (m *SMTPMailer) SendPasswordResetEmail(ctx context.Context, to, resetURL string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	var body bytes.Buffer
	err := resetTemplate.Execute(&body, map[string]string{
		"AppName":  m.cfg.App.Name,
		"ResetURL": resetURL,
	})
	if err != nil {
		return fmt.Errorf("failed to render reset email template: %w", err)
	}

	return m.sendEmail(ctx, to, fmt.Sprintf("%s - Password Reset Request", m.cfg.App.Name), body.String())
}

func (m *SMTPMailer) SendVerificationEmail(ctx context.Context, to, verifyURL string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	var body bytes.Buffer
	err := verifyTemplate.Execute(&body, map[string]string{
		"AppName":   m.cfg.App.Name,
		"VerifyURL": verifyURL,
	})
	if err != nil {
		return fmt.Errorf("failed to render verification email template: %w", err)
	}

	return m.sendEmail(ctx, to, fmt.Sprintf("%s - Verify Your Email", m.cfg.App.Name), body.String())
}

func (m *SMTPMailer) sendEmail(_ context.Context, to, subject, htmlBody string) error {
	cleanSubject, err := SanitizeHeader(subject)
	if err != nil {
		return err
	}

	fromHeader := fmt.Sprintf("%s <%s>", m.cfg.SMTP.FromName, m.cfg.SMTP.From)
	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = to
	headers["Subject"] = cleanSubject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&message, "%s: %s\r\n", k, v)
	}
	message.WriteString("\r\n" + htmlBody)

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTP.Host, m.cfg.SMTP.Port)
	var auth smtp.Auth
	if m.cfg.SMTP.User != "" && m.cfg.SMTP.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTP.User, m.cfg.SMTP.Password, m.cfg.SMTP.Host)
	}

	if err := smtp.SendMail(addr, auth, m.cfg.SMTP.From, []string{to}, []byte(message.String())); err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}

// LogMailer logs emails in stdout for development & testing
type LogMailer struct{}

func NewLogMailer() *LogMailer {
	return &LogMailer{}
}

func (m *LogMailer) SendWelcomeEmail(_ context.Context, to, name string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	var body bytes.Buffer
	_ = welcomeTemplate.Execute(&body, map[string]string{
		"AppName": "Gozaq",
		"Name":    name,
	})

	slog.Info("📧 [MAILER DISPATCH] Welcome Email Sent",
		slog.String("to", to),
		slog.String("subject", "Welcome to Gozaq!"),
		slog.String("preview_html", body.String()),
	)
	return nil
}

func (m *LogMailer) SendPasswordResetEmail(_ context.Context, to, resetURL string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	slog.Info("📧 [MAILER DISPATCH] Password Reset Email Sent",
		slog.String("to", to),
		slog.String("reset_url", resetURL),
	)
	return nil
}

func (m *LogMailer) SendVerificationEmail(_ context.Context, to, verifyURL string) error {
	if err := ValidateRecipient(to); err != nil {
		return err
	}

	slog.Info("📧 [MAILER DISPATCH] Verification Email Sent",
		slog.String("to", to),
		slog.String("verify_url", verifyURL),
	)
	return nil
}
