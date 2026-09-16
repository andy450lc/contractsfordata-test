package services

import (
	"math"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"golang.org/x/text/language"
)

const maxContractNumber = 10_000_000_000_000

var (
	uppercaseTwo        = regexp.MustCompile(`^[A-Z]{2}$`)
	uppercaseThree      = regexp.MustCompile(`^[A-Z]{3}$`)
	decimalText         = regexp.MustCompile(`^\d+(\.\d+)?$`)
	idempotencyKey      = regexp.MustCompile(`^[\x21-\x7e]{8,256}$`)
	supportedCurrencies = map[string]struct{}{
		"USD": {}, "EUR": {}, "GBP": {}, "INR": {}, "JPY": {}, "CNY": {}, "CAD": {}, "AUD": {},
		"SGD": {}, "AED": {}, "CHF": {}, "HKD": {}, "KRW": {}, "BRL": {}, "MXN": {}, "SEK": {},
		"NOK": {}, "DKK": {}, "NZD": {}, "ZAR": {}, "PLN": {}, "IDR": {}, "PHP": {}, "THB": {},
		"VND": {}, "TRY": {}, "SAR": {}, "ILS": {}, "NGN": {}, "KES": {},
	}
)

type validationCollector struct{ details []common.ValidationDetail }

func (v *validationCollector) add(field, message string) {
	v.details = append(v.details, common.ValidationDetail{Field: field, Message: message})
}
func (v *validationCollector) required(field, value string) {
	if strings.TrimSpace(cleanText(value)) == "" {
		v.add(field, "is required")
	} else {
		v.text(field, value)
	}
}
func (v *validationCollector) optional(field, value string) {
	if value != "" {
		v.text(field, value)
	}
}
func (v *validationCollector) text(field, value string) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(cleanText(value)) > 10000 {
		v.add(field, "must contain at most 10000 valid characters")
	}
}
func (v *validationCollector) rows(field string, rows []dto.ContractTextRow, required bool) {
	if required && len(rows) == 0 {
		v.add(field, "must contain at least one row")
	}
	if len(rows) > 250 {
		v.add(field, "must contain at most 250 rows")
	}
	for i, row := range rows {
		v.required(field+"["+itoa(i)+"].text", row.Text)
	}
}
func (v *validationCollector) result() error {
	if len(v.details) == 0 {
		return nil
	}
	return &common.ValidationError{Details: v.details}
}

func validateConfiguration(config *dto.ContractConfiguration) error {
	if config == nil {
		return nil
	}
	v := &validationCollector{}
	if config.TemplateVersion != dto.ContractConfigurationTemplateVersionContentDevelopment1 {
		v.add("configuration.template_version", "is unsupported")
	}
	validateOrganization(v, config.Organization)
	validateAgreement(v, config.Steps.Agreement)
	validateScope(v, config.Steps.Scope)
	validatePricing(v, config.Steps.Pricing, config.Steps.Agreement)
	validateEnvironment(v, config.Steps.Environment)
	validateTechnical(v, config.Steps.Technical)
	validateDelivery(v, config.Steps.Delivery)
	validateCounterparty(v, config.Steps.Counterparty)
	return v.result()
}

func validateOrganization(v *validationCollector, value dto.ContractOrganization) {
	for field, text := range map[string]string{"legal_name": value.LegalName, "entity_type": value.EntityType, "incorporation_place": value.IncorporationPlace, "governing_law": value.GoverningLaw, "address": value.Address, "short_name": value.ShortName, "signer_name": value.SignerName, "signer_title": value.SignerTitle} {
		v.required("configuration.organization."+field, text)
	}
	v.optional("configuration.organization.contact_phone", value.ContactPhone)
	validateEmailField(v, "configuration.organization.signer_email", string(value.SignerEmail))
}

