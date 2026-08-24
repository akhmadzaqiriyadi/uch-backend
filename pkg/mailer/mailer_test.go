package mailer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gozaq/config"
)

func TestSanitizeHeader(t *testing.T) {
	t.Run("Valid header", func(t *testing.T) {
		clean, err := SanitizeHeader("Welcome to Gozaq!")
		require.NoError(t, err)
		assert.Equal(t, "Welcome to Gozaq!", clean)
	})

	t.Run("CRLF Injection Attempt with newline", func(t *testing.T) {
		_, err := SanitizeHeader("Subject\nBcc: victim@example.com")
		assert.ErrorIs(t, err, ErrHeaderInjection)
	})

	t.Run("CRLF Injection Attempt with carriage return", func(t *testing.T) {
		_, err := SanitizeHeader("Subject\r\nTo: hacker@example.com")
		assert.ErrorIs(t, err, ErrHeaderInjection)
	})

	t.Run("URL encoded CRLF injection", func(t *testing.T) {
		_, err := SanitizeHeader("Subject%0aBcc: evil@example.com")
		assert.ErrorIs(t, err, ErrHeaderInjection)
	})
}

func TestValidateRecipient(t *testing.T) {
	t.Run("Valid email", func(t *testing.T) {
		err := ValidateRecipient("user@example.com")
		assert.NoError(t, err)
	})

	t.Run("Invalid email format", func(t *testing.T) {
		err := ValidateRecipient("invalid-email-address")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("Email with CRLF injection", func(t *testing.T) {
		err := ValidateRecipient("user@example.com\r\nBcc: evil@example.com")
		assert.ErrorIs(t, err, ErrHeaderInjection)
	})
}

func TestLogMailer(t *testing.T) {
	mailer := NewLogMailer()
	ctx := context.Background()

	t.Run("SendWelcomeEmail", func(t *testing.T) {
		err := mailer.SendWelcomeEmail(ctx, "john@example.com", "John <script>alert(1)</script>")
		assert.NoError(t, err)
	})

	t.Run("SendPasswordResetEmail", func(t *testing.T) {
		err := mailer.SendPasswordResetEmail(ctx, "john@example.com", "http://localhost:8080/reset?token=123")
		assert.NoError(t, err)
	})

	t.Run("SendVerificationEmail", func(t *testing.T) {
		err := mailer.SendVerificationEmail(ctx, "john@example.com", "http://localhost:8080/verify-email?token=123")
		assert.NoError(t, err)
	})

	t.Run("SendWithInvalidRecipient", func(t *testing.T) {
		err := mailer.SendWelcomeEmail(ctx, "invalid-email", "John")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})
}

func TestNewMailer(t *testing.T) {
	t.Run("Fallback to LogMailer when SMTP Host is empty", func(t *testing.T) {
		cfg := &config.Config{}
		m := NewMailer(cfg)
		_, ok := m.(*LogMailer)
		assert.True(t, ok)
	})

	t.Run("Initialize SMTPMailer when SMTP Host is configured", func(t *testing.T) {
		cfg := &config.Config{
			SMTP: config.SMTPConfig{
				Host: "smtp.sendgrid.net",
				Port: 587,
			},
		}
		m := NewMailer(cfg)
		_, ok := m.(*SMTPMailer)
		assert.True(t, ok)
	})
}
