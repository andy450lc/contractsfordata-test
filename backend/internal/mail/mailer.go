// Package mail sends the platform's transactional email. LogMailer
// stands in for the real provider in development. ResendMailer sends
// through Resend in every deployed environment.
package mail

import "context"

// Message is one outgoing email.
type Message struct {
	To             string
	Subject        string
	Text           string
	IdempotencyKey string
	Attachments    []Attachment
}

// Attachment is one in-memory file sent with a message.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Mailer sends a message. Implementations never log the recipient's
// full address.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// AvailabilityReporter reports whether a mailer delivers beyond this process.
type AvailabilityReporter interface {
	Available() bool
}