func validateAgreement(v *validationCollector, value dto.AgreementTerms) {
	prefix := "configuration.steps.agreement."
	if value.SowNumber < 1 || value.SowNumber > 999999 {
		v.add(prefix+"sow_number", "must be between 1 and 999999")
	}
	v.required(prefix+"title", value.Title)
	v.required(prefix+"materials_summary", value.MaterialsSummary)
	if value.EffectiveDate.IsZero() {
		v.add(prefix+"effective_date", "must be a valid date")
	}
	whole := map[string]int{"incident_notice_hours": value.IncidentNoticeHours, "cure_days": value.CureDays, "convenience_notice_days": value.ConvenienceNoticeDays, "records_retention_years": value.RecordsRetentionYears, "liability_lookback_months": value.LiabilityLookbackMonths}
	for field, number := range whole {
		validateWhole(v, prefix+field, number)
	}
	validateNumber(v, prefix+"excluded_claims_cap_multiplier", float64(value.ExcludedClaimsCapMultiplier), true)
	validateNumber(v, prefix+"data_claims_cap_floor", float64(value.DataClaimsCapFloor), false)
}

func validateScope(v *validationCollector, value dto.ContractScope) {
	prefix := "configuration.steps.scope."
	v.required(prefix+"deliverable_description", value.DeliverableDescription)
	v.required(prefix+"venue_constraint", value.VenueConstraint)
	v.required(prefix+"unit", value.Unit)
	v.optional(prefix+"notes", value.Notes)
	validateNumber(v, prefix+"target_volume", float64(value.TargetVolume), true)
	if len(value.Regions) == 0 || len(value.Regions) > 250 {
		v.add(prefix+"regions", "must contain 1 to 250 country codes")
	}
	seen := map[string]struct{}{}
	for i, code := range value.Regions {
		region, regionErr := language.ParseRegion(code)
		if !uppercaseTwo.MatchString(code) || regionErr != nil || !region.IsCountry() {
			v.add(prefix+"regions["+itoa(i)+"]", "must be a supported country code")
		}
		if _, ok := seen[code]; ok {
			v.add(prefix+"regions", "must contain unique codes")
		}
		seen[code] = struct{}{}
	}
	v.rows(prefix+"deliverables", value.Deliverables, true)
}

func validatePricing(v *validationCollector, value dto.ContractPricing, agreement dto.AgreementTerms) {
	prefix := "configuration.steps.pricing."
	_, currencySupported := supportedCurrencies[value.Currency]
	if !uppercaseThree.MatchString(value.Currency) || !currencySupported {
		v.add(prefix+"currency", "must be a supported currency code")
	}
	validateNumber(v, prefix+"unit_fee", float64(value.UnitFee), false)
	validateWhole(v, prefix+"payment_terms_days", value.PaymentTermsDays)
	v.required(prefix+"invoicing_cadence", value.InvoicingCadence)
	v.required(prefix+"payment_methods", value.PaymentMethods)
	validateEmailField(v, prefix+"invoice_email", string(value.InvoiceEmail))
	if value.DepositRequired && (value.DepositAmount == nil || *value.DepositAmount <= 0) {
		v.add(prefix+"deposit_amount", "must be greater than zero when deposit_required is true")
	}
	if !value.DepositRequired && value.DepositAmount != nil {
		v.add(prefix+"deposit_amount", "must be null when deposit_required is false")
	}
	if value.DepositAmount != nil {
		validateNumber(v, prefix+"deposit_amount", float64(*value.DepositAmount), false)
	}
	validateMilestones(v, prefix, value.Milestones, agreement)
}

func validateMilestones(v *validationCollector, prefix string, rows []dto.ContractMilestone, agreement dto.AgreementTerms) {
	if len(rows) == 0 || len(rows) > 100 {
		v.add(prefix+"milestones", "must contain 1 to 100 rows")
		return
	}
	finalCount := 0
	var finalDeadline time.Time
	for _, row := range rows {
		if row.IsFinal {
			finalCount++
			finalDeadline = row.Deadline.Time
		}
	}
	for i, row := range rows {
		field := prefix + "milestones[" + itoa(i) + "]"
		v.required(field+".name", row.Name)
		v.required(field+".volume", row.Volume)
		v.optional(field+".notes", row.Notes)
		validateMilestoneDeadline(v, field, row, agreement, finalDeadline)
	}
	if finalCount != 1 {
		v.add(prefix+"milestones", "must contain exactly one final milestone")
	}
}

