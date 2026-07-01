package properties

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
)

const (
	managerCertificateDocumentType            = "manager_certificate"
	managerCertificateExtractionKind          = "manager_certificate"
	managerCertificateExtractionSchemaVersion = "manager_certificate_source.v1"
	maxPropertyDocumentBytes                  = 25 * 1024 * 1024
	managerCertificateClaimConfidence         = 90
)

var (
	ErrPropertyDocumentInvalid  = errors.New("property document invalid")
	ErrPropertyDocumentTooLarge = errors.New("property document too large")
)

type normalizedPropertyDocumentUpload struct {
	Filename string
	MimeType string
	SHA256   string
	Bytes    []byte
}

type managerCertificateObject struct {
	Document       managerCertificateDocumentObject       `json:"document" description:"Document-level metadata and extraction warnings."`
	HousingCompany managerCertificateHousingCompanyObject `json:"housing_company" description:"Housing company facts from the certificate."`
	Building       managerCertificateBuildingObject       `json:"building" description:"Building and site facts from the certificate."`
	Unit           managerCertificateUnitObject           `json:"unit" description:"Apartment/unit facts from the certificate."`
	Finances       managerCertificateFinancesObject       `json:"finances" description:"Charges, debt, loans, and risk facts."`
	Renovations    []managerCertificateRenovationObject   `json:"renovations" description:"Completed and planned renovations, inspections, and major maintenance items."`
	Risks          managerCertificateRiskObject           `json:"risks" description:"Administrative, legal, financial, maintenance, and document-quality risk signals."`
}

type managerCertificateEvidenceObject struct {
	Text    string `json:"text,omitempty"`
	Page    *int32 `json:"page,omitempty"`
	Section string `json:"section,omitempty"`
}

