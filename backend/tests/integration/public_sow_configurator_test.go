package integration

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/mail"
)

const validPublicSOWBody = `{
	"acknowledged": true,
	"client_information": {"party_type": "buyer", "company_name": "Acme Robotics"},
	"answers": {
		"subcontracting": "buyer_written_consent",
		"jurisdiction_rules": "sow_country_appendix",
		"data_ip_liability_cap": "greater_of_amount_or_fee_multiple",
		"data_ip_liability_floor_amount": 50000,
		"data_ip_liability_multiplier": 3,
		"exclusivity": "non_exclusive"
	}
}`

func TestPublicSOWDownloadRequiresOriginAndAcknowledgment(t *testing.T) {
	app := startApp(t, nil)

	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", validPublicSOWBody, "", "")
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "forbidden_origin") {
		t.Fatalf("missing origin = %d %s", resp.StatusCode, body)
	}

	falseAck := `{"acknowledged":false,"client_information":{"party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"non_exclusive"}}`
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", falseAck, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("false acknowledgment = %d %s", resp.StatusCode, body)
	}

	unknownField := `{"acknowledged":true,"client_information":{"party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"non_exclusive"},"unknown":1}`
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", unknownField, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("unknown property = %d %s", resp.StatusCode, body)
	}
}

func TestPublicSOWDownloadRejectsMalformedAnswers(t *testing.T) {
	app := startApp(t, nil)

	malformedEnum := `{"acknowledged":true,"client_information":{"party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"sometimes","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"non_exclusive"}}`
	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", malformedEnum, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") ||
		!strings.Contains(body, "answers.subcontracting") {
		t.Fatalf("malformed enum = %d %s", resp.StatusCode, body)
	}

	missingScope := `{"acknowledged":true,"client_information":{"party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"exclusive_limited_field"}}`
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", missingScope, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") ||
		!strings.Contains(body, "answers.exclusivity_field_scope") {
		t.Fatalf("missing field scope = %d %s", resp.StatusCode, body)
	}
}

func TestPublicSOWDownloadRequiresClientInformation(t *testing.T) {
	app := startApp(t, nil)

	missingClientInfo := `{"acknowledged":true,"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"non_exclusive"}}`
	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", missingClientInfo, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") ||
		!strings.Contains(body, "client_information.company_name") || !strings.Contains(body, "client_information.party_type") {
		t.Fatalf("missing client_information = %d %s", resp.StatusCode, body)
	}

	emptyCompanyName := `{"acknowledged":true,"client_information":{"party_type":"buyer","company_name":""},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","exclusivity":"non_exclusive"}}`
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/download", emptyCompanyName, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "client_information.company_name") {
		t.Fatalf("empty company name = %d %s", resp.StatusCode, body)
	}
}

func TestPublicSOWDownloadReturnsWordPackageHeaders(t *testing.T) {
	app := startApp(t, nil)

	resp, body := postTemplateBytes(t, app.BaseURL+"/v1/public/sow-configurator/download", validPublicSOWBody, "http://allowed.example", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if resp.Header.Get("Content-Type") != docxMIME {
		t.Errorf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("Content-Disposition") != `attachment; filename="Robot_Sensor_Data_Content_Development_Agreement.docx"` {
		t.Errorf("Content-Disposition = %q", resp.Header.Get("Content-Disposition"))
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Access-Control-Expose-Headers")), "content-disposition") {
		t.Errorf("expose headers = %q", resp.Header.Get("Access-Control-Expose-Headers"))
	}
	if !bytes.HasPrefix(body, []byte("PK")) {
		t.Error("response is not a ZIP package")
	}
}

func TestPublicSOWEmailUsesSameArtifactAndReportsProviderFailure(t *testing.T) {
	fm := &fakeMailer{}
	app := startApp(t, nil, fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer)))))

	download, downloadBody := postTemplateBytes(t, app.BaseURL+"/v1/public/sow-configurator/download", validPublicSOWBody, "http://allowed.example", "")
	if download.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d", download.StatusCode)
	}

	emailBody := `{"acknowledged":true,"email":"Reader@Example.COM","client_information":{"party_type":"buyer","company_name":"Acme Robotics"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","data_ip_liability_floor_amount":50000,"data_ip_liability_multiplier":3,"exclusivity":"non_exclusive"}}`
	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", emailBody, "http://allowed.example", "public-key-123")
	if resp.StatusCode != http.StatusAccepted || body != `{"accepted":true}`+"\n" {
		t.Fatalf("email response = %d %s", resp.StatusCode, body)
	}
	messages := fm.Sent()
	if len(messages) != 1 || messages[0].To != "reader@example.com" || messages[0].IdempotencyKey != "public-key-123" {
		t.Fatalf("messages = %#v", messages)
	}
	if len(messages[0].Attachments) != 1 || !bytes.Equal(messages[0].Attachments[0].Content, downloadBody) || messages[0].Attachments[0].ContentType != docxMIME {
		t.Error("email attachment differs from direct download")
	}

	fm.FailWith(errors.New("provider rejected request"))
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", emailBody, "http://allowed.example", "public-key-456")
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "email_delivery_unavailable") {
		t.Fatalf("provider failure = %d %s", resp.StatusCode, body)
	}
}

