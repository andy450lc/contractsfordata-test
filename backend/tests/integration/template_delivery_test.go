package integration

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/mail"
)

const docxMIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

func TestTemplateDownloadRequiresOriginAndAcknowledgment(t *testing.T) {
	app := startApp(t, nil)

	resp, body := postTemplate(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":true}`, "", "")
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "forbidden_origin") {
		t.Fatalf("missing origin = %d %s", resp.StatusCode, body)
	}
	resp, body = postTemplate(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":false}`, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("false acknowledgment = %d %s", resp.StatusCode, body)
	}
	resp, body = postTemplate(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":true,"unknown":1}`, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("unknown property = %d %s", resp.StatusCode, body)
	}
}

func TestTemplateDownloadReturnsWordPackageHeaders(t *testing.T) {
	app := startApp(t, nil)

	resp, body := postTemplateBytes(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":true}`, "http://allowed.example", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if resp.Header.Get("Content-Type") != docxMIME {
		t.Errorf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("Content-Disposition") != `attachment; filename="content-development-agreement-template.docx"` {
		t.Errorf("Content-Disposition = %q", resp.Header.Get("Content-Disposition"))
	}
	if !bytes.HasPrefix(body, []byte("PK")) {
		t.Error("response is not a ZIP package")
	}
}

func TestTemplateEmailUsesSameArtifactAndReportsFailures(t *testing.T) {
	fm := &fakeMailer{}
	app := startApp(t, nil, fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer)))))

	download, downloadBody := postTemplateBytes(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":true}`, "http://allowed.example", "")
	if download.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d", download.StatusCode)
	}
	resp, body := postTemplate(t, app.BaseURL+"/v1/template-deliveries/email", `{"acknowledged":true,"email":"Reader@Example.COM"}`, "http://allowed.example", "delivery-key-123")
	if resp.StatusCode != http.StatusAccepted || body != `{"accepted":true}`+"\n" {
		t.Fatalf("email response = %d %s", resp.StatusCode, body)
	}
	messages := fm.Sent()
	if len(messages) != 1 || messages[0].To != "reader@example.com" || messages[0].IdempotencyKey != "delivery-key-123" {
		t.Fatalf("messages = %#v", messages)
	}
	if len(messages[0].Attachments) != 1 || !bytes.Equal(messages[0].Attachments[0].Content, downloadBody) || messages[0].Attachments[0].ContentType != docxMIME {
		t.Error("email attachment differs from direct download")
	}

	fm.FailWith(errors.New("provider rejected request"))
	resp, body = postTemplate(t, app.BaseURL+"/v1/template-deliveries/email", `{"acknowledged":true,"email":"reader@example.com"}`, "http://allowed.example", "delivery-key-456")
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "email_delivery_unavailable") {
		t.Fatalf("provider failure = %d %s", resp.StatusCode, body)
	}
}

func TestTemplateEmailRejectsUnconfiguredMailerAndMissingKey(t *testing.T) {
	app := startApp(t, nil)

	resp, body := postTemplate(t, app.BaseURL+"/v1/template-deliveries/email", `{"acknowledged":true,"email":"reader@example.com"}`, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("missing key = %d %s", resp.StatusCode, body)
	}
	resp, body = postTemplate(t, app.BaseURL+"/v1/template-deliveries/email", `{"acknowledged":true,"email":"reader@example.com"}`, "http://allowed.example", "delivery-key-789")
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "email_delivery_unavailable") {
		t.Fatalf("unconfigured mail = %d %s", resp.StatusCode, body)
	}
}

func TestTemplateCORSAllowsIdempotencyAndExposesFilename(t *testing.T) {
	app := startApp(t, nil)
	req, _ := http.NewRequest(http.MethodOptions, app.BaseURL+"/v1/template-deliveries/email", nil)
	req.Header.Set("Origin", "http://allowed.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.Contains(strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers")), "idempotency-key") {
		t.Errorf("allow headers = %q", resp.Header.Get("Access-Control-Allow-Headers"))
	}

	// Access-Control-Expose-Headers only applies to the actual response,
	// not the OPTIONS preflight, so it is checked against a real request.
	download, _ := postTemplateBytes(t, app.BaseURL+"/v1/template-deliveries/download", `{"acknowledged":true}`, "http://allowed.example", "")
	if !strings.Contains(strings.ToLower(download.Header.Get("Access-Control-Expose-Headers")), "content-disposition") {
		t.Errorf("expose headers = %q", download.Header.Get("Access-Control-Expose-Headers"))
	}
}

func postTemplate(t *testing.T, target, body, origin, key string) (*http.Response, string) {
	t.Helper()
	resp, data := postTemplateBytes(t, target, body, origin, key)
	return resp, string(data)
}

func postTemplateBytes(t *testing.T, target, body, origin, key string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}