type managerCertificateDocumentObject struct {
	DocumentDate    string                             `json:"document_date,omitempty" description:"Certificate or print date in YYYY-MM-DD when available."`
	Issuer          string                             `json:"issuer,omitempty" description:"Issuer or property manager name."`
	PropertyManager string                             `json:"property_manager,omitempty" description:"Isännöitsijä or management company."`
	Warnings        []string                           `json:"warnings,omitempty" description:"Ambiguities, missing sections, unreadable pages, or fields that need human review."`
	Evidence        []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateHousingCompanyObject struct {
	Name              string                             `json:"name,omitempty"`
	BusinessID        string                             `json:"business_id,omitempty"`
	BuildYear         *int32                             `json:"build_year,omitempty"`
	ApartmentCount    *int32                             `json:"apartment_count,omitempty"`
	PlotOwnershipType string                             `json:"plot_ownership_type,omitempty" enum:"owned,rented,unknown"`
	EnergyClass       string                             `json:"energy_class,omitempty"`
	Evidence          []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateBuildingObject struct {
	BuildYear      *int32                             `json:"build_year,omitempty"`
	FloorCount     *int32                             `json:"floor_count,omitempty"`
	ApartmentCount *int32                             `json:"apartment_count,omitempty"`
	EnergyClass    string                             `json:"energy_class,omitempty"`
	HeatingMethod  string                             `json:"heating_method,omitempty"`
	Material       string                             `json:"material,omitempty"`
	RoofType       string                             `json:"roof_type,omitempty"`
	RoofMaterial   string                             `json:"roof_material,omitempty"`
	Elevator       *bool                              `json:"elevator,omitempty"`
	Evidence       []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateUnitObject struct {
	ApartmentNumber      string                             `json:"apartment_number,omitempty"`
	Shares               string                             `json:"shares,omitempty"`
	AreaM2               *float64                           `json:"area_m2,omitempty"`
	RoomLayout           string                             `json:"room_layout,omitempty"`
	FloorLevel           *int32                             `json:"floor_level,omitempty"`
	MaintenanceCharge    *float64                           `json:"maintenance_charge_monthly,omitempty"`
	CapitalCharge        *float64                           `json:"capital_charge_monthly,omitempty"`
	TotalCharge          *float64                           `json:"total_charge_monthly,omitempty"`
	DebtShare            *float64                           `json:"debt_share_eur,omitempty"`
	ShareholderLiability string                             `json:"shareholder_liability,omitempty"`
	Evidence             []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateFinancesObject struct {
	FinancialRisk     string                             `json:"financial_risk,omitempty" enum:"unknown,low,medium,high"`
	MaintenanceRisk   string                             `json:"maintenance_risk,omitempty" enum:"unknown,low,medium,high"`
	RepairBacklogRisk string                             `json:"repair_backlog_risk,omitempty" enum:"unknown,low,medium,high"`
	LoanSummary       string                             `json:"loan_summary,omitempty"`
	ChargeSummary     string                             `json:"charge_summary,omitempty"`
	Loans             []managerCertificateLoanObject     `json:"loans,omitempty"`
	Charges           []managerCertificateChargeObject   `json:"charges,omitempty"`
	Evidence          []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateLoanObject struct {
	Name       string                             `json:"name,omitempty"`
	Lender     string                             `json:"lender,omitempty"`
	Purpose    string                             `json:"purpose,omitempty"`
	BalanceEUR *float64                           `json:"balance_eur,omitempty"`
	LimitEUR   *float64                           `json:"limit_eur,omitempty"`
	UsedEUR    *float64                           `json:"used_eur,omitempty"`
	AsOf       string                             `json:"as_of,omitempty"`
	Evidence   []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateChargeObject struct {
	ChargeType    string                             `json:"charge_type,omitempty" enum:"maintenance,capital,water,parking,sauna,storage,other,unknown"`
	Target        string                             `json:"target,omitempty" enum:"unit,housing_company,unknown"`
	Label         string                             `json:"label,omitempty"`
	AmountMonthly *float64                           `json:"amount_monthly,omitempty"`
	AmountPerM2   *float64                           `json:"amount_per_m2,omitempty"`
	Basis         string                             `json:"basis,omitempty"`
	LoanName      string                             `json:"loan_name,omitempty"`
	VATIncluded   *bool                              `json:"vat_included,omitempty"`
	Evidence      []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateRenovationObject struct {
	SystemType      string                             `json:"system_type" description:"pipe, water_supply, sewer, roof, facade, window, balcony, elevator, heating, ventilation, drainage, electricity, yard, common_areas, other."`
	Action          string                             `json:"action" enum:"replacement,repair,renovation,maintenance,inspection,condition_assessment,planning,installation,painting,cleaning,unknown"`
	SourceLabel     string                             `json:"source_label,omitempty"`
	Status          string                             `json:"status" enum:"done,planned,suspected,forecast,unknown"`
	Stage           string                             `json:"stage" enum:"unknown,study,condition_assessment,planning,tendering,execution,completed"`
	Scope           string                             `json:"scope" enum:"unknown,full,partial,maintenance"`
	Responsibility  string                             `json:"responsibility" enum:"unknown,housing_company,shareholder,mixed"`
	Year            *int32                             `json:"year,omitempty"`
	StartYear       *int32                             `json:"start_year,omitempty"`
	EndYear         *int32                             `json:"end_year,omitempty"`
	CostEstimateEUR *int64                             `json:"cost_estimate_eur,omitempty"`
	Summary         string                             `json:"summary,omitempty"`
	Evidence        []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

type managerCertificateRiskObject struct {
	AdministrativeLegalRisk string                             `json:"administrative_legal_risk,omitempty" enum:"unknown,low,medium,high"`
	Restrictions            []string                           `json:"restrictions,omitempty"`
	Disputes                []string                           `json:"disputes,omitempty"`
	MissingEvidence         []string                           `json:"missing_evidence,omitempty"`
	Evidence                []managerCertificateEvidenceObject `json:"evidence,omitempty"`
}

func normalizePropertyDocumentUpload(upload PropertyDocumentUpload) (normalizedPropertyDocumentUpload, error) {
	filename := cleanDisplayString(upload.Filename)
	if filename == "" {
		filename = "isannoitsijantodistus.pdf"
	}
	mimeType := strings.ToLower(strings.TrimSpace(upload.MimeType))
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "application/pdf"
	}
	if len(upload.Bytes) == 0 || !bytes.HasPrefix(upload.Bytes, []byte("%PDF-")) || mimeType != "application/pdf" {
		return normalizedPropertyDocumentUpload{}, ErrPropertyDocumentInvalid
	}
	if len(upload.Bytes) > maxPropertyDocumentBytes {
		return normalizedPropertyDocumentUpload{}, ErrPropertyDocumentTooLarge
	}
	hashBytes := sha256.Sum256(upload.Bytes)
	return normalizedPropertyDocumentUpload{Filename: filename, MimeType: mimeType, SHA256: hex.EncodeToString(hashBytes[:]), Bytes: upload.Bytes}, nil
}

func (s *Service) UploadManagerCertificate(ctx context.Context, input string, upload PropertyDocumentUpload) (PropertyDocumentSummary, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return PropertyDocumentSummary{}, ErrNotFound
	}
	normalized, err := normalizePropertyDocumentUpload(upload)
	if err != nil {
		return PropertyDocumentSummary{}, err
	}
	documentType := managerCertificateDocumentType
	sizeBytes := int64(len(normalized.Bytes))
	row, err := s.queries.CreatePropertyDocumentForOffering(ctx, db.CreatePropertyDocumentForOfferingParams{DocumentType: &documentType, Filename: &normalized.Filename, MimeType: &normalized.MimeType, SizeBytes: &sizeBytes, Sha256: &normalized.SHA256, DocumentBytes: normalized.Bytes, PropertyOfferingID: &offeringID})
	if err != nil {
		return PropertyDocumentSummary{}, mapNotFound(err)
	}
	return propertyDocumentSummaryFromCreateRow(row), nil
}

func (s *Service) UploadDetachedManagerCertificate(ctx context.Context, upload PropertyDocumentUpload) (PropertyDocumentSummary, error) {
	normalized, err := normalizePropertyDocumentUpload(upload)
	if err != nil {
		return PropertyDocumentSummary{}, err
	}
	documentType := managerCertificateDocumentType
	sizeBytes := int64(len(normalized.Bytes))
	row, err := s.queries.CreateDetachedPropertyDocument(ctx, db.CreateDetachedPropertyDocumentParams{DocumentType: &documentType, Filename: &normalized.Filename, MimeType: &normalized.MimeType, SizeBytes: &sizeBytes, Sha256: &normalized.SHA256, DocumentBytes: normalized.Bytes})
	if err != nil {
		return PropertyDocumentSummary{}, err
	}
	return propertyDocumentSummaryFromDetachedCreateRow(row), nil
}

func (s *Service) DownloadPropertyDocument(ctx context.Context, input string) (PropertyDocumentDownload, error) {
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return PropertyDocumentDownload{}, ErrNotFound
	}
	row, err := s.queries.GetPropertyDocumentDownload(ctx, &documentID)
	if err != nil {
		return PropertyDocumentDownload{}, mapNotFound(err)
	}
	return PropertyDocumentDownload{ID: row.PropertyDocumentID.String(), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, Bytes: row.PropertyDocumentBytes}, nil
}

func (s *Service) PropertyDocumentSummary(ctx context.Context, input string) (PropertyDocumentSummary, error) {
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return PropertyDocumentSummary{}, ErrNotFound
	}
	return s.propertyDocumentSummary(ctx, documentID)
}

func (s *Service) ExtractManagerCertificate(ctx context.Context, input string, modelName string) (ManagerCertificateExtractionResult, error) {
	source, err := s.ExtractManagerCertificateSource(ctx, input, modelName)
	if err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	return s.ProjectManagerCertificateExtraction(ctx, source.Document.ID)
}

func (s *Service) ExtractManagerCertificateSource(ctx context.Context, input string, modelName string) (ManagerCertificateSourceExtractionResult, error) {
	if strings.TrimSpace(s.managerCertificateAPIKey) == "" {
		return ManagerCertificateSourceExtractionResult{}, ErrManagerCertificateExtractorNotConfigured
	}
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return ManagerCertificateSourceExtractionResult{}, ErrNotFound
	}
	document, err := s.queries.GetPropertyDocumentForExtraction(ctx, &documentID)
	if err != nil {
		return ManagerCertificateSourceExtractionResult{}, mapNotFound(err)
	}
	modelName = firstNonEmpty(modelName, s.managerCertificateModelName, defaultOpenAIManagerCertificateModel)
	operation := propertyLLMOperationConfig("manager_certificate_extraction")
	status := "extracting"
	errorText := ""
	if err := s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: &status, ErrorText: &errorText, PropertyDocumentID: &documentID}); err != nil {
		return ManagerCertificateSourceExtractionResult{}, err
	}
	promptVersion := operation.Version
	runID, err := s.queries.CreatePropertyDocumentExtractionRun(ctx, db.CreatePropertyDocumentExtractionRunParams{PropertyDocumentID: &documentID, Model: &modelName, PromptVersion: &promptVersion})
	if err != nil {
		return ManagerCertificateSourceExtractionResult{}, err
	}
	extractor := openAIManagerCertificateExtractor{apiKey: s.managerCertificateAPIKey}
	extracted, rawJSON, err := extractor.Extract(ctx, document, operation, modelName)
	if err != nil {
		s.finishFailedDocumentExtraction(ctx, documentID, runID, err)
		return ManagerCertificateSourceExtractionResult{}, err
	}
	kind := managerCertificateExtractionKind
	schemaVersion := managerCertificateExtractionSchemaVersion
	if _, err := s.queries.UpsertPropertyDocumentExtraction(ctx, db.UpsertPropertyDocumentExtractionParams{PropertyDocumentID: &documentID, Kind: &kind, SchemaVersion: &schemaVersion, Model: &modelName, PromptVersion: &promptVersion, SourceJson: rawJSON}); err != nil {
		s.finishFailedDocumentExtraction(ctx, documentID, runID, err)
		return ManagerCertificateSourceExtractionResult{}, fmt.Errorf("store manager certificate source extraction: %w", err)
	}
	status = "succeeded"
	if err := s.queries.FinishPropertyDocumentExtractionRun(ctx, db.FinishPropertyDocumentExtractionRunParams{Status: &status, RawJson: rawJSON, ErrorText: &errorText, PropertyDocumentExtractionRunID: &runID}); err != nil {
		return ManagerCertificateSourceExtractionResult{}, err
	}
	summary, err := s.propertyDocumentSummary(ctx, documentID)
	if err != nil {
		return ManagerCertificateSourceExtractionResult{}, err
	}
	return ManagerCertificateSourceExtractionResult{Document: summary, Model: modelName, SchemaVersion: managerCertificateExtractionSchemaVersion, CreatedAt: time.Now().UTC().Format(time.RFC3339), RawJSON: rawJSON, Warnings: extracted.Document.Warnings}, nil
}

func (s *Service) AttachPropertyDocumentToOffering(ctx context.Context, documentInput string, offeringInput string) (PropertyDocumentSummary, error) {
	documentID, err := uuid.Parse(strings.TrimSpace(documentInput))
	if err != nil {
		return PropertyDocumentSummary{}, ErrNotFound
	}
	offeringID, err := uuid.Parse(strings.TrimSpace(offeringInput))
	if err != nil {
		return PropertyDocumentSummary{}, ErrNotFound
	}
	reason := "property_document_relinked"
	row, err := s.queries.AttachPropertyDocumentToOffering(ctx, db.AttachPropertyDocumentToOfferingParams{PropertyDocumentID: &documentID, PropertyOfferingID: &offeringID, Reason: &reason})
	if err != nil {
		return PropertyDocumentSummary{}, mapNotFound(err)
	}
	return propertyDocumentSummaryFromAttachRow(row), nil
}

func (s *Service) ProjectManagerCertificateExtraction(ctx context.Context, input string) (ManagerCertificateExtractionResult, error) {
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return ManagerCertificateExtractionResult{}, ErrNotFound
	}
	document, err := s.queries.GetPropertyDocumentForExtraction(ctx, &documentID)
	if err != nil {
		return ManagerCertificateExtractionResult{}, mapNotFound(err)
	}
	kind := managerCertificateExtractionKind
	extraction, err := s.queries.GetLatestPropertyDocumentExtraction(ctx, db.GetLatestPropertyDocumentExtractionParams{PropertyDocumentID: &documentID, Kind: &kind})
	if err != nil {
		return ManagerCertificateExtractionResult{}, mapNotFound(err)
	}
	var extracted managerCertificateObject
	if err := json.Unmarshal(extraction.PropertyDocumentExtractionSourceJson, &extracted); err != nil {
		return ManagerCertificateExtractionResult{}, fmt.Errorf("decode stored manager certificate extraction: %w", err)
	}
	claims, err := s.projectManagerCertificateExtraction(ctx, document, extracted, extraction.PropertyDocumentExtractionModel, extraction.PropertyDocumentExtractionPromptVersion, extraction.PropertyDocumentExtractionSourceJson)
	if err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	status := "extracted"
	errorText := ""
	if err := s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: &status, ErrorText: &errorText, PropertyDocumentID: &documentID}); err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	summary, err := s.propertyDocumentSummary(ctx, documentID)
	if err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	return ManagerCertificateExtractionResult{Document: summary, Model: extraction.PropertyDocumentExtractionModel, Claims: claims}, nil
}

