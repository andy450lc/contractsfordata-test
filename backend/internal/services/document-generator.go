package services

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pixels-two/sow/backend/internal/models/dto"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// DocumentMIMEType is the media type for an OOXML Word document.
const DocumentMIMEType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

var fixedZipTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
var vowelSoundPattern = regexp.MustCompile(`^(a|e|i|o|u|hour|honest|honor|heir)`)

// DocumentArtifact is a complete request-scoped Word document.
type DocumentArtifact struct {
	Bytes    []byte
	Filename string
	MIMEType string
}

type documentModel struct {
	SOWNumber   string
	Customer    partyModel
	Developer   partyModel
	Terms       termsDocumentModel
	Scope       scopeDocumentModel
	Pricing     pricingDocumentModel
	Environment environmentDocumentModel
	Technical   technicalDocumentModel
	Delivery    deliveryDocumentModel
	Public      *publicSOWDocumentChoices
}

type publicSOWDocumentChoices struct {
	Subcontracting, JurisdictionRules, DataIPLiabilityCap, Exclusivity string
	ExclusivityFieldScope, ClientReference                             string
	DataIPLiabilityFloorAmountText, DataIPLiabilityMultiplierText      string
}

type partyModel struct {
	LegalName, EntityType, Place, Law, Address, ShortName, SignerName, SignerTitle string
}

type termsDocumentModel struct {
	EffectiveDate, Title, MaterialsSummary                              string
	IncidentHours, CureDays, NoticeDays, RetentionYears, LookbackMonths string
	ClaimsMultiplier, ClaimsFloor                                       string
	Exclusive                                                           bool
}

type scopeDocumentModel struct {
	Description, Regions, Venue, Volume, Unit, UnitSingular, ShortUnit, ShortSingular, Notes string
	Deliverables                                                                             []string
	AmbientAudio                                                                             bool
}

type milestoneDocumentModel struct{ Name, Volume, Deadline string }

type pricingDocumentModel struct {
	Currency, CurrencyName, UnitFee, MaximumTotal, DepositAmount, Cadence, PaymentDays, PaymentMethods, InvoiceEmail string
	Milestones                                                                                                       []milestoneDocumentModel
	FirstDeadline, FinalDeadline                                                                                     string
	DepositRequired, FirmDeadline                                                                                    bool
}

type environmentDocumentModel struct {
	Verticals                      [][]string
	Limits                         []dto.ContractLimit
	Difficulty                     [3]string
	Caps                           []dto.ContractDifficultyCap
	PlanRequired                   bool
	ChangeHours                    string
	ExtraProhibited, ExtraExcluded []string
}

type technicalDocumentModel struct {
	Requirements       [][]string
	Materials          []string
	ConditionOfPayment bool
}

type deliveryDocumentModel struct {
	Cadence, Storage, ManifestFormat, Integrity, ReviewDays, CorrectionDays, DeletionDays string
	ManifestFields, SiteFields                                                            []string
	DeemedAcceptance, RejectedStaysDeveloperOwned                                         bool
}

type documentElement struct {
	Style     string
	Text      string
	Rows      [][]string
	PageBreak bool
}

// DocumentGenerator creates one Word artifact from a request configuration.
type DocumentGenerator interface {
	Generate(configuration *dto.ContractConfiguration) (DocumentArtifact, error)
}

// OOXMLDocumentGenerator creates deterministic OOXML packages in memory.
type OOXMLDocumentGenerator struct{}

// NewDocumentGenerator builds the canonical Word generator.
func NewDocumentGenerator() *OOXMLDocumentGenerator { return &OOXMLDocumentGenerator{} }

// Generate validates presentation conversion and creates one Word artifact.
func (g *OOXMLDocumentGenerator) Generate(configuration *dto.ContractConfiguration) (DocumentArtifact, error) {
	if configuration == nil {
		return g.generateModel(blankDocumentModel())
	}
	return g.generateModel(configuredDocumentModel(*configuration))
}

func (g *OOXMLDocumentGenerator) generateModel(model documentModel) (DocumentArtifact, error) {
	documentXML := renderDocumentXML(canonicalDocument(model))
	entries := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypesXML},
		{"_rels/.rels", packageRelationshipsXML},
		{"docProps/app.xml", appPropertiesXML},
		{"docProps/core.xml", corePropertiesXML},
		{"word/document.xml", documentXML},
		{"word/_rels/document.xml.rels", documentRelationshipsXML},
		{"word/header1.xml", watermarkHeaderXML},
		{"word/footer1.xml", footerXML(model.SOWNumber)},
		{"word/styles.xml", stylesXML},
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate, Modified: fixedZipTime}
		header.SetMode(0o600)
		part, err := writer.CreateHeader(header)
		if err != nil {
			return DocumentArtifact{}, fmt.Errorf("creating OOXML part %s: %w", entry.name, err)
		}
		if _, err := part.Write([]byte(entry.body)); err != nil {
			return DocumentArtifact{}, fmt.Errorf("writing OOXML part %s: %w", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return DocumentArtifact{}, fmt.Errorf("closing OOXML package: %w", err)
	}

	return DocumentArtifact{Bytes: output.Bytes(), Filename: documentFilename(model), MIMEType: DocumentMIMEType}, nil
}

