package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pixels-two/sow/backend/internal/logging"
)

// SendTimeout bounds one outbound call to the mail provider.
const SendTimeout = 10 * time.Second

// ResendMailer sends mail through the Resend API.
type ResendMailer struct {
	client  *http.Client
	baseURL string
	apiKey  string
	from    string
}

// NewResendMailer builds a ResendMailer that posts to baseURL with
// apiKey and sends every message from the from address.
func NewResendMailer(client *http.Client, baseURL string, apiKey string, from string) *ResendMailer {
	return &ResendMailer{client: client, baseURL: baseURL, apiKey: apiKey, from: from}
}

// Available reports that this mailer sends through an external provider.
func (m *ResendMailer) Available() bool { return true }

type resendMessage struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	Text        string             `json:"text"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

type resendAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type resendResponse struct {
	ID string `json:"id"`
}

// Send posts msg to Resend. Logs the provider message id on success.
// The recipient's full address never reaches the log.
func (m *ResendMailer) Send(ctx context.Context, msg Message) error {
	attachments := make([]resendAttachment, 0, len(msg.Attachments))
	for _, attachment := range msg.Attachments {
		attachments = append(attachments, resendAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     base64.StdEncoding.EncodeToString(attachment.Content),
		})
	}
	body, err := json.Marshal(resendMessage{
		From: m.from, To: []string{msg.To}, Subject: msg.Subject, Text: msg.Text, Attachments: attachments,
	})
	if err != nil {
		return fmt.Errorf("encoding mail message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building mail request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	if msg.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", msg.IdempotencyKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sending mail: resend responded %d", resp.StatusCode)
	}

	var result resendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding resend acceptance: %w", err)
	}
	if result.ID == "" {
		return fmt.Errorf("decoding resend acceptance: response has no message id")
	}
	logging.Info(ctx, "sent mail", "provider_message_id", result.ID)
	return nil
}