func (s *Service) ExtractManagerCertificatePDF(ctx context.Context, upload PropertyDocumentUpload, modelName string) (ManagerCertificatePDFExtractionResult, error) {
	if strings.TrimSpace(s.managerCertificateAPIKey) == "" {
		return ManagerCertificatePDFExtractionResult{}, ErrManagerCertificateExtractorNotConfigured
	}
	filename := cleanDisplayString(upload.Filename)
	if filename == "" {
		filename = "isannoitsijantodistus.pdf"
	}
	mimeType := strings.ToLower(strings.TrimSpace(upload.MimeType))
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "application/pdf"
	}
	if len(upload.Bytes) == 0 || !bytes.HasPrefix(upload.Bytes, []byte("%PDF-")) || mimeType != "application/pdf" {
		return ManagerCertificatePDFExtractionResult{}, ErrPropertyDocumentInvalid
	}
	if len(upload.Bytes) > maxPropertyDocumentBytes {
		return ManagerCertificatePDFExtractionResult{}, ErrPropertyDocumentTooLarge
	}
	modelName = firstNonEmpty(modelName, s.managerCertificateModelName, defaultOpenAIManagerCertificateModel)
	operation := propertyLLMOperationConfig("manager_certificate_extraction")
	extractor := openAIManagerCertificateExtractor{apiKey: s.managerCertificateAPIKey}
	_, rawJSON, err := extractor.ExtractPDF(ctx, filename, upload.Bytes, operation, modelName)
	if err != nil {
		return ManagerCertificatePDFExtractionResult{}, err
	}
	return ManagerCertificatePDFExtractionResult{Filename: filename, Model: modelName, SchemaVersion: managerCertificateExtractionSchemaVersion, CreatedAt: time.Now().UTC().Format(time.RFC3339), RawJSON: rawJSON}, nil
}

