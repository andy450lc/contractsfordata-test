package boot

import (
	"log/slog"
	"net/http"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/mail"
)

// NewMailer builds the mail sender from config. A ResendMailer sends
// through Resend when RESEND_API_KEY is set. A LogMailer preserves the
// existing local platform-mail behavior otherwise; template delivery checks
// its availability and refuses to present a logged message as a successful
// attachment send.
func NewMailer(cfg config.AppConfig, logger *slog.Logger) mail.Mailer {
	if cfg.ResendAPIKey == "" || cfg.EmailFrom == "" {
		logger.Info("mail: provider configuration incomplete, logging platform mail locally")
		return &mail.LogMailer{}
	}

	baseURL := "https://api.resend.com"
	client := &http.Client{Timeout: mail.SendTimeout}
	return mail.NewResendMailer(client, baseURL, cfg.ResendAPIKey, cfg.EmailFrom)
}
