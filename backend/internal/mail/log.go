package mail

import (
	"context"
	"strings"

	"github.com/pixels-two/sow/backend/internal/logging"
)

// LogMailer writes outgoing mail to the application log. No message
// leaves the process. Development environments with no RESEND_API_KEY
// use it.
type LogMailer struct{}

// Available reports that this mailer keeps messages inside the process.
func (m *LogMailer) Available() bool { return false }

// Send logs the message. The recipient is reduced to its domain so the
// full address never reaches the log.
func (m *LogMailer) Send(ctx context.Context, msg Message) error {
	logging.Info(ctx, "sending mail",
		"subject", msg.Subject,
		"recipient_domain", domainOf(msg.To),
	)
	return nil
}

// domainOf returns the part of an email address after "@", or "" when
// the address carries none.
func domainOf(address string) string {
	_, domain, found := strings.Cut(address, "@")
	if !found {
		return ""
	}
	return domain
}