func (s *Service) finishFailedDocumentExtraction(ctx context.Context, documentID uuid.UUID, runID uuid.UUID, cause error) {
	message := cause.Error()
	status := "failed"
	_ = s.queries.FinishPropertyDocumentExtractionRun(ctx, db.FinishPropertyDocumentExtractionRunParams{Status: &status, RawJson: json.RawMessage(`null`), ErrorText: &message, PropertyDocumentExtractionRunID: &runID})
	_ = s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: &status, ErrorText: &message, PropertyDocumentID: &documentID})
}

func (s *Service) ensureManagerCertificateDocumentTarget(ctx context.Context, queries *db.Queries, document db.GetPropertyDocumentForExtractionRow, extracted managerCertificateObject) (db.GetPropertyDocumentForExtractionRow, error) {
	if document.PropertyOfferingID != nil && document.PropertyUnitID != nil && document.HousingCompanyID != nil {
		return document, nil
	}
	documentKey := document.PropertyDocumentID.String()
	companyIdentityKey := "manager_certificate_document:" + documentKey + ":housing_company"
	if businessID := normalizeIdentityPart(extracted.HousingCompany.BusinessID); businessID != "" {
		companyIdentityKey = "business_id:" + businessID
	}
	companyID, err := queries.EnsureManagerCertificateHousingCompany(ctx, db.EnsureManagerCertificateHousingCompanyParams{
		IdentityKey:        &companyIdentityKey,
		Name:               optionalText(extracted.HousingCompany.Name),
		BusinessID:         optionalText(extracted.HousingCompany.BusinessID),
		BuildYear:          firstInt32(extracted.HousingCompany.BuildYear, extracted.Building.BuildYear),
		ApartmentCount:     firstInt32(extracted.HousingCompany.ApartmentCount, extracted.Building.ApartmentCount),
		EnergyClass:        optionalText(firstNonEmpty(extracted.HousingCompany.EnergyClass, extracted.Building.EnergyClass)),
		PropertyDocumentID: &documentKey,
	})
	if err != nil {
		return document, fmt.Errorf("ensure manager certificate housing company: %w", err)
	}
	buildingIdentityKey := "manager_certificate_document:" + documentKey + ":building"
	buildingID, err := queries.EnsureManagerCertificatePhysicalBuilding(ctx, db.EnsureManagerCertificatePhysicalBuildingParams{
		HousingCompanyID: &companyID,
		IdentityKey:      &buildingIdentityKey,
		BuildYear:        firstInt32(extracted.Building.BuildYear, extracted.HousingCompany.BuildYear),
		FloorCount:       extracted.Building.FloorCount,
		ApartmentCount:   firstInt32(extracted.Building.ApartmentCount, extracted.HousingCompany.ApartmentCount),
		Elevator:         extracted.Building.Elevator,
	})
	if err != nil {
		return document, fmt.Errorf("ensure manager certificate physical building: %w", err)
	}
	unitIdentityKey := "manager_certificate_document:" + documentKey + ":unit"
	unitID, err := queries.EnsureManagerCertificatePropertyUnit(ctx, db.EnsureManagerCertificatePropertyUnitParams{
		HousingCompanyID:   &companyID,
		PhysicalBuildingID: &buildingID,
		IdentityKey:        &unitIdentityKey,
		FloorLevel:         extracted.Unit.FloorLevel,
		AreaM2:             extracted.Unit.AreaM2,
		RoomsCount:         nil,
		RoomLayout:         optionalText(extracted.Unit.RoomLayout),
		LayoutMatchKey:     optionalText(normalizeIdentityPart(extracted.Unit.RoomLayout)),
		PropertyDocumentID: &documentKey,
	})
	if err != nil {
		return document, fmt.Errorf("ensure manager certificate property unit: %w", err)
	}
	if err := queries.SyncUnitFromPropertyUnit(ctx, &unitID); err != nil {
		return document, fmt.Errorf("sync manager certificate unit: %w", err)
	}
	offeringIdentityKey := "manager_certificate_document:" + documentKey + ":offering"
	headline := managerCertificateOfferingHeadline(extracted)
	offeringID, err := queries.EnsureManagerCertificatePropertyOffering(ctx, db.EnsureManagerCertificatePropertyOfferingParams{
		PropertyUnitID:     &unitID,
		IdentityKey:        &offeringIdentityKey,
		Headline:           &headline,
		PropertyDocumentID: &documentKey,
	})
	if err != nil {
		return document, fmt.Errorf("ensure manager certificate property offering: %w", err)
	}
	if err := queries.SyncListingFromPropertyOffering(ctx, &offeringID); err != nil {
		return document, fmt.Errorf("sync manager certificate listing: %w", err)
	}
	reason := "manager_certificate_target_created"
	if _, err := queries.AttachPropertyDocumentToOffering(ctx, db.AttachPropertyDocumentToOfferingParams{PropertyDocumentID: &document.PropertyDocumentID, PropertyOfferingID: &offeringID, Reason: &reason}); err != nil {
		return document, fmt.Errorf("attach manager certificate document to offering: %w", err)
	}
	updated, err := queries.GetPropertyDocumentForExtraction(ctx, &document.PropertyDocumentID)
	if err != nil {
		return document, fmt.Errorf("reload manager certificate document target: %w", err)
	}
	return updated, nil
}