func configuredDocumentModel(config dto.ContractConfiguration) documentModel {
	organization := config.Organization
	agreement := config.Steps.Agreement
	scope := config.Steps.Scope
	pricing := config.Steps.Pricing
	environment := config.Steps.Environment
	technical := config.Steps.Technical
	delivery := config.Steps.Delivery
	counterparty := config.Steps.Counterparty

	developer := partyModel{
		LegalName: cleanText(counterparty.LegalName), EntityType: cleanText(counterparty.EntityJurisdiction),
		Address: cleanText(counterparty.Address), SignerName: cleanText(counterparty.SignatoryName), SignerTitle: cleanText(counterparty.SignatoryTitle),
	}
	if counterparty.Mode == dto.ContractCounterpartyModeInvite {
		developer = partyModel{LegalName: "[Developer legal name]", EntityType: "[Developer entity type and jurisdiction]", Address: "[Developer address]", SignerName: "[Developer signatory name]", SignerTitle: "[Developer signatory title]"}
	}

	unit := cleanText(scope.Unit)
	unitSingular := singularize(unit)
	milestones := make([]milestoneDocumentModel, 0, len(pricing.Milestones))
	firstDate := pricing.Milestones[0].Deadline.Time
	finalDate := pricing.Milestones[len(pricing.Milestones)-1].Deadline.Time
	for _, row := range pricing.Milestones {
		deadline := formatLongDate(row.Deadline.Time)
		if row.Notes != "" {
			deadline = cleanText(row.Notes) + ", no later than " + deadline
		}
		milestones = append(milestones, milestoneDocumentModel{cleanText(row.Name), cleanText(row.Volume), deadline})
		if row.Deadline.Before(firstDate) {
			firstDate = row.Deadline.Time
		}
		if row.IsFinal {
			finalDate = row.Deadline.Time
		}
	}

	deposit := ""
	if pricing.DepositAmount != nil {
		deposit = formatMoney(*pricing.DepositAmount, pricing.Currency, false)
	}
	return documentModel{
		SOWNumber:   strconv.Itoa(agreement.SowNumber),
		Customer:    partyModel{LegalName: cleanText(organization.LegalName), EntityType: cleanText(organization.EntityType), Place: cleanText(organization.IncorporationPlace), Law: cleanText(organization.GoverningLaw), Address: cleanText(organization.Address), ShortName: cleanText(organization.ShortName), SignerName: cleanText(organization.SignerName), SignerTitle: cleanText(organization.SignerTitle)},
		Developer:   developer,
		Terms:       termsDocumentModel{EffectiveDate: formatLongDate(agreement.EffectiveDate.Time), Title: cleanText(agreement.Title), MaterialsSummary: cleanText(agreement.MaterialsSummary), IncidentHours: formatNumber(float64(agreement.IncidentNoticeHours)), CureDays: formatNumber(float64(agreement.CureDays)), NoticeDays: formatNumber(float64(agreement.ConvenienceNoticeDays)), RetentionYears: formatNumber(float64(agreement.RecordsRetentionYears)), LookbackMonths: formatNumber(float64(agreement.LiabilityLookbackMonths)), ClaimsMultiplier: formatNumber(float64(agreement.ExcludedClaimsCapMultiplier)), ClaimsFloor: formatMoney(agreement.DataClaimsCapFloor, "USD", true), Exclusive: agreement.Exclusive},
		Scope:       scopeDocumentModel{Description: cleanText(scope.DeliverableDescription), Regions: countryList(scope.Regions), Venue: cleanText(scope.VenueConstraint), Volume: formatNumber(float64(scope.TargetVolume)), Unit: unit, UnitSingular: unitSingular, ShortUnit: lastWord(unit), ShortSingular: lastWord(unitSingular), Deliverables: textRows(scope.Deliverables), Notes: cleanText(scope.Notes), AmbientAudio: scope.AmbientAudio != nil && *scope.AmbientAudio},
		Pricing:     pricingDocumentModel{Currency: pricing.Currency, CurrencyName: currencyName(pricing.Currency), UnitFee: formatMoney(pricing.UnitFee, pricing.Currency, false), MaximumTotal: formatMoney(pricing.UnitFee*scope.TargetVolume, pricing.Currency, true), DepositAmount: deposit, Cadence: cleanText(pricing.InvoicingCadence), PaymentDays: strconv.Itoa(pricing.PaymentTermsDays), PaymentMethods: cleanText(pricing.PaymentMethods), InvoiceEmail: string(pricing.InvoiceEmail), Milestones: milestones, FirstDeadline: formatLongDate(firstDate), FinalDeadline: formatLongDate(finalDate), DepositRequired: pricing.DepositRequired, FirmDeadline: pricing.FirmDeadline},
		Environment: environmentDocumentModel{Verticals: verticalRows(environment.Verticals), Limits: environment.Limits, Difficulty: [3]string{formatNumber(float64(environment.Difficulty.Easy)) + "%", formatNumber(float64(environment.Difficulty.Medium)) + "%", formatNumber(float64(environment.Difficulty.Hard)) + "%"}, Caps: environment.DifficultyCaps, PlanRequired: environment.PlanRequired, ChangeHours: strconv.Itoa(environment.ChangeWindowHours), ExtraProhibited: textRows(environment.ExtraProhibited), ExtraExcluded: textRows(environment.ExtraExcludedSettings)},
		Technical:   technicalDocumentModel{Requirements: requirementRows(technical.Requirements), Materials: textRows(technical.PreCollectionMaterials), ConditionOfPayment: technical.ConditionOfPayment},
		Delivery:    deliveryDocumentModel{Cadence: lowerFirst(cleanText(delivery.DeliveryCadence)), Storage: cleanText(delivery.StorageLocation), ManifestFormat: cleanText(delivery.ManifestFormat), Integrity: cleanText(delivery.IntegrityText), ReviewDays: strconv.Itoa(delivery.ReviewWindowDays), CorrectionDays: strconv.Itoa(delivery.CorrectionDays), DeletionDays: strconv.Itoa(delivery.DeletionWindowDays), ManifestFields: textRows(delivery.ManifestFields), SiteFields: textRows(delivery.SiteDataFields), DeemedAcceptance: delivery.DeemedAcceptance, RejectedStaysDeveloperOwned: delivery.RejectedStaysDeveloperOwned},
	}
}

func blankDocumentModel() documentModel {
	return documentModel{
		SOWNumber:   "[number]",
		Customer:    partyModel{LegalName: "[Customer legal name]", EntityType: "[entity type]", Place: "[place of incorporation]", Law: "[governing law and venue]", Address: "[Customer address]", ShortName: "Customer", SignerName: "[Customer signer name]", SignerTitle: "[Customer signer title]"},
		Developer:   partyModel{LegalName: "[Developer legal name]", EntityType: "[Developer entity type and jurisdiction]", Address: "[Developer address]", SignerName: "[Developer signatory name]", SignerTitle: "[Developer signatory title]"},
		Terms:       termsDocumentModel{EffectiveDate: "[effective date]", Title: "[SOW title]", MaterialsSummary: "recordings, video, sensor data, calibration data, metadata, and other materials", IncidentHours: "24", CureDays: "30", NoticeDays: "30", RetentionYears: "2", LookbackMonths: "12", ClaimsMultiplier: "2", ClaimsFloor: "US $50,000", Exclusive: true},
		Scope:       scopeDocumentModel{Description: "[deliverable description]", Regions: "[collection region or regions]", Venue: "[venue constraints]", Volume: "[target volume]", Unit: "accepted usable units", UnitSingular: "accepted usable unit", ShortUnit: "units", ShortSingular: "unit", Deliverables: []string{"Original source files", "Delivery manifest", "Consent and collection records", "All other files specified in this SOW"}},
		Pricing:     pricingDocumentModel{Currency: "USD", CurrencyName: "U.S. dollars", UnitFee: "US $[unit fee]", MaximumTotal: "US $[maximum total]", Cadence: "Weekly in arrears", PaymentDays: "30", PaymentMethods: "[payment method]", InvoiceEmail: "[invoice email]", Milestones: []milestoneDocumentModel{{"Rolling uploads", "Completed work as available", "[first delivery date]"}, {"Final delivery", "[target volume]", "[final delivery date]"}}, FirstDeadline: "[first delivery date]", FinalDeadline: "[final delivery date]", FirmDeadline: true},
		Environment: environmentDocumentModel{Verticals: [][]string{{"[collection verticals]", "[target]", "[details]"}}, Difficulty: [3]string{"[easy %]", "[medium %]", "[hard %]"}, PlanRequired: true, ChangeHours: "24"},
		Technical:   technicalDocumentModel{Requirements: [][]string{{"[requirement]", "[specification]", "[rejection criteria]"}}},
		Delivery:    deliveryDocumentModel{Cadence: "[delivery cadence]", Storage: "[storage location]", ManifestFormat: "[manifest format]", Integrity: "[integrity requirements]", ReviewDays: "30", CorrectionDays: "5", DeletionDays: "30", ManifestFields: []string{"[required manifest fields]"}, SiteFields: []string{"[required site data]"}, DeemedAcceptance: true},
	}
}