func TestPublicSOWEmailRejectsUnconfiguredMailerAndMissingKey(t *testing.T) {
	app := startApp(t, nil)

	emailBody := `{"acknowledged":true,"email":"reader@example.com","client_information":{"party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","data_ip_liability_floor_amount":50000,"data_ip_liability_multiplier":3,"exclusivity":"non_exclusive"}}`

	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", emailBody, "http://allowed.example", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "validation_failed") {
		t.Fatalf("missing key = %d %s", resp.StatusCode, body)
	}
	resp, body = postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", emailBody, "http://allowed.example", "public-key-789")
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "email_delivery_unavailable") {
		t.Fatalf("unconfigured mail = %d %s", resp.StatusCode, body)
	}
}

func TestPublicSOWEmailRejectsDeliveryPreferenceMismatch(t *testing.T) {
	fm := &fakeMailer{}
	app := startApp(t, nil, fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer)))))

	// delivery_preference says "download" but this hits the /email endpoint.
	mismatched := `{"acknowledged":true,"email":"reader@example.com","client_information":{"delivery_preference":"download","party_type":"buyer","company_name":"Acme"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","data_ip_liability_floor_amount":50000,"data_ip_liability_multiplier":3,"exclusivity":"non_exclusive"}}`
	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", mismatched, "http://allowed.example", "public-key-mismatch")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "client_information.delivery_preference") {
		t.Fatalf("delivery preference mismatch = %d %s", resp.StatusCode, body)
	}
	if len(fm.Sent()) != 0 {
		t.Error("mailer received a message for a rejected request")
	}
}

// signalingMailer wraps fakeMailer to signal the instant Send is
// entered, before it blocks on the gate. That is the point at which the
// service has already registered the delivery as pending, which is what
// a second request needs to race against deterministically.
type signalingMailer struct {
	*fakeMailer
	entered chan struct{}
	once    sync.Once
}

func newSignalingMailer() *signalingMailer {
	return &signalingMailer{fakeMailer: &fakeMailer{}, entered: make(chan struct{})}
}

func (m *signalingMailer) Send(ctx context.Context, msg mail.Message) error {
	m.once.Do(func() { close(m.entered) })
	return m.fakeMailer.Send(ctx, msg)
}

func TestPublicSOWEmailIdempotencyConflict(t *testing.T) {
	fm := newSignalingMailer()
	app := startApp(t, nil, fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer)))))

	openGate := fm.Gate()
	t.Cleanup(openGate)

	firstBody := `{"acknowledged":true,"email":"reader@example.com","client_information":{"party_type":"buyer","company_name":"Acme Robotics"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","data_ip_liability_floor_amount":50000,"data_ip_liability_multiplier":3,"exclusivity":"non_exclusive"}}`
	secondBody := `{"acknowledged":true,"email":"reader@example.com","client_information":{"party_type":"buyer","company_name":"Different Company"},"answers":{"subcontracting":"buyer_written_consent","jurisdiction_rules":"sow_country_appendix","data_ip_liability_cap":"greater_of_amount_or_fee_multiple","data_ip_liability_floor_amount":50000,"data_ip_liability_multiplier":3,"exclusivity":"non_exclusive"}}`

	firstDone := make(chan struct{})
	var firstStatus int
	go func() {
		defer close(firstDone)
		resp, _ := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", firstBody, "http://allowed.example", "conflict-key")
		firstStatus = resp.StatusCode
	}()

	select {
	case <-fm.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request never reached the mailer")
	}

	resp, body := postTemplate(t, app.BaseURL+"/v1/public/sow-configurator/email", secondBody, "http://allowed.example", "conflict-key")
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, "idempotency_conflict") {
		t.Fatalf("conflicting key = %d %s", resp.StatusCode, body)
	}

	openGate()
	<-firstDone
	if firstStatus != http.StatusAccepted {
		t.Fatalf("first request status = %d", firstStatus)
	}
}
