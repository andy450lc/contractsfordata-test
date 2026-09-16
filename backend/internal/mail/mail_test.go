package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pixels-two/sow/backend/internal/logging"
)

func TestLogMailerSendLogsSubjectAndDomainOnly(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter("dev", &buf)
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mailer := &LogMailer{}
	err := mailer.Send(ctx, Message{To: "jane.doe@example.com", Subject: "Reset your SoW password", Text: "body"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	line := buf.String()
	if !strings.Contains(line, "Reset your SoW password") {
		t.Errorf("log line = %q, want the subject", line)
	}
	if !strings.Contains(line, "example.com") {
		t.Errorf("log line = %q, want the recipient's domain", line)
	}
	if strings.Contains(line, "jane.doe@example.com") {
		t.Errorf("log line = %q, must not carry the full recipient address", line)
	}
}

func TestResendMailerSendPostsMessage(t *testing.T) {
	var gotAuth string
	var gotIdempotencyKey string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotIdempotencyKey = r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer server.Close()

	mailer := NewResendMailer(server.Client(), server.URL, "key_test", "noreply@sow.example")
	err := mailer.Send(context.Background(), Message{
		To:             "jane@example.com",
		Subject:        "Hello",
		Text:           "body text",
		IdempotencyKey: "delivery-key-123",
		Attachments: []Attachment{{
			Filename:    "agreement.docx",
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			Content:     []byte("docx bytes"),
		}},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAuth != "Bearer key_test" {
		t.Errorf("Authorization = %q, want Bearer key_test", gotAuth)
	}
	if gotIdempotencyKey != "delivery-key-123" {
		t.Errorf("Idempotency-Key = %q, want delivery-key-123", gotIdempotencyKey)
	}
	if gotBody["from"] != "noreply@sow.example" {
		t.Errorf("from = %v, want noreply@sow.example", gotBody["from"])
	}
	if gotBody["subject"] != "Hello" {
		t.Errorf("subject = %v, want Hello", gotBody["subject"])
	}
	if gotBody["text"] != "body text" {
		t.Errorf("text = %v, want body text", gotBody["text"])
	}
	to, ok := gotBody["to"].([]any)
	if !ok || len(to) != 1 || to[0] != "jane@example.com" {
		t.Errorf("to = %v, want [jane@example.com]", gotBody["to"])
	}
	attachments, ok := gotBody["attachments"].([]any)
	if !ok || len(attachments) != 1 {
		t.Fatalf("attachments = %#v, want one attachment", gotBody["attachments"])
	}
	attachment := attachments[0].(map[string]any)
	if attachment["filename"] != "agreement.docx" || attachment["content_type"] != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("attachment metadata = %#v", attachment)
	}
	if attachment["content"] != "ZG9jeCBieXRlcw==" {
		t.Errorf("attachment content = %v, want Base64 bytes", attachment["content"])
	}
}

func TestResendMailerSendRejectsMalformedSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	mailer := NewResendMailer(server.Client(), server.URL, "key_test", "noreply@sow.example")
	err := mailer.Send(context.Background(), Message{To: "jane@example.com", Subject: "Hello", Text: "body"})
	if err == nil || !strings.Contains(err.Error(), "message id") {
		t.Fatalf("Send() error = %v, want missing message id error", err)
	}
}

func TestResendMailerSendErrorsOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"invalid recipient"}`))
	}))
	defer server.Close()

	mailer := NewResendMailer(server.Client(), server.URL, "key_test", "noreply@sow.example")
	err := mailer.Send(context.Background(), Message{To: "jane@example.com", Subject: "Hello", Text: "body"})
	if err == nil {
		t.Fatal("Send() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("error = %q, want it to name the status 422", err.Error())
	}
	if strings.Contains(err.Error(), "invalid recipient") {
		t.Errorf("error = %q, must not carry the response body", err.Error())
	}
}

func TestPasswordResetMessage(t *testing.T) {
	expiresAt := time.Date(2026, time.September, 10, 15, 4, 0, 0, time.UTC)
	msg := PasswordResetMessage("https://sow.example", "jane@example.com", "https://sow.example/reset-password?token=prt_1", expiresAt)

	if msg.To != "jane@example.com" {
		t.Errorf("To = %q, want jane@example.com", msg.To)
	}
	if msg.Subject != "Reset your SoW password" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "Reset your SoW password")
	}
	for _, want := range []string{
		"Someone asked to reset the password for this SoW account.",
		"https://sow.example/reset-password?token=prt_1",
		"The link works once and expires at Thu, 10 Sep 2026 15:04:00 UTC.",
		"If you did not ask for this, you can ignore this email. Your password\nstays the same.",
	} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("Text does not contain %q, got %q", want, msg.Text)
		}
	}
}

func TestAccountExistsMessage(t *testing.T) {
	msg := AccountExistsMessage("https://sow.example", "jane@example.com")

	if msg.To != "jane@example.com" {
		t.Errorf("To = %q, want jane@example.com", msg.To)
	}
	if msg.Subject != "You already have a SoW account" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "You already have a SoW account")
	}
	for _, want := range []string{
		"Someone tried to create a SoW account with this email address, but one\nalready exists.",
		"Sign in here:\nhttps://sow.example/",
		"Forgot your password? Reset it here:\nhttps://sow.example/forgot-password",
		"If this was not you, you can ignore this email.",
	} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("Text does not contain %q, got %q", want, msg.Text)
		}
	}
}