func canonicalDocument(m documentModel) []documentElement {
	s, d, t := m.Customer.ShortName, m.Developer, m.Terms
	e := []documentElement{
		p("Title", "CONTENT DEVELOPMENT AGREEMENT"),
	}
	if m.Public != nil && m.Public.ClientReference != "" {
		e = append(e, p("", m.Public.ClientReference))
	}
	e = append(e,
		p("", fmt.Sprintf("This Content Development Agreement (the “Agreement”) is effective as of %s and is entered into between %s, a %s of %s, with an address at %s (“%s”), and %s, a %s, with an address at %s (“Developer”). %s and Developer are each a “Party” and together the “Parties.”", t.EffectiveDate, m.Customer.LegalName, m.Customer.EntityType, m.Customer.Place, m.Customer.Address, s, d.LegalName, d.EntityType, d.Address, s)),
		h("1. Services."),
		p("", "1.1. Content Development Services. Developer will source, coordinate, engage, supervise, and compensate individual contributors and other Personnel who will create the "+t.MaterialsSummary+" described in an applicable statement of work (“SOW,” and collectively, the “Deliverables”). Developer is responsible for all Personnel and subcontractors as if it performed the Services itself."),
		p("", "1.2. SOW. The Parties may enter into one or more SOWs under this Agreement. Each SOW will identify the requested content, volume, delivery schedule, specifications and acceptance criteria, pricing and payment terms, and any project-specific requirements. In the event of a conflict, the applicable SOW controls solely with respect to that SOW."),
		h("2. Contributor Management."),
		p("", "2.1. Developer Responsibilities. Developer is solely responsible for sourcing, coordinating, communicating with, managing, and paying its Personnel. Developer will ensure that Personnel are genuine, individually vetted adults and meet the eligibility criteria specified in the applicable SOW before performing Services."),
		p("", "2.2. Consents and Site Permissions. Developer will obtain and maintain all worker consents, releases, notices, site permissions, and other authorizations needed for "+s+" and its customers to own, use, modify, commercialize, and sublicense the Deliverables as contemplated by this Agreement. Developer will comply with all applicable labor, workplace-safety, recording, privacy, data-protection, export-control, and employment laws."),
		p("", subcontractingClause(m)),
		h("3. Submission and Acceptance."),
		p("", "3.1. Submissions. Developer will submit Deliverables only through the delivery path stated in the applicable SOW. Developer will use commercially reasonable quality-control measures before each submission and will promptly correct, replace, or re-perform at no additional charge any non-conforming Deliverables."),
		p("", "3.2. Acceptance Testing. "+s+" may inspect, test, validate, and review each delivery for conformity with this Agreement and the applicable SOW. Only Deliverables accepted by "+s+" are billable. "+s+" may reject non-conforming footage and require correction or replacement at no additional charge. No footage is deemed accepted merely because "+s+" has not completed review."),
		h("4. Work Product; Exclusive Rights."),
		p("", "4.1. Ownership. All Deliverables, including raw, setup, sample, test, rejected, processed, and derived material created in connection with an SOW, are specially commissioned for "+s+". "+s+" exclusively owns all right, title, and interest in that Work Product from the moment of capture. To the extent any right does not vest automatically, Developer irrevocably assigns it to "+s+" and will procure equivalent assignments, consents, releases, and waivers from all applicable Personnel. Developer and its Personnel waive, and agree not to assert, any moral rights or similar rights to the maximum extent permitted by law."),
		p("", exclusivityClause(m)),
		p("", backupLicenseClause(m)),
		h("5. Compliance with Laws and Data Protection."),
		p("", "5.1. Developer Responsibility. Developer will comply with all applicable laws and regulations in connection with the Services and Work Product, including applicable recording, privacy, data-protection, employment, workplace-safety, storage, transfer, and export requirements. Developer is responsible for obtaining and maintaining all consents, notices, permissions, approvals, and other legal mechanisms required for its collection, processing, storage, transfer, and delivery of the Work Product."),
		p("", "5.2. Lawful Collection and Delivery. Developer represents and warrants that all Work Product delivered to "+s+" was lawfully collected, processed, stored, and transferred in accordance with applicable law. On reasonable request, Developer will provide reasonable evidence of the consents, permissions, and compliance records supporting the collection and delivery of the Work Product."),
		p("", "5.3. Prohibited Material. Developer will not collect, transmit, or deliver government-classified or controlled data, restricted technical data, precise mapping or geolocation data, data from restricted or sensitive facilities, or any material whose collection, transfer, or contemplated use is prohibited or requires authorization that Developer has not obtained. Developer will ensure all site-specific business information and metadata are lawfully collectable and transferable. Any unauthorized collection, transfer, retention, sale, or disclosure of Work Product is a material, non-curable breach."),
		p("", "5.4. Records. Developer will retain relevant compliance records, including worker consents, site permissions, delivery logs, and security records, for at least "+t.RetentionYears+" years after final delivery, to the extent permitted by applicable law. On reasonable request, Developer will provide reasonable documentary evidence of compliance, which may be appropriately redacted to protect personal information and third-party confidential information."),
		h("6. Confidentiality."),
		p("", "6.1. Confidential Information. “Confidential Information” means any non-public information disclosed by one Party to the other that is designated confidential or reasonably should be understood to be confidential. Each Party will protect the other’s Confidential Information using at least reasonable care and use it only to perform this Agreement."),
		p("", "6.2. Restrictions. Developer will restrict access to "+s+" Confidential Information and Work Product to Personnel with a need to know for the Services and who are bound by written confidentiality obligations no less protective than this Agreement. On expiration or termination, Developer will return or securely delete "+s+" Confidential Information and Work Product, except to the limited extent retention is required by applicable law."),
		p("", "6.3. Publicity. Developer may not issue a press release, use "+s+"’s name, trademarks, or logos, identify "+s+" as a customer, or otherwise publicize this Agreement or the relationship without "+s+"’s prior written approval."),
		h("7. Security."),
		p("", "7.1. Safeguards. Developer will maintain appropriate administrative, technical, and physical safeguards for "+s+" Confidential Information and Work Product, including encryption in transit and at rest, least-privilege access controls, malware scanning, integrity checks, and private access settings. Developer will notify "+s+" within "+t.IncidentHours+" hours of any actual or suspected unauthorized access, loss, disclosure, transfer, or security incident."),
		h("8. Term and Termination."),
		p("", "8.1. Term. This Agreement begins on the Effective Date and continues until terminated under this Section. Either Party may terminate for an uncured material breach after "+t.CureDays+" days’ written notice, except where this Agreement provides for immediate termination. "+s+" may terminate this Agreement or an SOW for convenience on "+t.NoticeDays+" days’ written notice."),
		p("", "8.2. Effect of Termination. "+s+" is liable only for accepted Services delivered before termination. Developer will provide reasonable transition assistance and will promptly return or delete "+s+" Confidential Information as required under Section 6.2."),
		p("", "8.3. Survival. Sections 4 (Work Product; Exclusive Rights), 5 (Compliance with Laws and Data Protection), 6 (Confidentiality), 7 (Security), 8.2 (Effect of Termination), 8.3 (Survival), 9 (Representations and Warranties), 10 (Indemnification), 11 (Limitation of Liability), and 12 (General) survive expiration or termination."),
		h("9. Representations and Warranties."),
		p("", "9.1. Developer Warranties. Developer represents and warrants that it will perform the Services professionally and in conformity with this Agreement and each SOW; it has all rights, permissions, consents, and authority required to provide the Work Product and authorize "+s+"’s contemplated use; and the Work Product will not infringe third-party rights or violate applicable law. Developer will take commercially reasonable measures to avoid nonconsenting bystanders, minors, sensitive personal information, confidential or proprietary material, and dangerous or restricted locations."),
		h("10. Indemnification."),
		p("", "10.1. Mutual Indemnity. Each Party will indemnify, defend, and hold the other harmless from third-party claims arising from its breach of this Agreement or its gross negligence, willful misconduct, or fraud. Developer will additionally indemnify "+s+" from third-party claims, regulatory actions, fines, penalties, and losses that the Work Product, its collection, export, transfer, or "+s+"’s use as contemplated by this Agreement infringes intellectual-property rights or violates privacy, publicity, data-protection, consent, data-export, or security laws, except to the extent directly caused by "+s+"’s modification or use outside this Agreement."),
		h("11. Limitation of Liability."),
		p("", liabilityClause(m)),
		h("12. General."),
		p("", "12.1. Independent Contractors. The Parties are independent contractors. This Agreement does not establish a partnership, joint venture, employment, or agency relationship."),
		p("", "12.2. Amendments; No Waiver. This Agreement may be amended only in a writing signed by both Parties. A waiver is effective only if in writing and signed by the waiving Party."),
		p("", "12.3. Severability. If any provision is unenforceable, the remainder of this Agreement will remain valid and enforceable to the fullest extent permitted by law."),
		p("", jurisdictionClause(m)),
		p("", "12.5. Entire Agreement. This Agreement and its SOWs are the entire agreement and supersede all prior or contemporaneous representations, understandings, proposals, and agreements concerning its subject matter."),
	)
	e = append(e, signatureElements(m)...)
	e = append(e, statementOfWorkElements(m)...)
	return e
}