func validateMilestoneDeadline(v *validationCollector, field string, row dto.ContractMilestone, agreement dto.AgreementTerms, finalDeadline time.Time) {
	if row.Deadline.IsZero() {
		v.add(field+".deadline", "must be a valid date")
	}
	if !agreement.EffectiveDate.IsZero() && row.Deadline.Before(agreement.EffectiveDate.Time) {
		v.add(field+".deadline", "must be on or after the effective date")
	}
	if !row.IsFinal && !finalDeadline.IsZero() && row.Deadline.After(finalDeadline) {
		v.add(field+".deadline", "must be on or before the final delivery")
	}
}

func validateEnvironment(v *validationCollector, value dto.ContractEnvironment) {
	prefix := "configuration.steps.environment."
	validateEnvironmentVerticals(v, prefix, value.Verticals)
	validateEnvironmentLimits(v, prefix, value.Limits)
	validateEnvironmentDifficulty(v, prefix, value.Difficulty)
	validateEnvironmentDifficultyCaps(v, prefix, value.DifficultyCaps)
	validateWhole(v, prefix+"change_window_hours", value.ChangeWindowHours)
	v.rows(prefix+"extra_prohibited", value.ExtraProhibited, false)
	v.rows(prefix+"extra_excluded_settings", value.ExtraExcludedSettings, false)
}

func validateEnvironmentVerticals(v *validationCollector, prefix string, rows []dto.ContractVertical) {
	if len(rows) == 0 || len(rows) > 100 {
		v.add(prefix+"verticals", "must contain 1 to 100 rows")
	}
	for i, row := range rows {
		field := prefix + "verticals[" + itoa(i) + "]"
		v.required(field+".verticals", row.Verticals)
		v.required(field+".target", row.Target)
		v.required(field+".details", row.Details)
	}
}

func validateEnvironmentLimits(v *validationCollector, prefix string, rows []dto.ContractLimit) {
	if len(rows) == 0 || len(rows) > 100 {
		v.add(prefix+"limits", "must contain 1 to 100 rows")
	}
	for i, row := range rows {
		field := prefix + "limits[" + itoa(i) + "]"
		v.required(field+".limit", row.Limit)
		v.optional(field+".unit", row.Unit)
		if len(row.Value) > 64 || !decimalText.MatchString(row.Value) {
			v.add(field+".value", "must be a non-negative decimal")
		}
	}
}

func validateEnvironmentDifficulty(v *validationCollector, prefix string, difficulty dto.ContractDifficulty) {
	total := float64(difficulty.Easy + difficulty.Medium + difficulty.Hard)
	if math.Abs(total-100) > 0.001 {
		v.add(prefix+"difficulty", "must total 100")
	}
	for _, item := range []struct {
		name  string
		value float32
	}{{"easy", difficulty.Easy}, {"medium", difficulty.Medium}, {"hard", difficulty.Hard}} {
		if item.value < 0 || item.value > 100 || math.IsNaN(float64(item.value)) {
			v.add(prefix+"difficulty."+item.name, "must be between 0 and 100")
		}
	}
}

func validateEnvironmentDifficultyCaps(v *validationCollector, prefix string, rows []dto.ContractDifficultyCap) {
	if len(rows) > 100 {
		v.add(prefix+"difficulty_caps", "must contain at most 100 rows")
	}
	for i, row := range rows {
		field := prefix + "difficulty_caps[" + itoa(i) + "]"
		v.required(field+".scope", row.Scope)
		validateDecimal(v, field+".easy", row.Easy)
		validateDecimal(v, field+".medium", row.Medium)
		validateDecimal(v, field+".hard", row.Hard)
	}
}

