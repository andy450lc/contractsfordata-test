package services

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pixels-two/sow/backend/internal/mail"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

func validPublicSOWAnswers() dto.PublicSOWAnswers {
	floorAmount := float32(50000)
	multiplier := float32(3)
	return dto.PublicSOWAnswers{
		Subcontracting:             dto.PublicSOWAnswersSubcontractingBuyerWrittenConsent,
		JurisdictionRules:          dto.PublicSOWAnswersJurisdictionRulesSowCountryAppendix,
		DataIpLiabilityCap:         dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple,
		DataIpLiabilityFloorAmount: &floorAmount,
		DataIpLiabilityMultiplier:  &multiplier,
		Exclusivity:                dto.PublicSOWAnswersExclusivityNonExclusive,
	}
}

func validPublicSOWClientInformation() dto.PublicSOWClientInformation {
	return dto.PublicSOWClientInformation{
		CompanyName: "Acme Robotics",
		PartyType:   dto.PublicSOWClientInformationPartyTypeSupplier,
	}
}

func TestPublicSOWGeneratorCoversAllEightyOneCombinations(t *testing.T) {
	generator := NewPublicSOWDocumentGenerator()
	subcontracting := []dto.PublicSOWAnswersSubcontracting{
		dto.PublicSOWAnswersSubcontractingNeverAllowed,
		dto.PublicSOWAnswersSubcontractingBuyerWrittenConsent,
		dto.PublicSOWAnswersSubcontractingNoticeWithBuyerObjection,
	}
	jurisdiction := []dto.PublicSOWAnswersJurisdictionRules{
		dto.PublicSOWAnswersJurisdictionRulesAgreementCountryTerms,
		dto.PublicSOWAnswersJurisdictionRulesSowCountryAppendix,
		dto.PublicSOWAnswersJurisdictionRulesSeparateCountryRider,
	}
	liability := []dto.PublicSOWAnswersDataIpLiabilityCap{
		dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple,
		dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly,
		dto.PublicSOWAnswersDataIpLiabilityCapNoSpecialCap,
	}
	exclusivity := []dto.PublicSOWAnswersExclusivity{
		dto.PublicSOWAnswersExclusivityFullyExclusiveGlobal,
		dto.PublicSOWAnswersExclusivityExclusiveLimitedField,
		dto.PublicSOWAnswersExclusivityNonExclusive,
	}

	for subcontractIndex, subcontractAnswer := range subcontracting {
		for jurisdictionIndex, jurisdictionAnswer := range jurisdiction {
			for liabilityIndex, liabilityAnswer := range liability {
				for exclusivityIndex, exclusivityAnswer := range exclusivity {
					scope := "autonomous warehouse robotics"
					floorAmount := float32(50000)
					multiplier := float32(3)
					answers := dto.PublicSOWAnswers{
						Subcontracting: subcontractAnswer, JurisdictionRules: jurisdictionAnswer,
						DataIpLiabilityCap: liabilityAnswer, Exclusivity: exclusivityAnswer,
					}
					if exclusivityAnswer == dto.PublicSOWAnswersExclusivityExclusiveLimitedField {
						answers.ExclusivityFieldScope = &scope
					}
					switch liabilityAnswer {
					case dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple:
						answers.DataIpLiabilityFloorAmount = &floorAmount
						answers.DataIpLiabilityMultiplier = &multiplier
					case dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly:
						answers.DataIpLiabilityMultiplier = &multiplier
					}
					configuration, err := validateAndNormalizePublicSOW(answers, validPublicSOWClientInformation(), "download")
					if err != nil {
						t.Fatalf("matrix %d/%d/%d/%d validation: %v", subcontractIndex, jurisdictionIndex, liabilityIndex, exclusivityIndex, err)
					}
					artifact, err := generator.Generate(configuration)
					if err != nil {
						t.Fatalf("matrix %d/%d/%d/%d generation: %v", subcontractIndex, jurisdictionIndex, liabilityIndex, exclusivityIndex, err)
					}
					if _, err := zip.NewReader(bytes.NewReader(artifact.Bytes), int64(len(artifact.Bytes))); err != nil {
						t.Fatalf("matrix %d/%d/%d/%d OOXML: %v", subcontractIndex, jurisdictionIndex, liabilityIndex, exclusivityIndex, err)
					}
					text := documentXML(t, artifact.Bytes)
					for _, fixed := range []string{"within 24 hours", "rights, permissions, consents", "general aggregate liability", "CONTENT DEVELOPMENT AGREEMENT"} {
						if !strings.Contains(text, fixed) {
							t.Errorf("matrix %d/%d/%d/%d missing %q", subcontractIndex, jurisdictionIndex, liabilityIndex, exclusivityIndex, fixed)
						}
					}
					assertExactlyOneClause(t, text, []string{"Subcontracting is prohibited.", "Buyer’s prior written consent", "Buyer may object"})
					assertExactlyOneClause(t, text, []string{"country terms added to this Agreement", "country appendix to each SOW", "separate Jurisdiction Rider"})
					assertExactlyOneClause(t, text, []string{"greater of US $50,000 or 3 times", "limited to 3 times", "No separate special cap applies"})
					assertExactlyOneClause(t, text, []string{"exclusive throughout the world", "exclusive only within the following field", "non-exclusive"})
					for _, bad := range []string{"[TBD]", "{{", "}}", "undefined", ">null<", "NaN"} {
						if strings.Contains(text, bad) {
							t.Errorf("matrix %d/%d/%d/%d contains %q", subcontractIndex, jurisdictionIndex, liabilityIndex, exclusivityIndex, bad)
						}
					}
				}
			}
		}
	}
}