func subcontractingClause(m documentModel) string {
	s := m.Customer.ShortName
	if m.Public == nil {
		return "2.3. Subcontractors. Developer may not use a subcontractor, affiliate, or other third party in connection with the Services without " + s + "’s prior written consent. Developer will ensure that each approved party is bound in writing by obligations no less protective of " + s + " than this Agreement and remains responsible for its acts and omissions."
	}
	switch m.Public.Subcontracting {
	case string(dto.PublicSOWAnswersSubcontractingNeverAllowed):
		return "2.3. Subcontractors. Subcontracting is prohibited. Developer will perform the Services using only its direct Personnel and remains responsible for their acts and omissions."
	case string(dto.PublicSOWAnswersSubcontractingNoticeWithBuyerObjection):
		return "2.3. Subcontractors. Developer may use a subcontractor after giving the Buyer written notice that identifies the proposed subcontractor and its role. Buyer may object in writing on reasonable data-protection, security, capability, or compliance grounds before work begins. Developer will bind every permitted subcontractor to obligations no less protective than this Agreement and remains responsible for its acts and omissions."
	default:
		return "2.3. Subcontractors. Developer may use a subcontractor only with the Buyer’s prior written consent. Developer will bind every approved subcontractor to obligations no less protective than this Agreement and remains responsible for its acts and omissions."
	}
}

func exclusivityClause(m documentModel) string {
	s := m.Customer.ShortName
	if m.Public == nil {
		return "4.2. Exclusivity. Developer retains no right to reuse, publish, sell, train on, license, disclose, or otherwise make available the Work Product, any derivative, or substantially identical capture to any third party. Developer may retain general-purpose tools, processes, and operational know-how that are not delivered as Work Product, but may not use them to circumvent an SOW exclusivity restriction."
	}
	switch m.Public.Exclusivity {
	case string(dto.PublicSOWAnswersExclusivityFullyExclusiveGlobal):
		return "4.2. Exclusivity. The Work Product and substantially identical captures are exclusive throughout the world. Developer retains no right to reuse, publish, sell, train on, license, disclose, or otherwise make them available to any third party."
	case string(dto.PublicSOWAnswersExclusivityExclusiveLimitedField):
		return "4.2. Exclusivity. The Work Product and substantially identical captures are exclusive only within the following field: " + m.Public.ExclusivityFieldScope + ". Outside that field, Developer may use independently created material that does not disclose " + s + " Confidential Information or include Work Product owned by " + s + "."
	default:
		return "4.2. Exclusivity. The commercial arrangement is non-exclusive. Ownership of delivered Work Product remains governed by Section 4.1, and Developer may undertake other engagements using independently created material that does not disclose " + s + " Confidential Information."
	}
}

func backupLicenseClause(m documentModel) string {
	s := m.Customer.ShortName
	licenseScope := "an exclusive, perpetual, irrevocable, worldwide"
	if m.Public != nil {
		switch m.Public.Exclusivity {
		case string(dto.PublicSOWAnswersExclusivityExclusiveLimitedField):
			licenseScope = "a perpetual, irrevocable, worldwide license that is exclusive within the field stated in Section 4.2"
		case string(dto.PublicSOWAnswersExclusivityNonExclusive):
			licenseScope = "a perpetual, irrevocable, worldwide license"
		}
	}
	return "4.3. Backup License. To the extent any right in the Work Product cannot be assigned, Developer grants " + s + " and its affiliates, customers, and sublicensees " + licenseScope + ", fully paid-up, royalty-free, transferable, sublicensable license to use, reproduce, host, modify, create derivatives of, distribute, commercialize, train, fine-tune, evaluate, improve, and otherwise exploit that Work Product for any purpose, including artificial-intelligence and machine-learning purposes."
}