func validateTechnical(v *validationCollector, value dto.ContractTechnical) {
	prefix := "configuration.steps.technical."
	if len(value.Requirements) == 0 || len(value.Requirements) > 100 {
		v.add(prefix+"requirements", "must contain 1 to 100 rows")
	}
	for i, row := range value.Requirements {
		field := prefix + "requirements[" + itoa(i) + "]"
		v.required(field+".requirement", row.Requirement)
		v.required(field+".specification", row.Specification)
		v.required(field+".rejection", row.Rejection)
	}
	v.rows(prefix+"pre_collection_materials", value.PreCollectionMaterials, false)
}
func validateDelivery(v *validationCollector, value dto.ContractDelivery) {
	prefix := "configuration.steps.delivery."
	v.required(prefix+"storage_location", value.StorageLocation)
	v.required(prefix+"delivery_cadence", value.DeliveryCadence)
	v.required(prefix+"manifest_format", value.ManifestFormat)
	v.required(prefix+"integrity_text", value.IntegrityText)
	v.rows(prefix+"manifest_fields", value.ManifestFields, true)
	v.rows(prefix+"site_data_fields", value.SiteDataFields, false)
	validateWhole(v, prefix+"review_window_days", value.ReviewWindowDays)
	validateWhole(v, prefix+"correction_days", value.CorrectionDays)
	validateWhole(v, prefix+"deletion_window_days", value.DeletionWindowDays)
}
func validateCounterparty(v *validationCollector, value dto.ContractCounterparty) {
	prefix := "configuration.steps.counterparty."
	if !value.Mode.Valid() {
		v.add(prefix+"mode", "is unsupported")
	}
	validateEmailField(v, prefix+"email", string(value.Email))
	validateCCEmails(v, prefix+"cc_emails", value.CcEmails)
	if value.Mode == dto.ContractCounterpartyModeDetails {
		v.required(prefix+"legal_name", value.LegalName)
		v.required(prefix+"entity_jurisdiction", value.EntityJurisdiction)
		v.required(prefix+"address", value.Address)
		v.required(prefix+"short_name", value.ShortName)
		v.required(prefix+"signatory_name", value.SignatoryName)
		v.optional(prefix+"signatory_title", value.SignatoryTitle)
	} else {
		for field, text := range map[string]string{"legal_name": value.LegalName, "entity_jurisdiction": value.EntityJurisdiction, "address": value.Address, "short_name": value.ShortName, "signatory_name": value.SignatoryName, "signatory_title": value.SignatoryTitle} {
			v.optional(prefix+field, text)
		}
	}
}

func normalizeEmail(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized || strings.ContainsAny(normalized, " \t\r\n") || len(normalized) > 254 {
		return "", &common.ValidationError{Details: []common.ValidationDetail{{Field: "email", Message: "must be a valid email address"}}}
	}
	return normalized, nil
}
func validateEmailField(v *validationCollector, field, value string) {
	if value == "" {
		v.add(field, "is required")
		return
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 || strings.ContainsAny(value, " \t\r\n") {
		v.add(field, "must be a valid email address")
	}
}
func validateCCEmails(v *validationCollector, field, value string) {
	v.optional(field, value)
	if strings.TrimSpace(value) == "" {
		return
	}
	for _, entry := range strings.Split(value, ",") {
		email := strings.TrimSpace(entry)
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || len(email) > 254 {
			v.add(field, "contains an invalid email address")
			return
		}
	}
}
func validateWhole(v *validationCollector, field string, value int) {
	if value < 0 || value > maxContractNumber {
		v.add(field, "must be between 0 and 10000000000000")
	}
}
func validateNumber(v *validationCollector, field string, value float64, positive bool) {
	invalid := math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= maxContractNumber
	if positive {
		invalid = invalid || value <= 0
	}
	if invalid {
		v.add(field, "is outside the allowed range")
	}
}
func validateDecimal(v *validationCollector, field, value string) {
	if len(value) > 64 || !decimalText.MatchString(value) {
		v.add(field, "must be a non-negative decimal")
	}
}
func validateIdempotencyKey(value string) error {
	if !idempotencyKey.MatchString(value) {
		return &common.ValidationError{Details: []common.ValidationDetail{{Field: "Idempotency-Key", Message: "must be 8 to 256 visible ASCII characters"}}}
	}
	return nil
}
func itoa(value int) string { return strconv.Itoa(value) }