func (s *Service) projectManagerCertificateExtraction(ctx context.Context, document db.GetPropertyDocumentForExtractionRow, extracted managerCertificateObject, modelName string, promptVersion string, rawJSON []byte) (int, error) {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return 0, fmt.Errorf("database handle does not support transactions")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin manager certificate transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	document, err = s.ensureManagerCertificateDocumentTarget(ctx, queries, document, extracted)
	if err != nil {
		return 0, err
	}
	if err := queries.DeleteLLMPropertyClaimsForDocument(ctx, &document.PropertyDocumentID); err != nil {
		return 0, fmt.Errorf("delete previous document claims: %w", err)
	}
	observedAt := managerCertificateObservedAt(extracted)
	claimWriter := managerCertificateClaimWriter{ctx: ctx, queries: queries, documentID: document.PropertyDocumentID, model: modelName, promptVersion: promptVersion, observedAt: observedAt}
	claimWriter.writeJSON("document", document.PropertyDocumentID, "document", "raw_extraction", rawJSON, "document", managerCertificateClaimConfidence, "Complete structured LLM extraction")
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "document_date", extracted.Document.DocumentDate, "document.document_date", managerCertificateClaimConfidence, evidenceText(extracted.Document.Evidence))
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "issuer", extracted.Document.Issuer, "document.issuer", managerCertificateClaimConfidence, evidenceText(extracted.Document.Evidence))
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "property_manager", extracted.Document.PropertyManager, "document.property_manager", managerCertificateClaimConfidence, evidenceText(extracted.Document.Evidence))
	claimWriter.writeStringSlice("document", document.PropertyDocumentID, "document", "warnings", extracted.Document.Warnings, "document.warnings", managerCertificateClaimConfidence, strings.Join(extracted.Document.Warnings, "; "))
	if document.HousingCompanyID != nil {
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "housing_company", "name", extracted.HousingCompany.Name, "housing_company.name", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "housing_company", "business_id", extracted.HousingCompany.BusinessID, "housing_company.business_id", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeNumberInt("housing_company", *document.HousingCompanyID, "housing_company", "build_year", extracted.HousingCompany.BuildYear, "housing_company.build_year", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeNumberInt("housing_company", *document.HousingCompanyID, "housing_company", "apartment_count", extracted.HousingCompany.ApartmentCount, "housing_company.apartment_count", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "site", "plot_ownership_type", normalizePlotOwnership(extracted.HousingCompany.PlotOwnershipType), "housing_company.plot_ownership_type", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "building", "energy_class", extracted.HousingCompany.EnergyClass, "housing_company.energy_class", managerCertificateClaimConfidence, evidenceText(extracted.HousingCompany.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "financial_risk", normalizeRiskLevel(extracted.Finances.FinancialRisk), "finances.financial_risk", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "maintenance_risk", normalizeRiskLevel(extracted.Finances.MaintenanceRisk), "finances.maintenance_risk", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "repair_backlog_risk", normalizeRiskLevel(extracted.Finances.RepairBacklogRisk), "finances.repair_backlog_risk", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "administrative_legal_risk", normalizeRiskLevel(extracted.Risks.AdministrativeLegalRisk), "risks.administrative_legal_risk", managerCertificateClaimConfidence, evidenceText(extracted.Risks.Evidence))
		claimWriter.writeStringSlice("housing_company", *document.HousingCompanyID, "risk", "restrictions", extracted.Risks.Restrictions, "risks.restrictions", managerCertificateClaimConfidence, evidenceText(extracted.Risks.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "finances", "loan_summary", extracted.Finances.LoanSummary, "finances.loan_summary", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "finances", "charge_summary", extracted.Finances.ChargeSummary, "finances.charge_summary", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeAnyJSON("housing_company", *document.HousingCompanyID, "finances", "loans", extracted.Finances.Loans, "finances.loans", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		claimWriter.writeAnyJSON("housing_company", *document.HousingCompanyID, "finances", "charges", extracted.Finances.Charges, "finances.charges", managerCertificateClaimConfidence, evidenceText(extracted.Finances.Evidence))
		if document.PropertyOfferingID != nil {
			if err := replaceManagerCertificateRenovations(ctx, tx, document.PropertyDocumentID, *document.HousingCompanyID, *document.PropertyOfferingID, observedAt, extracted.Renovations); err != nil {
				return 0, err
			}
		} else {
			return 0, fmt.Errorf("manager certificate document %s has no property offering after target resolution", document.PropertyDocumentID)
		}
	}
	if document.PhysicalBuildingID != nil {
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "build_year", firstInt32(extracted.Building.BuildYear, extracted.HousingCompany.BuildYear), "building.build_year", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "floor_count", extracted.Building.FloorCount, "building.floor_count", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "apartment_count", firstInt32(extracted.Building.ApartmentCount, extracted.HousingCompany.ApartmentCount), "building.apartment_count", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "energy_class", firstNonEmpty(extracted.Building.EnergyClass, extracted.HousingCompany.EnergyClass), "building.energy_class", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "heating_method", extracted.Building.HeatingMethod, "building.heating_method", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "material", extracted.Building.Material, "building.material", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "roof_type", extracted.Building.RoofType, "building.roof_type", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "roof_material", extracted.Building.RoofMaterial, "building.roof_material", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
		claimWriter.writeBool("physical_building", *document.PhysicalBuildingID, "building", "elevator", extracted.Building.Elevator, "building.elevator", managerCertificateClaimConfidence, evidenceText(extracted.Building.Evidence))
	}
	if document.PropertyUnitID != nil {
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "unit", "apartment_number", extracted.Unit.ApartmentNumber, "unit.apartment_number", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "unit", "shares", extracted.Unit.Shares, "unit.shares", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "unit", "area_m2", extracted.Unit.AreaM2, "unit.area_m2", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "layout", "room_layout", extracted.Unit.RoomLayout, "unit.room_layout", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumberInt("property_unit", *document.PropertyUnitID, "unit", "floor_level", extracted.Unit.FloorLevel, "unit.floor_level", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "maintenance_charge_monthly", extracted.Unit.MaintenanceCharge, "unit.maintenance_charge_monthly", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "capital_charge_monthly", extracted.Unit.CapitalCharge, "unit.capital_charge_monthly", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "total_charge_monthly", extracted.Unit.TotalCharge, "unit.total_charge_monthly", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "debt_share_eur", extracted.Unit.DebtShare, "unit.debt_share_eur", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "risk", "shareholder_liability", extracted.Unit.ShareholderLiability, "unit.shareholder_liability", managerCertificateClaimConfidence, evidenceText(extracted.Unit.Evidence))
	}
	if claimWriter.err != nil {
		return claimWriter.count, claimWriter.err
	}
	if document.PropertyOfferingID != nil {
		if _, err := queries.MarkPropertyOfferingDimensionTargetsDirty(ctx, db.MarkPropertyOfferingDimensionTargetsDirtyParams{PropertyOfferingID: *document.PropertyOfferingID, Reason: "document_claims_changed"}); err != nil {
			return claimWriter.count, fmt.Errorf("mark dimension targets dirty from document: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return claimWriter.count, fmt.Errorf("commit manager certificate transaction: %w", err)
	}
	return claimWriter.count, nil
}

type managerCertificateClaimWriter struct {
	ctx           context.Context
	queries       *db.Queries
	documentID    uuid.UUID
	model         string
	promptVersion string
	observedAt    *time.Time
	count         int
	err           error
}

func (w *managerCertificateClaimWriter) writeText(entityType string, entityID uuid.UUID, section string, key string, value string, sourceField string, confidence int32, evidence string) {
	value = cleanDisplayString(value)
	if value == "" || w.err != nil {
		return
	}
	w.insert(entityType, entityID, section, key, "text", value, nil, nil, nil, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeNumber(entityType string, entityID uuid.UUID, section string, key string, value *float64, sourceField string, confidence int32, evidence string) {
	if value == nil || w.err != nil {
		return
	}
	w.insert(entityType, entityID, section, key, "number", "", value, nil, nil, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeNumberInt(entityType string, entityID uuid.UUID, section string, key string, value *int32, sourceField string, confidence int32, evidence string) {
	if value == nil || w.err != nil {
		return
	}
	number := float64(*value)
	w.writeNumber(entityType, entityID, section, key, &number, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeBool(entityType string, entityID uuid.UUID, section string, key string, value *bool, sourceField string, confidence int32, evidence string) {
	if value == nil || w.err != nil {
		return
	}
	w.insert(entityType, entityID, section, key, "bool", "", nil, value, nil, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeJSON(entityType string, entityID uuid.UUID, section string, key string, value []byte, sourceField string, confidence int32, evidence string) {
	if len(value) == 0 || w.err != nil {
		return
	}
	w.insert(entityType, entityID, section, key, "json", "", nil, nil, value, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeAnyJSON(entityType string, entityID uuid.UUID, section string, key string, value any, sourceField string, confidence int32, evidence string) {
	if w.err != nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		w.err = err
		return
	}
	if string(data) == "null" || string(data) == "[]" || string(data) == "{}" {
		return
	}
	w.writeJSON(entityType, entityID, section, key, data, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) writeStringSlice(entityType string, entityID uuid.UUID, section string, key string, values []string, sourceField string, confidence int32, evidence string) {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = cleanDisplayString(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	if len(cleaned) == 0 || w.err != nil {
		return
	}
	data, err := json.Marshal(cleaned)
	if err != nil {
		w.err = err
		return
	}
	w.insert(entityType, entityID, section, key, "json", "", nil, nil, data, sourceField, confidence, evidence)
}

func (w *managerCertificateClaimWriter) insert(entityType string, entityID uuid.UUID, section string, key string, valueKind string, valueText string, valueNumber *float64, valueBool *bool, valueJSON json.RawMessage, sourceField string, confidence int32, evidence string) {
	if confidence <= 0 {
		confidence = 50
	}
	if confidence > 100 {
		confidence = 100
	}
	confidenceValue := float64(confidence)
	evidenceText := cleanDisplayString(evidence)
	w.err = w.queries.InsertDocumentPropertyClaim(w.ctx, db.InsertDocumentPropertyClaimParams{EntityType: &entityType, EntityID: &entityID, Section: &section, Key: &key, ValueKind: &valueKind, ValueText: &valueText, ValueNumber: valueNumber, ValueBool: valueBool, ValueJson: valueJSON, PropertyDocumentID: &w.documentID, SourceField: &sourceField, SourceObservedAt: w.observedAt, EvidenceText: &evidenceText, Confidence: &confidenceValue, Model: &w.model, PromptVersion: &w.promptVersion})
	if w.err == nil {
		w.count++
	}
}

func replaceManagerCertificateRenovations(ctx context.Context, tx pgx.Tx, documentID uuid.UUID, housingCompanyID uuid.UUID, offeringID uuid.UUID, observedAt *time.Time, items []managerCertificateRenovationObject) error {
	queries := db.New(tx)
	runID, err := queries.CreateManagerCertificateRenovationProjectionRun(ctx, &documentID)
	if err != nil {
		return fmt.Errorf("create manager certificate renovation projection run: %w", err)
	}
	if err := queries.DeleteManagerCertificateRenovationEvents(ctx, db.DeleteManagerCertificateRenovationEventsParams{HousingCompanyID: &housingCompanyID, PropertyDocumentID: &documentID}); err != nil {
		return fmt.Errorf("delete previous manager certificate renovations: %w", err)
	}
	for _, item := range items {
		category := normalizeManagerCertificateRenovationCategory(item.SystemType)
		if category == "" || cleanDisplayString(item.Summary) == "" {
			continue
		}
		status := normalizeManagerCertificateRenovationStatus(item.Status)
		stage := normalizeManagerCertificateRenovationStage(item.Stage)
		scope := normalizeManagerCertificateRenovationScope(item.Scope)
		responsibility := normalizeRenovationResponsibility(item.Responsibility)
		summary := cleanDisplayString(firstNonEmpty(item.Summary, item.SourceLabel))
		sourceLabel := cleanDisplayString(item.SourceLabel)
		action := normalizeManagerCertificateRenovationAction(item.Action)
		evidenceText := evidenceText(item.Evidence)
		if err := queries.InsertManagerCertificateRenovationEvent(ctx, db.InsertManagerCertificateRenovationEventParams{PropertyDimensionProjectionRunID: &runID, HousingCompanyID: &housingCompanyID, PropertyDocumentID: &documentID, Category: &category, Status: &status, Stage: emptyToNil(stage), Scope: emptyToNil(scope), Responsibility: emptyToNil(responsibility), Year: item.Year, StartYear: item.StartYear, EndYear: item.EndYear, CostEstimateEur: item.CostEstimateEUR, Summary: emptyToNil(summary), SourceLabel: &sourceLabel, Action: &action, EvidenceText: &evidenceText, SourceObservedAt: observedAt}); err != nil {
			return fmt.Errorf("insert manager certificate renovation: %w", err)
		}
	}
	if _, err := queries.MarkPropertyOfferingDimensionTargetsDirty(ctx, db.MarkPropertyOfferingDimensionTargetsDirtyParams{PropertyOfferingID: offeringID, Reason: "document_renovation_events_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty from document renovations: %w", err)
	}
	return nil
}

func (s *Service) propertyDocumentSummary(ctx context.Context, documentID uuid.UUID) (PropertyDocumentSummary, error) {
	row, err := s.queries.GetPropertyDocumentSummary(ctx, &documentID)
	if err != nil {
		return PropertyDocumentSummary{}, mapNotFound(err)
	}
	return propertyDocumentSummaryFromGetRow(row), nil
}

func (s *Service) enrichSaleListingDocuments(ctx context.Context, listing *SaleListing, offeringID uuid.UUID) error {
	rows, err := s.queries.ListPropertyDocumentsForOffering(ctx, &offeringID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.Documents = append(listing.Documents, propertyDocumentSummaryFromListRow(row))
	}
	return nil
}

func propertyDocumentSummaryFromCreateRow(row db.CreatePropertyDocumentForOfferingRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: ptrUUIDString(row.PropertyOfferingID), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentSummaryFromDetachedCreateRow(row db.CreateDetachedPropertyDocumentRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: ptrUUIDString(row.PropertyOfferingID), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentSummaryFromAttachRow(row db.AttachPropertyDocumentToOfferingRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: ptrUUIDString(row.PropertyOfferingID), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentSummaryFromGetRow(row db.GetPropertyDocumentSummaryRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: ptrUUIDString(row.PropertyOfferingID), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentSummaryFromListRow(row db.ListPropertyDocumentsForOfferingRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: ptrUUIDString(row.PropertyOfferingID), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentDownloadURL(id uuid.UUID) string {
	return "/api/v1/property-documents/" + id.String() + "/download"
}

func timePtrString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func optionalText(value string) *string {
	cleaned := cleanDisplayString(value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func managerCertificateOfferingHeadline(extracted managerCertificateObject) string {
	parts := compactStrings([]string{extracted.HousingCompany.Name, extracted.Unit.ApartmentNumber, extracted.Unit.RoomLayout})
	if len(parts) == 0 {
		return "Manager certificate offering"
	}
	return strings.Join(parts, " ")
}

func managerCertificateObservedAt(extracted managerCertificateObject) *time.Time {
	value := cleanDisplayString(extracted.Document.DocumentDate)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	observed := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, time.UTC)
	return &observed
}

func normalizeRiskLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "matala", "pieni":
		return "low"
	case "medium", "keskitaso", "kohtalainen":
		return "medium"
	case "high", "korkea", "suuri":
		return "high"
	case "unknown", "tuntematon":
		return "unknown"
	default:
		return ""
	}
}

func normalizePlotOwnership(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "owned", "own", "oma", "omistettu":
		return "owned"
	case "rented", "rent", "lease", "vuokra", "vuokratontti":
		return "rented"
	case "unknown", "tuntematon":
		return "unknown"
	default:
		return cleanDisplayString(value)
	}
}

func normalizeManagerCertificateRenovationCategory(value string) string {
	value = normalizeRenovationCategory(value)
	switch value {
	case "pipe", "water_supply", "sewer", "roof", "facade", "window", "balcony", "elevator", "heating", "ventilation", "drainage", "electricity", "yard", "common_areas", "other":
		return value
	case "windows":
		return "window"
	case "common_area":
		return "common_areas"
	default:
		return "other"
	}
}

func normalizeManagerCertificateRenovationStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "done", "completed", "valmis", "tehty":
		return "done"
	case "planned", "suunniteltu":
		return "planned"
	case "suspected", "forecast", "unknown", "cancelled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func normalizeManagerCertificateRenovationStage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "study", "condition_assessment", "planning", "tendering", "execution", "completed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func normalizeManagerCertificateRenovationScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "full", "partial", "maintenance":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func normalizeManagerCertificateRenovationAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "replacement", "repair", "renovation", "maintenance", "inspection", "condition_assessment", "planning", "installation", "painting", "cleaning":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func evidenceText(items []managerCertificateEvidenceObject) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		text := cleanDisplayString(item.Text)
		if text == "" {
			continue
		}
		if item.Page != nil && *item.Page > 0 {
			text = fmt.Sprintf("%s (s.%d)", text, *item.Page)
		}
		if section := cleanDisplayString(item.Section); section != "" {
			text = section + ": " + text
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "; ")
}