func liabilityClause(m documentModel) string {
	t := m.Terms
	if m.Public == nil {
		return "11.1. Limitations. Except for indemnification, confidentiality breach, data-export breach, gross negligence, willful misconduct, and fraud (collectively, “Excluded Claims”), neither Party’s aggregate liability will exceed the fees paid or payable under the applicable SOW in the " + t.LookbackMonths + " months preceding the event giving rise to liability. Developer’s aggregate liability for Excluded Claims will not exceed " + t.ClaimsMultiplier + " times the fees paid or payable under the applicable SOW in that period; provided that Developer’s aggregate liability for claims arising from the collection, export, transfer, or delivery of Work Product in breach of Section 5, or from infringement of intellectual-property rights or violation of privacy, publicity, data-protection, consent, export, or security laws, will not exceed the greater of " + t.ClaimsFloor + " or " + t.ClaimsMultiplier + " times the fees paid or payable under the applicable SOW in that period. Neither Party will be liable for indirect, incidental, special, punitive, or consequential damages, except to the extent such damages are payable to a third party under an indemnified claim."
	}
	dataTier := "For claims arising from data compliance, privacy, security, consent, data transfer, or intellectual-property infringement, Developer’s aggregate liability will be the greater of " + m.Public.DataIPLiabilityFloorAmountText + " or " + m.Public.DataIPLiabilityMultiplierText + " times the fees paid or payable under the applicable SOW during that period."
	switch m.Public.DataIPLiabilityCap {
	case string(dto.PublicSOWAnswersDataIpLiabilityCapFeeMultipleOnly):
		dataTier = "For claims arising from data compliance, privacy, security, consent, data transfer, or intellectual-property infringement, Developer’s aggregate liability will be limited to " + m.Public.DataIPLiabilityMultiplierText + " times the fees paid or payable under the applicable SOW during that period."
	case string(dto.PublicSOWAnswersDataIpLiabilityCapNoSpecialCap):
		dataTier = "No separate special cap applies to data-compliance or intellectual-property claims, and those claims remain subject to the general aggregate liability cap."
	}
	return "11.1. Limitations. Except for gross negligence, willful misconduct, and fraud, each Party’s general aggregate liability under the applicable SOW will not exceed the fees paid or payable under that SOW in the 12 months preceding the event giving rise to liability. " + dataTier + " Neither Party will be liable for indirect, incidental, special, punitive, or consequential damages, except to the extent such damages are payable to a third party under an indemnified claim."
}

func jurisdictionClause(m documentModel) string {
	base := "12.4. Governing Law; Venue. This Agreement is governed by " + m.Customer.Law + " law. The state or federal courts located in " + m.Customer.Law + " have exclusive jurisdiction, subject to good-faith mediation before suit except where equitable relief is required."
	if m.Public == nil {
		return base
	}
	switch m.Public.JurisdictionRules {
	case string(dto.PublicSOWAnswersJurisdictionRulesAgreementCountryTerms):
		return base + " Applicable country-specific data-collection and data-transfer requirements will be documented as country terms added to this Agreement and signed by both Parties."
	case string(dto.PublicSOWAnswersJurisdictionRulesSeparateCountryRider):
		return base + " The Parties will document applicable country-specific data-collection and data-transfer requirements in a separate Jurisdiction Rider for each country before collection begins there."
	default:
		return base + " Each SOW will include a country appendix to each SOW that states the applicable data-collection and data-transfer requirements before collection begins in that country."
	}
}

