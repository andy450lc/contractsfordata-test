package services

import (
	"strings"
	"unicode/utf8"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

const (
	maxPublicCompanyNameLength = 120
	maxPublicFieldScopeLength  = 160

	minPublicLiabilityFloorAmount = 0.01
	maxPublicLiabilityFloorAmount = 100_000_000
	minPublicLiabilityMultiplier  = 0.01
	maxPublicLiabilityMultiplier  = 1_000
)

// PublicSOWConfiguration is validated input for one public document request.
type PublicSOWConfiguration struct {
	Subcontracting             dto.PublicSOWAnswersSubcontracting
	JurisdictionRules          dto.PublicSOWAnswersJurisdictionRules
	DataIPLiabilityCap         dto.PublicSOWAnswersDataIpLiabilityCap
	DataIPLiabilityFloorAmount float32
	DataIPLiabilityMultiplier  float32
	Exclusivity                dto.PublicSOWAnswersExclusivity
	ExclusivityFieldScope      string
	CompanyName                string
	PartyType                  string
}

func validateAndNormalizePublicSOW(answers dto.PublicSOWAnswers, client dto.PublicSOWClientInformation, deliveryMethod string) (PublicSOWConfiguration, error) {
	v := &validationCollector{}
	validatePublicChoice(v, "answers.subcontracting", string(answers.Subcontracting), []string{
		string(dto.PublicSOWAnswersSubcontractingNeverAllowed),
		string(dto.PublicSOWAnswersSubcontractingBuyerWrittenConsent),
		string(dto.PublicSOWAnswersSubcontractingNoticeWithBuyerObjection),
	})
	validatePublicChoice(v, "answers.jurisdiction_rules", string(answers.JurisdictionRules), []string{
		string(dto.PublicSOWAnswersJurisdictionRulesAgreementCountryTerms),
		string(dto.PublicSOWAnswersJurisdictionRulesSowCountryAppendix),
		string(dto.PublicSOWAnswersJurisdictionRulesSeparateCountryRider),
	})
	validatePublicChoice(v, "answers.data_ip_liability_cap", string(answers.DataIpLiabilityCap), []string{
		string(dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple),
		string(dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly),
		string(dto.PublicSOWAnswersDataIpLiabilityCapNoSpecialCap),
	})
	validatePublicChoice(v, "answers.exclusivity", string(answers.Exclusivity), []string{
		string(dto.PublicSOWAnswersExclusivityFullyExclusiveGlobal),
		string(dto.PublicSOWAnswersExclusivityExclusiveLimitedField),
		string(dto.PublicSOWAnswersExclusivityNonExclusive),
	})

	scope := normalizePublicText(valueOrEmpty(answers.ExclusivityFieldScope))
	if answers.Exclusivity == dto.PublicSOWAnswersExclusivityExclusiveLimitedField && scope == "" {
		v.add("answers.exclusivity_field_scope", "is required for field-limited exclusivity")
	}
	if answers.Exclusivity != dto.PublicSOWAnswersExclusivityExclusiveLimitedField && scope != "" {
		v.add("answers.exclusivity_field_scope", "must be omitted unless exclusivity is field-limited")
	}
	validatePublicTextLength(v, "answers.exclusivity_field_scope", scope, maxPublicFieldScopeLength)

	floorAmount, multiplier := validatePublicLiabilityFigures(v, answers)

	configuration := PublicSOWConfiguration{
		Subcontracting: answers.Subcontracting, JurisdictionRules: answers.JurisdictionRules,
		DataIPLiabilityCap: answers.DataIpLiabilityCap, Exclusivity: answers.Exclusivity,
		ExclusivityFieldScope:      scope,
		DataIPLiabilityFloorAmount: floorAmount,
		DataIPLiabilityMultiplier:  multiplier,
	}
	configuration.CompanyName = normalizePublicText(client.CompanyName)
	if configuration.CompanyName == "" {
		v.add("client_information.company_name", "is required")
	}
	validatePublicTextLength(v, "client_information.company_name", configuration.CompanyName, maxPublicCompanyNameLength)

	configuration.PartyType = string(client.PartyType)
	validatePublicChoice(v, "client_information.party_type", configuration.PartyType, []string{"supplier", "buyer", "other"})

	if client.DeliveryPreference != nil {
		preference := string(*client.DeliveryPreference)
		validatePublicChoice(v, "client_information.delivery_preference", preference, []string{"download", "email"})
		if preference != deliveryMethod {
			v.add("client_information.delivery_preference", "must match the selected delivery method")
		}
	}
	if err := v.result(); err != nil {
		return PublicSOWConfiguration{}, err
	}
	return configuration, nil
}

// validatePublicLiabilityFigures checks the dollar floor and fee
// multiplier the reader typed in for question 3 against which value(s)
// the selected liability cap option actually uses, and returns the
// normalized values to store on the configuration.
func validatePublicLiabilityFigures(v *validationCollector, answers dto.PublicSOWAnswers) (floorAmount, multiplier float32) {
	needsFloor := answers.DataIpLiabilityCap == dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple
	needsMultiplier := answers.DataIpLiabilityCap == dto.PublicSOWAnswersDataIpLiabilityCapGreaterOfAmountOrFeeMultiple ||
		answers.DataIpLiabilityCap == dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly

	floorAmount = validatePublicLiabilityFigure(v, "answers.data_ip_liability_floor_amount", "dollar amount",
		needsFloor, answers.DataIpLiabilityFloorAmount, minPublicLiabilityFloorAmount, maxPublicLiabilityFloorAmount)
	multiplier = validatePublicLiabilityFigure(v, "answers.data_ip_liability_multiplier", "fee multiplier",
		needsMultiplier, answers.DataIpLiabilityMultiplier, minPublicLiabilityMultiplier, maxPublicLiabilityMultiplier)
	return floorAmount, multiplier
}

// validatePublicLiabilityFigure checks one optional numeric figure
// against whether the selected liability cap option needs it.
func validatePublicLiabilityFigure(v *validationCollector, field, label string, needed bool, value *float32, minimum, maximum float32) float32 {
	if !needed {
		if value != nil {
			v.add(field, "must be omitted unless the cap includes a "+label)
		}
		return 0
	}
	if value == nil {
		v.add(field, "is required for this liability cap option")
		return 0
	}
	if *value < minimum || *value > maximum {
		v.add(field, "must be a reasonable "+label)
	}
	return *value
}

func validatePublicChoice(v *validationCollector, field, value string, allowed []string) {
	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}
	v.add(field, "must be one of the supported choices")
}

func validatePublicTextLength(v *validationCollector, field, value string, maximum int) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximum {
		v.add(field, "must contain at most "+itoa(maximum)+" valid characters")
	}
}

func normalizePublicText(value string) string {
	return strings.Join(strings.Fields(cleanText(value)), " ")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ error = (*common.ValidationError)(nil)