func TestPublicSOWValidationSanitizesTextAndChecksConditionalFields(t *testing.T) {
	answers := validPublicSOWAnswers()
	answers.Exclusivity = dto.PublicSOWAnswersExclusivityExclusiveLimitedField
	delivery := dto.PublicSOWClientInformationDeliveryPreferenceDownload
	client := dto.PublicSOWClientInformation{
		CompanyName:        "  Acme\x00   Robotics & <Sensors>  ",
		PartyType:          dto.PublicSOWClientInformationPartyTypeSupplier,
		DeliveryPreference: &delivery,
	}

	if _, err := validateAndNormalizePublicSOW(answers, client, "download"); err == nil {
		t.Fatal("field-limited exclusivity accepted without a field scope")
	}
	scope := "  warehouse\x00   robotics & <navigation>  "
	answers.ExclusivityFieldScope = &scope
	configuration, err := validateAndNormalizePublicSOW(answers, client, "download")
	if err != nil {
		t.Fatal(err)
	}
	if configuration.CompanyName != "Acme Robotics & <Sensors>" || configuration.ExclusivityFieldScope != "warehouse robotics & <navigation>" {
		t.Fatalf("normalized configuration = %#v", configuration)
	}
	artifact, err := NewPublicSOWDocumentGenerator().Generate(configuration)
	if err != nil {
		t.Fatal(err)
	}
	document := documentXML(t, artifact.Bytes)
	if strings.ContainsRune(document, '\x00') || !strings.Contains(document, "Acme Robotics &amp; &lt;Sensors&gt;") || !strings.Contains(document, "warehouse robotics &amp; &lt;navigation&gt;") {
		t.Error("generated OOXML did not sanitize and escape public text")
	}
}