func statementOfWorkElements(m documentModel) []documentElement {
	s, scope, price, env, technical, delivery := m.Customer.ShortName, m.Scope, m.Pricing, m.Environment, m.Technical, m.Delivery
	suffix, exclusiveText, audioText := "", "", "No ambient audio is permitted unless "+s+" approves it in writing."
	if m.Terms.Exclusive {
		suffix = " (EXCLUSIVE)"
		exclusiveText = "exclusive, "
	}
	if scope.AmbientAudio {
		audioText = "Ambient audio is permitted."
	}
	carveout := " subject to Section 4 of the Agreement."
	if delivery.RejectedStaysDeveloperOwned {
		carveout = ", subject to the rejected-material carveout in Section 9."
	}
	deposit := "There is no deposit, advance payment, or hardware contribution."
	if price.DepositRequired {
		deposit = s + " will pay a deposit of " + price.DepositAmount + " against the fees above. There is no other advance payment or hardware contribution."
	}
	firm := ""
	if price.FirmDeadline {
		firm = " The final-delivery date is a firm obligation, not a best-efforts target. If the " + scope.Volume + " " + scope.Unit + " are not delivered by that date, " + s + " may reduce or reallocate remaining scope or terminate this SOW on written notice, and will pay only for QC-passed " + scope.ShortUnit + " delivered by that date. Any extension requires " + s + "’s prior written approval."
	}
	condition := "Every technical item is a condition of acceptance."
	if technical.ConditionOfPayment {
		condition = "Every technical item is a condition of acceptance and payment."
	}
	deemed, rejected := "", ""
	if delivery.DeemedAcceptance {
		deemed = " A batch not rejected within the " + delivery.ReviewDays + "-day review period is deemed accepted only if Developer has first delivered the complete manifest and all required supporting materials."
	}
	if delivery.RejectedStaysDeveloperOwned {
		rejected = " Notwithstanding Section 4, material that " + s + " expressly rejects in writing through its quality-control review under this Section 9 and that Developer does not correct or replace so that " + s + " accepts it will not constitute Work Product and will remain Developer-owned. " + s + " will delete its copies of that material within " + delivery.DeletionDays + " days after rejection, except for copies required by applicable law or retained solely in routine backup systems until deleted in the ordinary course, and Developer may retain or otherwise use that material."
	}

	e := []documentElement{
		{Style: "Heading1", Text: "EXHIBIT A", PageBreak: true},
		p("Heading2", "SOW "+m.SOWNumber+" — "+strings.ToUpper(m.Terms.Title)+suffix),
		p("", "This Statement of Work (“SOW”) is entered into under the Content Development Agreement between "+m.Customer.LegalName+" and "+m.Developer.LegalName+" dated "+m.Terms.EffectiveDate+" (the “Agreement”). Capitalized terms not defined here have the meanings in the Agreement."),
		h("1. Overview"),
		p("", "Developer will collect, quality-control, and deliver to "+s+" "+scope.Volume+" "+scope.Unit+" of "+exclusiveText+scope.Description+" captured in "+scope.Venue+" in "+scope.Regions+". "+conditionalSentence(m.Terms.Exclusive, "The engagement and all Deliverables are exclusive. ")+"All raw, setup, sample, test, processed, and derived material collected in connection with this SOW is Work Product"+carveout),
	}
	if scope.Notes != "" {
		e = append(e, p("", scope.Notes))
	}
	e = append(e,
		h("2. Deliverables"), p("", "Each delivery will include: "+lettered(scope.Deliverables)+". "+audioText),
		h("3. Pricing"), table([][]string{{"Unit Fee", "Target Volume", "Total", "Payment Basis"}, {price.UnitFee + " per " + scope.UnitSingular, scope.Volume + " " + scope.Unit, "Maximum " + price.MaximumTotal, upperFirst(scope.Unit) + " only; " + cadenceBasis(price.Cadence) + "; Net " + price.PaymentDays}}),
		p("", s+" pays only for Deliverables that are received, accepted, and usable under this SOW. "+deposit),
		h("4. Delivery Schedule"), table(prependRow([]string{"Delivery Milestone", "Target Volume", "Deadline"}, milestoneRows(price.Milestones))),
		p("", "Beginning when collection starts, and no later than "+price.FirstDeadline+", Developer will upload completed Deliverables "+delivery.Cadence+" as they become available rather than holding them for larger batch delivery. Developer must upload the full "+scope.Volume+" "+scope.Unit+" by EOD "+price.FinalDeadline+"."+firm),
		h("5. Environment and Task Mix"), table(prependRow([]string{"Collection Verticals", "Target Hours or Percentage", "Excluded or Required Details"}, env.Verticals)),
		p("", limitsText(env, scope)), p("", capsText(env, scope)),
		h("6. Collection Requirements"),
	)
	number := 1
	if env.PlanRequired {
		e = append(e, p("", "6.1. Collection Plan. Before filming, Developer will submit a written collection plan identifying each planned site, environment category, operator count, planned hours, tasks, and difficulty split. No plan or update is effective until approved by "+s+" in writing. "+s+" may direct changes to collection verticals, environments, tasks, difficulty, site allocation, or operator allocation; Developer will implement them within "+env.ChangeHours+" hours unless "+s+" specifies otherwise."))
		number++
	}
	excluded := append([]string{"residential or household settings", "studio/staged/simulated environments", "long stretches of a single repeated motion at one station"}, lowerItems(env.ExtraExcluded)...)
	excluded = append(excluded, "material previously captured, sold, licensed, or delivered to another party")
	prohibited := append([]string{"minors", "nonconsenting bystanders", "credentials", "financial or health data", "confidential third-party information", "government-classified or controlled data", "restricted technical data", "precise mapping/geolocation data", "data from restricted/sensitive facilities"}, lowerItems(env.ExtraProhibited)...)
	e = append(e,
		p("", fmt.Sprintf("6.%d. Natural Activity. Recordings must depict genuine, varied, naturally performed work. No %s.", number, series(excluded, "or"))),
		p("", fmt.Sprintf("6.%d. Privacy and Sensitive Content. Developer will obtain documented, specific consent from each recorded individual for the recording, contemplated processing, and %s’s and its customers’ commercial and AI/ML uses where consent is required. Developer will obtain written permission from each site owner or authorized operator. No %s may be captured or delivered.", number+1, s, series(prohibited, "or"))),
		h("7. Technical and Quality Standards"), table(prependRow([]string{"Requirement", "Default Specification", "Rejection Criteria"}, technical.Requirements)),
		p("", condition+conditionalSentence(len(technical.Materials) > 0, " Before collection, Developer will provide "+series(lowerItems(technical.Materials), "and")+".")+" Material that does not meet every applicable requirement is non-conforming and not billable."),
		h("8. Delivery Location, Manifest, and Metadata"),
		p("", "Developer will deliver completed Deliverables "+delivery.Cadence+" to the "+s+"-designated cloud bucket using "+s+"-specified credentials, access controls, naming conventions, and manifest requirements. A batch is delivered only when accessible to "+s+" with all required data, metadata, calibration records, and consent/site-permission records. Developer will promptly correct failed, corrupted, incomplete, duplicated, or misdirected submissions at no additional charge."),
		table([][]string{{"Delivery / Manifest Field", "Required Value"}, {"Storage / transfer location", delivery.Storage}, {"Manifest format", delivery.ManifestFormat}, {"Required fields", series(lowerItemsFrom(delivery.ManifestFields, 1), "and") + "."}, {"Site data", series(lowerItemsFrom(delivery.SiteFields, 1), "and") + " per site, which " + s + " may assign."}, {"Integrity and security", delivery.Integrity}}),
		p("", "Missing, inaccurate, or inconsistent required metadata is non-conforming and may be rejected."),
		h("9. Acceptance"),
		p("", s+" will have "+delivery.ReviewDays+" days after receipt of a complete delivery batch and manifest to inspect, test, validate, and accept or reject the batch. "+upperFirst(article(scope.UnitSingular))+" “"+scope.UnitSingular+"” is "+article(scope.ShortSingular)+" "+scope.ShortSingular+" of unique footage accepted by "+s+" that satisfies all collection, technical, metadata, rights, privacy, consent, and quality requirements in this SOW. Duplicate, corrupt, non-conforming, or previously delivered footage is not payable. If "+s+" rejects material, Developer will, at "+s+"’s option, correct or replace it within "+delivery.CorrectionDays+" business days at no additional charge."+deemed+rejected),
		h("10. Invoicing and Payment"),
		p("", "Developer may invoice "+cadenceInvoice(price.Cadence)+" for "+scope.Unit+". Each valid, undisputed invoice is due Net "+price.PaymentDays+" days after receipt. Payments will be made in "+price.CurrencyName+" by "+price.PaymentMethods+". Developer will send invoices to "+price.InvoiceEmail+"."),
	)
	return append(e, signatureElements(m)...)
}

func signatureElements(m documentModel) []documentElement {
	return []documentElement{
		p("", "IN WITNESS WHEREOF, the parties have executed this Agreement as of the Effective Date."),
		table([][]string{{m.Customer.LegalName, m.Developer.LegalName}, {"Signature: ______________________________", "Signature: ______________________________"}, {"Name: " + m.Customer.SignerName, "Name: " + m.Developer.SignerName}, {"Title: " + m.Customer.SignerTitle, "Title: " + m.Developer.SignerTitle}, {"Date: __________________", "Date: __________________"}}),
	}
}

func renderDocumentXML(elements []documentElement) string {
	var body strings.Builder
	for _, element := range elements {
		if len(element.Rows) > 0 {
			writeTableXML(&body, element.Rows)
			continue
		}
		body.WriteString("<w:p>")
		if element.Style != "" {
			body.WriteString("<w:pPr><w:pStyle w:val=\"")
			body.WriteString(element.Style)
			body.WriteString("\"/></w:pPr>")
		}
		if element.PageBreak {
			body.WriteString("<w:r><w:br w:type=\"page\"/></w:r>")
		}
		writeParagraphRuns(&body, element.Text)
		body.WriteString("</w:p>")
	}
	body.WriteString(`<w:sectPr><w:headerReference w:type="default" r:id="rId2"/><w:footerReference w:type="default" r:id="rId3"/><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1280" w:right="1440" w:bottom="1320" w:left="1440" w:header="720" w:footer="720"/></w:sectPr>`)
	return xml.Header + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>` + body.String() + `</w:body></w:document>`
}

var subsectionPattern = regexp.MustCompile(`^(\d+\.\d+\. [^.]+\.)(.*)$`)

func writeParagraphRuns(body *strings.Builder, text string) {
	parts := subsectionPattern.FindStringSubmatch(text)
	if len(parts) == 3 {
		body.WriteString(`<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">`)
		escapeXML(body, parts[1])
		body.WriteString(`</w:t></w:r><w:r><w:t xml:space="preserve">`)
		escapeXML(body, parts[2])
		body.WriteString(`</w:t></w:r>`)
		return
	}
	body.WriteString(`<w:r><w:t xml:space="preserve">`)
	escapeXML(body, text)
	body.WriteString(`</w:t></w:r>`)
}

