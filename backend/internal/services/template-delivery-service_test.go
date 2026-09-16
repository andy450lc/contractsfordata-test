package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/mail"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

type captureDeliveryMailer struct {
	mu        sync.Mutex
	messages  []mail.Message
	err       error
	available bool
	gate      chan struct{}
}

type failingDocumentGenerator struct{ err error }

func (g failingDocumentGenerator) Generate(*dto.ContractConfiguration) (DocumentArtifact, error) {
	return DocumentArtifact{}, g.err
}

func (m *captureDeliveryMailer) Available() bool { return m.available }

func (m *captureDeliveryMailer) Send(_ context.Context, message mail.Message) error {
	if m.gate != nil {
		<-m.gate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
	return m.err
}

func (m *captureDeliveryMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestDocumentGeneratorBuildsDeterministicOOXML(t *testing.T) {
	generator := NewDocumentGenerator()
	configuration := validContractConfiguration()

	first, err := generator.Generate(&configuration)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	second, err := generator.Generate(&configuration)
	if err != nil {
		t.Fatalf("Generate() second error = %v", err)
	}
	if !bytes.Equal(first.Bytes, second.Bytes) {
		t.Error("equal configurations produced different bytes")
	}
	if first.MIMEType != DocumentMIMEType || first.Filename != "sow-7-pixels2-india-stereo-pilot.docx" {
		t.Errorf("artifact metadata = %q, %q", first.MIMEType, first.Filename)
	}

	reader, err := zip.NewReader(bytes.NewReader(first.Bytes), int64(len(first.Bytes)))
	if err != nil {
		t.Fatalf("opening generated package: %v", err)
	}
	entries := map[string]string{}
	for _, file := range reader.File {
		rc, openErr := file.Open()
		if openErr != nil {
			t.Fatalf("opening %s: %v", file.Name, openErr)
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			t.Fatalf("reading %s: %v", file.Name, readErr)
		}
		entries[file.Name] = string(data)
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/styles.xml"} {
		if _, ok := entries[name]; !ok {
			t.Errorf("package is missing %s", name)
		}
	}
	for name, content := range entries {
		if err := parseXML(content); err != nil {
			t.Errorf("%s is not well-formed XML: %v", name, err)
		}
	}
	if !strings.Contains(entries["word/styles.xml"], `w:ascii="Helvetica"`) || !strings.Contains(entries["word/styles.xml"], `w:sz w:val="21"`) {
		t.Error("styles do not preserve Helvetica 10.5 point body text")
	}
	if !strings.Contains(entries["word/header1.xml"], `string="DRAFT"`) || !strings.Contains(entries["word/footer1.xml"], "NUMPAGES") {
		t.Error("package does not contain the draft watermark and page footer")
	}
	document := entries["word/document.xml"]
	for _, want := range []string{
		"CONTENT DEVELOPMENT AGREEMENT",
		"Developer will source, coordinate, engage, supervise",
		"The engagement and all Deliverables are exclusive.",
		"There is no deposit, advance payment, or hardware contribution.",
		"Every technical item is a condition of acceptance.",
		"An “accepted usable hour” is an hour of unique footage",
		"governed by Delaware law",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("document does not contain %q", want)
		}
	}
	for _, bad := range []string{"{{", "}}", "undefined", ">null<", "NaN"} {
		if strings.Contains(document, bad) {
			t.Errorf("document contains bad token %q", bad)
		}
	}
}

func TestDocumentGeneratorSanitizesFilenameAndXMLCharacters(t *testing.T) {
	configuration := validContractConfiguration()
	configuration.Steps.Agreement.Title = ` Résumé / Q4:*? "Launch" `
	configuration.Steps.Scope.Notes = "safe\x00text\ufffeafter"

	artifact, err := NewDocumentGenerator().Generate(&configuration)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Filename != "sow-7-r-sum-q4-launch.docx" {
		t.Errorf("filename = %q", artifact.Filename)
	}
	document := documentXML(t, artifact.Bytes)
	if strings.ContainsRune(document, '\x00') || strings.ContainsRune(document, '\ufffe') || !strings.Contains(document, "safetextafter") {
		t.Error("document did not strip XML-invalid characters")
	}
}

func TestDocumentGeneratorClauseMatrix(t *testing.T) {
	generator := NewDocumentGenerator()
	for agreementScope := 0; agreementScope < 3; agreementScope++ {
		for pricing := 0; pricing < 3; pricing++ {
			for technical := 0; technical < 3; technical++ {
				for acceptance := 0; acceptance < 3; acceptance++ {
					model := blankDocumentModel()
					model.Terms.Exclusive = agreementScope == 1
					model.Scope.AmbientAudio = agreementScope == 2
					model.Pricing.FirmDeadline = pricing == 1
					model.Pricing.DepositRequired = pricing == 2
					model.Environment.PlanRequired = technical == 1
					model.Technical.ConditionOfPayment = technical == 2
					model.Delivery.DeemedAcceptance = acceptance == 1
					model.Delivery.RejectedStaysDeveloperOwned = acceptance == 2

					artifact, err := generator.generateModel(model)
					if err != nil {
						t.Fatalf("matrix %d/%d/%d/%d: %v", agreementScope, pricing, technical, acceptance, err)
					}
					if _, err := zip.NewReader(bytes.NewReader(artifact.Bytes), int64(len(artifact.Bytes))); err != nil {
						t.Fatalf("matrix %d/%d/%d/%d invalid OOXML: %v", agreementScope, pricing, technical, acceptance, err)
					}
					text := documentXML(t, artifact.Bytes)
					assertClause(t, text, "The engagement and all Deliverables are exclusive.", agreementScope == 1)
					assertClause(t, text, "Ambient audio is permitted.", agreementScope == 2)
					assertClause(t, text, "firm obligation", pricing == 1)
					assertClause(t, text, "will pay a deposit", pricing == 2)
					assertClause(t, text, "Collection Plan.", technical == 1)
					assertClause(t, text, "condition of acceptance and payment", technical == 2)
					assertClause(t, text, "is deemed accepted only if", acceptance == 1)
					assertClause(t, text, "will remain Developer-owned", acceptance == 2)
				}
			}
		}
	}
}

func TestTemplateDeliveryRequiresAcknowledgmentAndConfiguredMailer(t *testing.T) {
	mailer := &captureDeliveryMailer{available: false}
	service := NewTemplateDeliveryService(NewDocumentGenerator(), mailer)
	configuration := validContractConfiguration()

	if _, err := service.Download(context.Background(), false, &configuration); err == nil {
		t.Fatal("Download() error = nil without acknowledgment")
	}
	_, err := service.Email(context.Background(), true, "reader@example.com", "delivery-key-123", &configuration)
	if !strings.Contains(err.Error(), common.ErrEmailDeliveryUnavailable.Error()) {
		t.Fatalf("Email() error = %v, want unavailable", err)
	}
}

func TestTemplateDeliveryMapsGenerationFailure(t *testing.T) {
	mailer := &captureDeliveryMailer{available: true}
	service := NewTemplateDeliveryService(failingDocumentGenerator{err: context.DeadlineExceeded}, mailer)
	configuration := validContractConfiguration()

	_, err := service.Download(context.Background(), true, &configuration)
	if !strings.Contains(err.Error(), common.ErrGenerationFailed.Error()) {
		t.Fatalf("Download() error = %v, want generation failure", err)
	}
	_, err = service.Email(context.Background(), true, "reader@example.com", "delivery-key-456", &configuration)
	if !strings.Contains(err.Error(), common.ErrGenerationFailed.Error()) {
		t.Fatalf("Email() error = %v, want generation failure", err)
	}
	if mailer.count() != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.count())
	}
}