func TestPublicSOWValidatesLiabilityFigures(t *testing.T) {
	client := validPublicSOWClientInformation()
	floorAmount := float32(50000)
	multiplier := float32(3)

	missingBoth := validPublicSOWAnswers()
	missingBoth.DataIpLiabilityFloorAmount = nil
	missingBoth.DataIpLiabilityMultiplier = nil
	if _, err := validateAndNormalizePublicSOW(missingBoth, client, "download"); err == nil {
		t.Fatal("greater-of cap accepted without floor amount or multiplier")
	}

	outOfRange := validPublicSOWAnswers()
	tooLarge := float32(1_000_000_000)
	outOfRange.DataIpLiabilityFloorAmount = &tooLarge
	if _, err := validateAndNormalizePublicSOW(outOfRange, client, "download"); err == nil {
		t.Fatal("greater-of cap accepted an unreasonable floor amount")
	}

	feeOnly := validPublicSOWAnswers()
	feeOnly.DataIpLiabilityCap = dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly
	feeOnly.DataIpLiabilityMultiplier = &multiplier
	if _, err := validateAndNormalizePublicSOW(feeOnly, client, "download"); err == nil {
		t.Fatal("fee-multiple-only cap accepted a floor amount left over from the greater-of default")
	}
	feeOnly.DataIpLiabilityFloorAmount = nil
	configuration, err := validateAndNormalizePublicSOW(feeOnly, client, "download")
	if err != nil {
		t.Fatal(err)
	}
	if configuration.DataIPLiabilityMultiplier != multiplier || configuration.DataIPLiabilityFloorAmount != 0 {
		t.Fatalf("fee-multiple-only configuration = %#v", configuration)
	}

	noCap := validPublicSOWAnswers()
	noCap.DataIpLiabilityCap = dto.PublicSOWAnswersDataIpLiabilityCapNoSpecialCap
	if _, err := validateAndNormalizePublicSOW(noCap, client, "download"); err == nil {
		t.Fatal("no-special-cap accepted a floor amount and multiplier left over from the greater-of default")
	}
	noCap.DataIpLiabilityFloorAmount = nil
	noCap.DataIpLiabilityMultiplier = nil
	if _, err := validateAndNormalizePublicSOW(noCap, client, "download"); err != nil {
		t.Fatal(err)
	}

	valid := validPublicSOWAnswers()
	valid.DataIpLiabilityFloorAmount = &floorAmount
	valid.DataIpLiabilityMultiplier = &multiplier
	configuration, err = validateAndNormalizePublicSOW(valid, client, "download")
	if err != nil {
		t.Fatal(err)
	}
	if configuration.DataIPLiabilityFloorAmount != floorAmount || configuration.DataIPLiabilityMultiplier != multiplier {
		t.Fatalf("greater-of configuration = %#v", configuration)
	}
}

func TestPublicSOWServiceRequiresAcknowledgmentAndSharesArtifactWithEmail(t *testing.T) {
	mailer := &captureDeliveryMailer{available: true}
	service := NewPublicSOWConfiguratorService(NewPublicSOWDocumentGenerator(), mailer)
	answers := validPublicSOWAnswers()
	client := validPublicSOWClientInformation()
	downloadRequest := dto.PublicSOWDownloadRequest{Acknowledged: false, Answers: answers, ClientInformation: client}

	if _, err := service.Download(context.Background(), downloadRequest); err == nil {
		t.Fatal("Download() error = nil without acknowledgment")
	}
	downloadRequest.Acknowledged = dto.PublicSOWDownloadRequestAcknowledgedTrue
	download, err := service.Download(context.Background(), downloadRequest)
	if err != nil {
		t.Fatal(err)
	}
	emailRequest := dto.PublicSOWEmailRequest{
		Acknowledged:      dto.PublicSOWEmailRequestAcknowledgedTrue,
		Answers:           answers,
		ClientInformation: client,
		Email:             openapi_types.Email("reader@example.com"),
	}
	emailed, err := service.Email(context.Background(), emailRequest, "public-email-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(download.Bytes, emailed.Bytes) {
		t.Error("download and email generated different document bytes")
	}
	if mailer.count() != 1 || len(mailer.messages[0].Attachments) != 1 || !bytes.Equal(mailer.messages[0].Attachments[0].Content, download.Bytes) {
		t.Fatal("mailer did not receive the generated Word artifact")
	}
}

func assertExactlyOneClause(t *testing.T, text string, clauses []string) {
	t.Helper()
	count := 0
	for _, clause := range clauses {
		if strings.Contains(text, clause) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("document contains %d selected clauses from %q", count, clauses)
	}
}

var _ mail.Mailer = (*captureDeliveryMailer)(nil)
