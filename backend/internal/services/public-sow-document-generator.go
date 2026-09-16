package services

import (
	"strings"

	"github.com/pixels-two/sow/backend/internal/models/dto"
)

// PublicSOWDocumentGenerator creates a public configured Word artifact.
type PublicSOWDocumentGenerator interface {
	Generate(configuration PublicSOWConfiguration) (DocumentArtifact, error)
}

// OOXMLPublicSOWDocumentGenerator renders the public choices with the canonical Word styles.
type OOXMLPublicSOWDocumentGenerator struct {
	base *OOXMLDocumentGenerator
}

// NewPublicSOWDocumentGenerator builds the public Word generator.
func NewPublicSOWDocumentGenerator() *OOXMLPublicSOWDocumentGenerator {
	return &OOXMLPublicSOWDocumentGenerator{base: NewDocumentGenerator()}
}

// Generate creates a deterministic public agreement package.
func (g *OOXMLPublicSOWDocumentGenerator) Generate(configuration PublicSOWConfiguration) (DocumentArtifact, error) {
	model := blankDocumentModel()
	model.Terms.Exclusive = false
	model.Public = &publicSOWDocumentChoices{
		Subcontracting:                 string(configuration.Subcontracting),
		JurisdictionRules:              string(configuration.JurisdictionRules),
		DataIPLiabilityCap:             string(configuration.DataIPLiabilityCap),
		Exclusivity:                    string(configuration.Exclusivity),
		ExclusivityFieldScope:          configuration.ExclusivityFieldScope,
		ClientReference:                publicClientReference(configuration),
		DataIPLiabilityFloorAmountText: formatMoney(configuration.DataIPLiabilityFloorAmount, "USD", true),
		DataIPLiabilityMultiplierText:  formatNumber(float64(configuration.DataIPLiabilityMultiplier)),
	}
	artifact, err := g.base.generateModel(model)
	if err != nil {
		return DocumentArtifact{}, err
	}
	artifact.Filename = "Robot_Sensor_Data_Content_Development_Agreement.docx"
	return artifact, nil
}

func publicClientReference(configuration PublicSOWConfiguration) string {
	parts := make([]string, 0, 2)
	if configuration.CompanyName != "" {
		parts = append(parts, "company “"+configuration.CompanyName+"”")
	}
	if configuration.PartyType != "" {
		parts = append(parts, "role preference “"+publicPartyTypeLabel(configuration.PartyType)+"”")
	}
	if len(parts) == 0 {
		return ""
	}
	return "Client reference only: " + strings.Join(parts, "; ") + ". This information does not assign the company to a contract party."
}

func publicPartyTypeLabel(value string) string {
	switch value {
	case string(dto.PublicSOWClientInformationPartyTypeSupplier):
		return "Supplier"
	case string(dto.PublicSOWClientInformationPartyTypeBuyer):
		return "Buyer"
	default:
		return "Other"
	}
}