func TestTemplateDeliveryValidatesNestedAndCrossFieldRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*dto.ContractConfiguration)
		field  string
	}{
		{"difficulty total", func(c *dto.ContractConfiguration) { c.Steps.Environment.Difficulty.Hard = 49 }, "difficulty"},
		{"deposit amount", func(c *dto.ContractConfiguration) {
			zero := float32(0)
			c.Steps.Pricing.DepositRequired = true
			c.Steps.Pricing.DepositAmount = &zero
		}, "deposit_amount"},
		{"final count", func(c *dto.ContractConfiguration) { c.Steps.Pricing.Milestones[0].IsFinal = true }, "milestones"},
		{"deadline ordering", func(c *dto.ContractConfiguration) { c.Steps.Pricing.Milestones[0].Deadline = dateValue("2026-09-14") }, "deadline"},
		{"detail party", func(c *dto.ContractConfiguration) { c.Steps.Counterparty.LegalName = "" }, "legal_name"},
		{"cc email", func(c *dto.ContractConfiguration) { c.Steps.Counterparty.CcEmails = "good@example.com, bad" }, "cc_emails"},
		{"country", func(c *dto.ContractConfiguration) { c.Steps.Scope.Regions = []string{"ZZ"} }, "regions"},
		{"currency", func(c *dto.ContractConfiguration) { c.Steps.Pricing.Currency = "ZZZ" }, "currency"},
	}
	service := NewTemplateDeliveryService(NewDocumentGenerator(), &captureDeliveryMailer{available: true})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := validContractConfiguration()
			test.mutate(&configuration)
			_, err := service.Download(context.Background(), true, &configuration)
			var validation *common.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Download() error = %v, want validation error", err)
			}
			found := false
			for _, detail := range validation.Details {
				if strings.Contains(detail.Field, test.field) {
					found = true
				}
			}
			if !found {
				t.Errorf("details = %#v, want field containing %q", validation.Details, test.field)
			}
		})
	}
}