func writeTableXML(body *strings.Builder, rows [][]string) {
	body.WriteString(`<w:tbl><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4" w:color="B7B7B7"/><w:left w:val="single" w:sz="4" w:color="B7B7B7"/><w:bottom w:val="single" w:sz="4" w:color="B7B7B7"/><w:right w:val="single" w:sz="4" w:color="B7B7B7"/><w:insideH w:val="single" w:sz="4" w:color="B7B7B7"/><w:insideV w:val="single" w:sz="4" w:color="B7B7B7"/></w:tblBorders></w:tblPr>`)
	for rowIndex, row := range rows {
		body.WriteString("<w:tr>")
		for _, cell := range row {
			body.WriteString(`<w:tc><w:p><w:r>`)
			if rowIndex == 0 {
				body.WriteString(`<w:rPr><w:b/></w:rPr>`)
			}
			body.WriteString(`<w:t xml:space="preserve">`)
			escapeXML(body, cell)
			body.WriteString(`</w:t></w:r></w:p></w:tc>`)
		}
		body.WriteString("</w:tr>")
	}
	body.WriteString("</w:tbl>")
}

func escapeXML(body *strings.Builder, value string) {
	_ = xml.EscapeText(body, []byte(cleanText(value)))
}
func p(style, text string) documentElement  { return documentElement{Style: style, Text: text} }
func h(text string) documentElement         { return p("Heading1", text) }
func table(rows [][]string) documentElement { return documentElement{Rows: rows} }

func cleanText(value string) string {
	return strings.Map(func(r rune) rune {
		if isXMLValidRune(r) {
			return r
		}
		return -1
	}, value)
}

// isXMLValidRune reports whether r is a valid XML 1.0 character. The
// range 0xe000-0xfffd already excludes the 0xfffe/0xffff noncharacters,
// but every supplementary plane (0x10000-0x10ffff) has its own pair
// ending in 0xfffe/0xffff that must be excluded separately.
func isXMLValidRune(r rune) bool {
	switch {
	case r == '\t', r == '\n', r == '\r':
		return true
	case r >= 0x20 && r <= 0xd7ff:
		return true
	case r >= 0xe000 && r <= 0xfffd:
		return true
	case r >= 0x10000 && r <= 0x10ffff:
		return !isUnicodeNoncharacter(r)
	default:
		return false
	}
}

func isUnicodeNoncharacter(r rune) bool {
	low := r & 0xffff
	return low == 0xfffe || low == 0xffff
}

func textRows(rows []dto.ContractTextRow) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = cleanText(row.Text)
	}
	return out
}
func verticalRows(rows []dto.ContractVertical) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = []string{cleanText(row.Verticals), cleanText(row.Target), cleanText(row.Details)}
	}
	return out
}
func requirementRows(rows []dto.ContractRequirement) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = []string{cleanText(row.Requirement), cleanText(row.Specification), cleanText(row.Rejection)}
	}
	return out
}
func milestoneRows(rows []milestoneDocumentModel) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = []string{row.Name, row.Volume, row.Deadline}
	}
	return out
}
func prependRow(header []string, rows [][]string) [][]string {
	return append([][]string{header}, rows...)
}