func TestTemplateDeliveryEmailUsesArtifactAndSuppressesRapidDuplicate(t *testing.T) {
	gate := make(chan struct{})
	mailer := &captureDeliveryMailer{available: true, gate: gate}
	service := NewTemplateDeliveryService(NewDocumentGenerator(), mailer)
	configuration := validContractConfiguration()
	want, err := service.Download(context.Background(), true, &configuration)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, sendErr := service.Email(context.Background(), true, " Reader@Example.COM ", "delivery-key-123", &configuration)
			errs <- sendErr
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(gate)
	for range 2 {
		if sendErr := <-errs; sendErr != nil {
			t.Fatalf("Email() error = %v", sendErr)
		}
	}
	if mailer.count() != 1 {
		t.Fatalf("mailer calls = %d, want 1", mailer.count())
	}
	message := mailer.messages[0]
	if message.To != "reader@example.com" || message.IdempotencyKey != "delivery-key-123" {
		t.Errorf("message routing = %#v", message)
	}
	if len(message.Attachments) != 1 || !bytes.Equal(message.Attachments[0].Content, want.Bytes) {
		t.Error("email attachment differs from download artifact")
	}
}

func assertClause(t *testing.T, text string, phrase string, want bool) {
	t.Helper()
	if strings.Contains(text, phrase) != want {
		t.Errorf("contains %q = %v, want %v", phrase, strings.Contains(text, phrase), want)
	}
}

func documentXML(t *testing.T, data []byte) string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, _ := file.Open()
			body, _ := io.ReadAll(rc)
			_ = rc.Close()
			return string(body)
		}
	}
	t.Fatal("word/document.xml missing")
	return ""
}