func formatLongDate(value time.Time) string {
	if value.IsZero() {
		return "[date]"
	}
	return value.Format("January 2, 2006")
}
func formatNumber(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', 4, 64)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	return addThousands(formatted)
}
func formatMoney(value float32, code string, whole bool) string {
	digits := 2
	if whole && value == float32(int64(value)) {
		digits = 0
	}
	amount := addThousands(strconv.FormatFloat(float64(value), 'f', digits, 32))
	if code == "USD" {
		return "US $" + amount
	}
	return code + " " + amount
}
func addThousands(value string) string {
	parts := strings.SplitN(value, ".", 2)
	sign, integer := "", parts[0]
	if strings.HasPrefix(integer, "-") {
		sign, integer = "-", integer[1:]
	}
	for i := len(integer) - 3; i > 0; i -= 3 {
		integer = integer[:i] + "," + integer[i:]
	}
	if len(parts) == 2 {
		return sign + integer + "." + parts[1]
	}
	return sign + integer
}
func singularize(value string) string {
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	last := words[len(words)-1]
	lower := strings.ToLower(last)
	switch {
	case strings.HasSuffix(lower, "ies") && len(lower) > 3:
		last = last[:len(last)-3] + "y"
	case regexp.MustCompile(`(ses|xes|zes|ches|shes)$`).MatchString(lower):
		last = last[:len(last)-2]
	case strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss"):
		last = last[:len(last)-1]
	}
	words[len(words)-1] = last
	return strings.Join(words, " ")
}
func lastWord(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return value
	}
	return fields[len(fields)-1]
}
func article(value string) string {
	lower := strings.ToLower(value)
	if vowelSoundPattern.MatchString(lower) {
		return "an"
	}
	return "a"
}
func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	r := []rune(value)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
func upperFirst(value string) string {
	if value == "" {
		return ""
	}
	r := []rune(value)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
func conditionalSentence(condition bool, value string) string {
	if condition {
		return value
	}
	return ""
}

func series(items []string, conjunction string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " " + conjunction + " " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", " + conjunction + " " + items[len(items)-1]
}
func lettered(items []string) string {
	parts := make([]string, len(items))
	for i, item := range lowerItems(items) {
		parts[i] = "(" + string(rune('a'+i)) + ") " + item
	}
	if len(parts) <= 1 {
		return series(parts, "and")
	}
	return strings.Join(parts[:len(parts)-1], "; ") + "; and " + parts[len(parts)-1]
}
func lowerItems(items []string) []string { return lowerItemsFrom(items, 0) }
func lowerItemsFrom(items []string, from int) []string {
	out := append([]string(nil), items...)
	for i := from; i < len(out); i++ {
		if regexp.MustCompile(`^[A-Z][a-z]`).MatchString(out[i]) {
			out[i] = lowerFirst(out[i])
		}
	}
	return out
}
func currencyName(code string) string {
	if code == "USD" {
		return "U.S. dollars"
	}
	return code
}
func countryList(codes []string) string {
	names := make([]string, len(codes))
	for i, code := range codes {
		region, err := language.ParseRegion(code)
		if err != nil {
			names[i] = code
		} else {
			names[i] = display.English.Regions().Name(region)
		}
	}
	return series(names, "and")
}

func cadenceBasis(label string) string {
	switch label {
	case "Weekly in arrears":
		return "weekly invoicing in arrears"
	case "Every two weeks in arrears":
		return "invoicing every two weeks in arrears"
	case "Monthly in arrears":
		return "monthly invoicing in arrears"
	case "Per milestone":
		return "invoicing per milestone"
	case "On acceptance of each batch":
		return "invoicing on acceptance of each batch"
	default:
		return lowerFirst(label) + " invoicing"
	}
}
func cadenceInvoice(label string) string {
	switch label {
	case "Weekly in arrears":
		return "weekly in arrears"
	case "Every two weeks in arrears":
		return "every two weeks in arrears"
	case "Monthly in arrears":
		return "monthly in arrears"
	case "Per milestone":
		return "per milestone"
	case "On acceptance of each batch":
		return "on acceptance of each batch"
	default:
		return lowerFirst(label)
	}
}

func limitsText(env environmentDocumentModel, scope scopeDocumentModel) string {
	parts := []string{}
	for _, row := range env.Limits {
		switch row.Limit {
		case "Maximum from any single category":
			parts = append(parts, "No more than "+limitAmount(row, scope, false)+" may come from a single category.")
		case "Minimum distinct physical sites":
			parts = append(parts, "At least "+row.Value+" distinct physical sites are required.")
		}
	}
	mix := "The target task-difficulty mix is " + env.Difficulty[0] + " easy, " + env.Difficulty[1] + " medium, and " + env.Difficulty[2] + " hard"
	for _, row := range env.Limits {
		if row.Limit == "Minimum share performed by skilled workers doing their own occupation at their own workplace" {
			mix += "; at least " + row.Value + "% of " + scope.Unit + " must be performed by skilled workers doing their own occupation at their own workplace"
		}
	}
	parts = append(parts, mix+".")
	for _, row := range env.Limits {
		if !knownLimit(row.Limit) {
			parts = append(parts, cleanText(row.Limit)+": "+row.Value+conditionalSentence(row.Unit != "", " "+cleanText(row.Unit))+".")
		}
	}
	return strings.Join(parts, " ")
}
func capsText(env environmentDocumentModel, scope scopeDocumentModel) string {
	parts := []string{}
	for _, cap := range env.Caps {
		parts = append(parts, lowerFirst(cleanText(cap.Scope))+": easy "+cap.Easy+" "+scope.ShortUnit+", medium "+cap.Medium+" "+scope.ShortUnit+", hard "+cap.Hard+" "+scope.ShortUnit)
	}
	operator := findLimit(env.Limits, "Maximum per operator at one site across all tasks")
	caps := ""
	if len(parts) > 0 {
		caps = "Binding diversity caps: " + strings.Join(parts, "; ")
		if operator != nil {
			caps += "; and per operator at one site across all tasks: " + limitAmount(*operator, scope, true)
		}
		caps += "."
	} else if operator != nil {
		caps = "Binding diversity caps: per operator at one site across all tasks: " + limitAmount(*operator, scope, true) + "."
	}
	base := findLimit(env.Limits, "Maximum repetitive motion per task and site")
	repetitive := ""
	if base != nil {
		repetitive = "Repetitive motion is limited to " + limitAmount(*base, scope, true) + " per task/site"
		varied := findLimit(env.Limits, "Maximum repetitive motion per task and site where the worker moves through the space and the workflow genuinely varies")
		if varied != nil {
			repetitive += ", or " + limitAmount(*varied, scope, true) + " only where the worker moves through the space and the workflow genuinely varies."
		} else {
			repetitive += "."
		}
	}
	return strings.TrimSpace(caps + conditionalSentence(caps != "" && repetitive != "", " ") + repetitive)
}

func findLimit(rows []dto.ContractLimit, label string) *dto.ContractLimit {
	for i := range rows {
		if rows[i].Limit == label {
			return &rows[i]
		}
	}
	return nil
}

func knownLimit(label string) bool {
	switch label {
	case "Maximum from any single category", "Minimum distinct physical sites",
		"Minimum share performed by skilled workers doing their own occupation at their own workplace",
		"Maximum per operator at one site across all tasks", "Maximum repetitive motion per task and site",
		"Maximum repetitive motion per task and site where the worker moves through the space and the workflow genuinely varies":
		return true
	default:
		return false
	}
}

func limitAmount(row dto.ContractLimit, scope scopeDocumentModel, short bool) string {
	unit := cleanText(row.Unit)
	if unit == scope.Unit {
		unit = scope.Unit
		if short {
			unit = scope.ShortUnit
		}
		if row.Value == "1" {
			unit = scope.UnitSingular
			if short {
				unit = scope.ShortSingular
			}
		}
	}
	return strings.TrimSpace(row.Value + " " + unit)
}

func documentFilename(model documentModel) string {
	if strings.HasPrefix(model.SOWNumber, "[") {
		return "content-development-agreement-template.docx"
	}
	title := strings.ToLower(cleanText(model.Terms.Title))
	var b strings.Builder
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 80 {
		slug = strings.TrimRight(slug[:80], "-")
	}
	if slug == "" {
		slug = "agreement"
	}
	return "sow-" + model.SOWNumber + "-" + slug + ".docx"
}

const contentTypesXML = xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/><Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/></Types>`
const packageRelationshipsXML = xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
const corePropertiesXML = xml.Header + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>Content Development Agreement</dc:title><dc:creator>SoW</dc:creator><dcterms:created xsi:type="dcterms:W3CDTF">2026-01-01T00:00:00Z</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">2026-01-01T00:00:00Z</dcterms:modified></cp:coreProperties>`
const appPropertiesXML = xml.Header + `<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>SoW</Application></Properties>`
const stylesXML = xml.Header + `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:pPr><w:spacing w:after="120" w:line="252" w:lineRule="auto"/></w:pPr><w:rPr><w:rFonts w:ascii="Helvetica" w:hAnsi="Helvetica"/><w:sz w:val="21"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr><w:jc w:val="center"/><w:spacing w:after="240"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="240" w:after="80"/></w:pPr><w:rPr><w:b/><w:sz w:val="24"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="180" w:after="80"/></w:pPr><w:rPr><w:b/><w:sz w:val="22"/></w:rPr></w:style></w:styles>`

const documentRelationshipsXML = xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/></Relationships>`
const watermarkHeaderXML = xml.Header + `<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office"><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:pict><v:shapetype id="_x0000_t136" coordsize="21600,21600" o:spt="136" path="m@7,l@8,m@5,21600l@6,21600e"><v:textpath on="t" fitshape="t"/></v:shapetype><v:shape id="PowerPlusWaterMarkObject" type="#_x0000_t136" style="position:absolute;margin-left:0;margin-top:0;width:468pt;height:117pt;rotation:315;z-index:-251654144" fillcolor="#d9d9d9" stroked="f"><v:fill opacity=".08"/><v:textpath style="font-family:&quot;Helvetica&quot;;font-size:1pt" string="DRAFT"/></v:shape></w:pict></w:r></w:p></w:hdr>`

func footerXML(sowNumber string) string {
	var b strings.Builder
	b.WriteString(xml.Header + `<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Helvetica" w:hAnsi="Helvetica"/><w:sz w:val="16"/></w:rPr><w:t xml:space="preserve">SOW `)
	escapeXML(&b, sowNumber)
	b.WriteString(` · DRAFT · Page </w:t></w:r><w:fldSimple w:instr=" PAGE "><w:r><w:t>1</w:t></w:r></w:fldSimple><w:r><w:t xml:space="preserve"> of </w:t></w:r><w:fldSimple w:instr=" NUMPAGES "><w:r><w:t>1</w:t></w:r></w:fldSimple></w:p></w:ftr>`)
	return b.String()
}