func parseXML(value string) error {
	decoder := xml.NewDecoder(strings.NewReader(value))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func dateValue(value string) openapi_types.Date {
	parsed, _ := time.Parse("2006-01-02", value)
	return openapi_types.Date{Time: parsed}
}

func validContractConfiguration() dto.ContractConfiguration {
	return dto.ContractConfiguration{
		TemplateVersion: dto.ContractConfigurationTemplateVersionContentDevelopment1,
		Organization: dto.ContractOrganization{
			LegalName: "Sieve Inc.", EntityType: "corporation", IncorporationPlace: "Delaware", GoverningLaw: "Delaware",
			Address: "100 Market St, San Francisco, CA", ShortName: "Sieve", SignerName: "Ishan Dhawan", SignerTitle: "Chief of Staff",
			SignerEmail: "ishan@sieve.example", ContactPhone: "",
		},
		Steps: dto.ContractSteps{
			Agreement:    dto.AgreementTerms{SowNumber: 7, Title: "Pixels2 India Stereo Pilot", EffectiveDate: dateValue("2026-08-29"), Exclusive: true, MaterialsSummary: "content materials", IncidentNoticeHours: 24, CureDays: 30, ConvenienceNoticeDays: 30, RecordsRetentionYears: 2, LiabilityLookbackMonths: 12, ExcludedClaimsCapMultiplier: 2, DataClaimsCapFloor: 50000},
			Scope:        dto.ContractScope{DeliverableDescription: "stereo RGB footage", Regions: []string{"IN"}, VenueConstraint: "real businesses", TargetVolume: 300, Unit: "accepted usable hours", Deliverables: []dto.ContractTextRow{{Text: "Original stereo video"}}, Notes: ""},
			Pricing:      dto.ContractPricing{Currency: "USD", UnitFee: 20, DepositRequired: false, DepositAmount: nil, InvoicingCadence: "Weekly in arrears", PaymentTermsDays: 30, PaymentMethods: "ACH or wire transfer", InvoiceEmail: "ap@sievedata.com", FirmDeadline: false, Milestones: []dto.ContractMilestone{{Name: "First upload", Volume: "100 hours", Deadline: dateValue("2026-09-06"), Notes: "", IsFinal: false}, {Name: "Final delivery", Volume: "300 hours", Deadline: dateValue("2026-09-13"), Notes: "", IsFinal: true}}},
			Environment:  dto.ContractEnvironment{Verticals: []dto.ContractVertical{{Verticals: "Food service", Target: "100 hours", Details: "No residential settings"}}, Limits: []dto.ContractLimit{{Limit: "Minimum distinct physical sites", Value: "15", Unit: ""}}, Difficulty: dto.ContractDifficulty{Easy: 20, Medium: 30, Hard: 50}, DifficultyCaps: []dto.ContractDifficultyCap{}, PlanRequired: false, ChangeWindowHours: 24, ExtraProhibited: []dto.ContractTextRow{}, ExtraExcludedSettings: []dto.ContractTextRow{}},
			Technical:    dto.ContractTechnical{Requirements: []dto.ContractRequirement{{Requirement: "Modality", Specification: "Stereo RGB", Rejection: "Missing stream"}}, PreCollectionMaterials: []dto.ContractTextRow{}, ConditionOfPayment: false},
			Delivery:     dto.ContractDelivery{StorageLocation: "Sieve bucket", DeliveryCadence: "Daily", ManifestFormat: "CSV", ManifestFields: []dto.ContractTextRow{{Text: "file name"}}, SiteDataFields: []dto.ContractTextRow{}, IntegrityText: "SHA-256 checksums", ReviewWindowDays: 30, CorrectionDays: 5, DeemedAcceptance: false, RejectedStaysDeveloperOwned: false, DeletionWindowDays: 45},
			Counterparty: dto.ContractCounterparty{Mode: dto.ContractCounterpartyModeDetails, Email: "akshaj@pixels2.example", LegalName: "Pixels Two Corporation", EntityJurisdiction: "Delaware corporation", Address: "251 Little Falls Drive", ShortName: "Pixels", SignatoryName: "Akshaj Jain", SignatoryTitle: "President", CcEmails: ""},
		},
	}
}
