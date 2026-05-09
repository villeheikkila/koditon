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
	managerCertificateDocumentType = "manager_certificate"
	maxPropertyDocumentBytes       = 25 * 1024 * 1024
)

var ErrPropertyDocumentInvalid = errors.New("property document invalid")
var ErrPropertyDocumentTooLarge = errors.New("property document too large")

type managerCertificateObject struct {
	Document       managerCertificateDocumentObject       `json:"document" description:"Document-level metadata and extraction warnings."`
	HousingCompany managerCertificateHousingCompanyObject `json:"housing_company" description:"Housing company facts from the certificate."`
	Building       managerCertificateBuildingObject       `json:"building" description:"Building and site facts from the certificate."`
	Unit           managerCertificateUnitObject           `json:"unit" description:"Apartment/unit facts from the certificate."`
	Finances       managerCertificateFinancesObject       `json:"finances" description:"Charges, debt, loans, and risk facts."`
	Renovations    []managerCertificateRenovationObject   `json:"renovations" description:"Completed and planned renovations, inspections, and major maintenance items."`
	Risks          managerCertificateRiskObject           `json:"risks" description:"Administrative, legal, financial, maintenance, and document-quality risk signals."`
}

type managerCertificateDocumentObject struct {
	DocumentDate    string   `json:"document_date,omitempty" description:"Certificate or print date in YYYY-MM-DD when available."`
	Issuer          string   `json:"issuer,omitempty" description:"Issuer or property manager name."`
	PropertyManager string   `json:"property_manager,omitempty" description:"Isännöitsijä or management company."`
	Warnings        []string `json:"warnings,omitempty" description:"Ambiguities, missing sections, unreadable pages, or fields that need human review."`
	Confidence      int32    `json:"confidence" description:"Overall extraction confidence from 0 to 100."`
}

type managerCertificateHousingCompanyObject struct {
	Name              string `json:"name,omitempty"`
	BusinessID        string `json:"business_id,omitempty"`
	BuildYear         *int32 `json:"build_year,omitempty"`
	ApartmentCount    *int32 `json:"apartment_count,omitempty"`
	PlotOwnershipType string `json:"plot_ownership_type,omitempty" enum:"owned,rented,unknown"`
	EnergyClass       string `json:"energy_class,omitempty"`
	Evidence          string `json:"evidence,omitempty"`
	Confidence        int32  `json:"confidence"`
}

type managerCertificateBuildingObject struct {
	BuildYear      *int32 `json:"build_year,omitempty"`
	FloorCount     *int32 `json:"floor_count,omitempty"`
	ApartmentCount *int32 `json:"apartment_count,omitempty"`
	EnergyClass    string `json:"energy_class,omitempty"`
	HeatingMethod  string `json:"heating_method,omitempty"`
	Material       string `json:"material,omitempty"`
	RoofType       string `json:"roof_type,omitempty"`
	RoofMaterial   string `json:"roof_material,omitempty"`
	Elevator       *bool  `json:"elevator,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
	Confidence     int32  `json:"confidence"`
}

type managerCertificateUnitObject struct {
	ApartmentNumber      string   `json:"apartment_number,omitempty"`
	Shares               string   `json:"shares,omitempty"`
	AreaM2               *float64 `json:"area_m2,omitempty"`
	RoomLayout           string   `json:"room_layout,omitempty"`
	FloorLevel           *int32   `json:"floor_level,omitempty"`
	MaintenanceCharge    *float64 `json:"maintenance_charge_monthly,omitempty"`
	CapitalCharge        *float64 `json:"capital_charge_monthly,omitempty"`
	TotalCharge          *float64 `json:"total_charge_monthly,omitempty"`
	DebtShare            *float64 `json:"debt_share_eur,omitempty"`
	ShareholderLiability string   `json:"shareholder_liability,omitempty"`
	Evidence             string   `json:"evidence,omitempty"`
	Confidence           int32    `json:"confidence"`
}

type managerCertificateFinancesObject struct {
	FinancialRisk     string `json:"financial_risk,omitempty" enum:"unknown,low,medium,high"`
	MaintenanceRisk   string `json:"maintenance_risk,omitempty" enum:"unknown,low,medium,high"`
	RepairBacklogRisk string `json:"repair_backlog_risk,omitempty" enum:"unknown,low,medium,high"`
	LoanSummary       string `json:"loan_summary,omitempty"`
	ChargeSummary     string `json:"charge_summary,omitempty"`
	Evidence          string `json:"evidence,omitempty"`
	Confidence        int32  `json:"confidence"`
}

type managerCertificateRenovationObject struct {
	Category        string `json:"category" description:"pipe, water_supply, sewer, roof, facade, window, balcony, elevator, heating, ventilation, drainage, electricity, yard, common_areas, other."`
	Status          string `json:"status" enum:"done,planned,suspected,forecast,unknown"`
	Stage           string `json:"stage" enum:"unknown,study,condition_assessment,planning,tendering,execution,completed"`
	Scope           string `json:"scope" enum:"unknown,full,partial,maintenance"`
	Responsibility  string `json:"responsibility" enum:"unknown,housing_company,shareholder,mixed"`
	Year            *int32 `json:"year,omitempty"`
	StartYear       *int32 `json:"start_year,omitempty"`
	EndYear         *int32 `json:"end_year,omitempty"`
	CostEstimateEUR *int64 `json:"cost_estimate_eur,omitempty"`
	Summary         string `json:"summary,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
	Confidence      int32  `json:"confidence"`
}

type managerCertificateRiskObject struct {
	AdministrativeLegalRisk string   `json:"administrative_legal_risk,omitempty" enum:"unknown,low,medium,high"`
	Restrictions            []string `json:"restrictions,omitempty"`
	Disputes                []string `json:"disputes,omitempty"`
	MissingEvidence         []string `json:"missing_evidence,omitempty"`
	Evidence                string   `json:"evidence,omitempty"`
	Confidence              int32    `json:"confidence"`
}

func (s *Service) UploadManagerCertificate(ctx context.Context, input string, upload PropertyDocumentUpload) (PropertyDocumentSummary, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return PropertyDocumentSummary{}, ErrNotFound
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
		return PropertyDocumentSummary{}, ErrPropertyDocumentInvalid
	}
	if len(upload.Bytes) > maxPropertyDocumentBytes {
		return PropertyDocumentSummary{}, ErrPropertyDocumentTooLarge
	}
	hashBytes := sha256.Sum256(upload.Bytes)
	row, err := s.queries.CreatePropertyDocumentForOffering(ctx, db.CreatePropertyDocumentForOfferingParams{DocumentType: managerCertificateDocumentType, Filename: filename, MimeType: mimeType, SizeBytes: int64(len(upload.Bytes)), Sha256: hex.EncodeToString(hashBytes[:]), DocumentBytes: upload.Bytes, PropertyOfferingID: offeringID})
	if err != nil {
		return PropertyDocumentSummary{}, mapNotFound(err)
	}
	return propertyDocumentSummaryFromCreateRow(row), nil
}

func (s *Service) DownloadPropertyDocument(ctx context.Context, input string) (PropertyDocumentDownload, error) {
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return PropertyDocumentDownload{}, ErrNotFound
	}
	row, err := s.queries.GetPropertyDocumentDownload(ctx, documentID)
	if err != nil {
		return PropertyDocumentDownload{}, mapNotFound(err)
	}
	return PropertyDocumentDownload{ID: row.PropertyDocumentID.String(), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, Bytes: row.PropertyDocumentBytes}, nil
}

func (s *Service) ExtractManagerCertificate(ctx context.Context, input string, modelName string) (ManagerCertificateExtractionResult, error) {
	if strings.TrimSpace(s.managerCertificateAPIKey) == "" {
		return ManagerCertificateExtractionResult{}, ErrManagerCertificateExtractorNotConfigured
	}
	documentID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return ManagerCertificateExtractionResult{}, ErrNotFound
	}
	document, err := s.queries.GetPropertyDocumentForExtraction(ctx, documentID)
	if err != nil {
		return ManagerCertificateExtractionResult{}, mapNotFound(err)
	}
	modelName = firstNonEmpty(modelName, s.managerCertificateModelName, defaultOpenAIManagerCertificateModel)
	operation := propertyLLMOperationConfig("manager_certificate_extraction")
	if err := s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: "extracting", ErrorText: "", PropertyDocumentID: documentID}); err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	runID, err := s.queries.CreatePropertyDocumentExtractionRun(ctx, db.CreatePropertyDocumentExtractionRunParams{PropertyDocumentID: documentID, Model: modelName, PromptVersion: operation.Version})
	if err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	extractor := openAIManagerCertificateExtractor{apiKey: s.managerCertificateAPIKey}
	extracted, rawJSON, err := extractor.Extract(ctx, document, operation, modelName)
	if err != nil {
		s.finishFailedDocumentExtraction(ctx, documentID, runID, err)
		return ManagerCertificateExtractionResult{}, err
	}
	claims, err := s.persistManagerCertificateExtraction(ctx, document, extracted, modelName, operation.Version, rawJSON)
	if err != nil {
		s.finishFailedDocumentExtraction(ctx, documentID, runID, err)
		return ManagerCertificateExtractionResult{}, err
	}
	if err := s.queries.FinishPropertyDocumentExtractionRun(ctx, db.FinishPropertyDocumentExtractionRunParams{Status: "succeeded", RawJson: rawJSON, ErrorText: "", PropertyDocumentExtractionRunID: runID}); err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	if err := s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: "extracted", ErrorText: "", PropertyDocumentID: documentID}); err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	summary, err := s.propertyDocumentSummary(ctx, documentID)
	if err != nil {
		return ManagerCertificateExtractionResult{}, err
	}
	return ManagerCertificateExtractionResult{Document: summary, Model: modelName, Claims: claims}, nil
}

func (s *Service) finishFailedDocumentExtraction(ctx context.Context, documentID uuid.UUID, runID uuid.UUID, cause error) {
	message := cause.Error()
	_ = s.queries.FinishPropertyDocumentExtractionRun(ctx, db.FinishPropertyDocumentExtractionRunParams{Status: "failed", RawJson: json.RawMessage(`null`), ErrorText: message, PropertyDocumentExtractionRunID: runID})
	_ = s.queries.UpdatePropertyDocumentExtractionStatus(ctx, db.UpdatePropertyDocumentExtractionStatusParams{Status: "failed", ErrorText: message, PropertyDocumentID: documentID})
}

func (s *Service) persistManagerCertificateExtraction(ctx context.Context, document db.GetPropertyDocumentForExtractionRow, extracted managerCertificateObject, modelName string, promptVersion string, rawJSON []byte) (int, error) {
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
	if err := queries.DeleteLLMPropertyClaimsForDocument(ctx, document.PropertyDocumentID); err != nil {
		return 0, fmt.Errorf("delete previous document claims: %w", err)
	}
	claimWriter := managerCertificateClaimWriter{ctx: ctx, queries: queries, documentID: document.PropertyDocumentID, model: modelName, promptVersion: promptVersion}
	claimWriter.writeJSON("document", document.PropertyDocumentID, "document", "raw_extraction", rawJSON, "document", extracted.Document.Confidence, "Complete structured LLM extraction")
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "document_date", extracted.Document.DocumentDate, "document.document_date", extracted.Document.Confidence, extracted.Document.Issuer)
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "issuer", extracted.Document.Issuer, "document.issuer", extracted.Document.Confidence, extracted.Document.Issuer)
	claimWriter.writeText("document", document.PropertyDocumentID, "document", "property_manager", extracted.Document.PropertyManager, "document.property_manager", extracted.Document.Confidence, extracted.Document.PropertyManager)
	claimWriter.writeStringSlice("document", document.PropertyDocumentID, "document", "warnings", extracted.Document.Warnings, "document.warnings", extracted.Document.Confidence, strings.Join(extracted.Document.Warnings, "; "))
	if document.HousingCompanyID != nil {
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "housing_company", "name", extracted.HousingCompany.Name, "housing_company.name", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "housing_company", "business_id", extracted.HousingCompany.BusinessID, "housing_company.business_id", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeNumberInt("housing_company", *document.HousingCompanyID, "housing_company", "build_year", extracted.HousingCompany.BuildYear, "housing_company.build_year", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeNumberInt("housing_company", *document.HousingCompanyID, "housing_company", "apartment_count", extracted.HousingCompany.ApartmentCount, "housing_company.apartment_count", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "site", "plot_ownership_type", normalizePlotOwnership(extracted.HousingCompany.PlotOwnershipType), "housing_company.plot_ownership_type", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "building", "energy_class", extracted.HousingCompany.EnergyClass, "housing_company.energy_class", extracted.HousingCompany.Confidence, extracted.HousingCompany.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "financial_risk", normalizeRiskLevel(extracted.Finances.FinancialRisk), "finances.financial_risk", extracted.Finances.Confidence, extracted.Finances.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "maintenance_risk", normalizeRiskLevel(extracted.Finances.MaintenanceRisk), "finances.maintenance_risk", extracted.Finances.Confidence, extracted.Finances.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "repair_backlog_risk", normalizeRiskLevel(extracted.Finances.RepairBacklogRisk), "finances.repair_backlog_risk", extracted.Finances.Confidence, extracted.Finances.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "risk", "administrative_legal_risk", normalizeRiskLevel(extracted.Risks.AdministrativeLegalRisk), "risks.administrative_legal_risk", extracted.Risks.Confidence, extracted.Risks.Evidence)
		claimWriter.writeStringSlice("housing_company", *document.HousingCompanyID, "risk", "restrictions", extracted.Risks.Restrictions, "risks.restrictions", extracted.Risks.Confidence, extracted.Risks.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "finances", "loan_summary", extracted.Finances.LoanSummary, "finances.loan_summary", extracted.Finances.Confidence, extracted.Finances.Evidence)
		claimWriter.writeText("housing_company", *document.HousingCompanyID, "finances", "charge_summary", extracted.Finances.ChargeSummary, "finances.charge_summary", extracted.Finances.Confidence, extracted.Finances.Evidence)
		if err := replaceManagerCertificateRenovations(ctx, tx, *document.HousingCompanyID, document.PropertyOfferingID, extracted.Renovations); err != nil {
			return 0, err
		}
	}
	if document.PhysicalBuildingID != nil {
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "build_year", firstInt32(extracted.Building.BuildYear, extracted.HousingCompany.BuildYear), "building.build_year", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "floor_count", extracted.Building.FloorCount, "building.floor_count", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeNumberInt("physical_building", *document.PhysicalBuildingID, "building", "apartment_count", firstInt32(extracted.Building.ApartmentCount, extracted.HousingCompany.ApartmentCount), "building.apartment_count", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "energy_class", firstNonEmpty(extracted.Building.EnergyClass, extracted.HousingCompany.EnergyClass), "building.energy_class", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "heating_method", extracted.Building.HeatingMethod, "building.heating_method", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "material", extracted.Building.Material, "building.material", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "roof_type", extracted.Building.RoofType, "building.roof_type", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeText("physical_building", *document.PhysicalBuildingID, "building", "roof_material", extracted.Building.RoofMaterial, "building.roof_material", extracted.Building.Confidence, extracted.Building.Evidence)
		claimWriter.writeBool("physical_building", *document.PhysicalBuildingID, "building", "elevator", extracted.Building.Elevator, "building.elevator", extracted.Building.Confidence, extracted.Building.Evidence)
	}
	if document.PropertyUnitID != nil {
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "unit", "apartment_number", extracted.Unit.ApartmentNumber, "unit.apartment_number", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "unit", "shares", extracted.Unit.Shares, "unit.shares", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "unit", "area_m2", extracted.Unit.AreaM2, "unit.area_m2", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "layout", "room_layout", extracted.Unit.RoomLayout, "unit.room_layout", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumberInt("property_unit", *document.PropertyUnitID, "unit", "floor_level", extracted.Unit.FloorLevel, "unit.floor_level", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "maintenance_charge_monthly", extracted.Unit.MaintenanceCharge, "unit.maintenance_charge_monthly", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "capital_charge_monthly", extracted.Unit.CapitalCharge, "unit.capital_charge_monthly", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "total_charge_monthly", extracted.Unit.TotalCharge, "unit.total_charge_monthly", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeNumber("property_unit", *document.PropertyUnitID, "charges", "debt_share_eur", extracted.Unit.DebtShare, "unit.debt_share_eur", extracted.Unit.Confidence, extracted.Unit.Evidence)
		claimWriter.writeText("property_unit", *document.PropertyUnitID, "risk", "shareholder_liability", extracted.Unit.ShareholderLiability, "unit.shareholder_liability", extracted.Unit.Confidence, extracted.Unit.Evidence)
	}
	if claimWriter.err != nil {
		return claimWriter.count, claimWriter.err
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, document.PropertyOfferingID)
	if err != nil {
		return claimWriter.count, err
	}
	if _, err := db.New(tx).MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "document_claims_changed"}); err != nil {
		return claimWriter.count, fmt.Errorf("mark dimension targets dirty from document: %w", err)
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
	w.err = w.queries.InsertDocumentPropertyClaim(w.ctx, db.InsertDocumentPropertyClaimParams{EntityType: entityType, EntityID: entityID, Section: section, Key: key, ValueKind: valueKind, ValueText: valueText, ValueNumber: valueNumber, ValueBool: valueBool, ValueJson: valueJSON, PropertyDocumentID: w.documentID, SourceField: sourceField, EvidenceText: cleanDisplayString(evidence), Confidence: float64(confidence), Model: w.model, PromptVersion: w.promptVersion})
	if w.err == nil {
		w.count++
	}
}

func replaceManagerCertificateRenovations(ctx context.Context, tx pgx.Tx, housingCompanyID uuid.UUID, offeringID uuid.UUID, items []managerCertificateRenovationObject) error {
	var saleListingID uuid.UUID
	if err := tx.QueryRow(ctx, `
SELECT pos.sale_listing_id
FROM public.property_offering_sources pos
WHERE pos.property_offering_id = $1
    AND pos.property_offering_source_link_status <> 'rejected'
ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
LIMIT 1`, offeringID).Scan(&saleListingID); err != nil {
		return mapNotFound(err)
	}
	var runID uuid.UUID
	if err := tx.QueryRow(ctx, `
INSERT INTO public.property_dimension_projection_runs (
    projection_type,
    projection_version,
    source_table,
    source_id,
    status,
    finished_at
) VALUES (
    'renovation_events',
    'manager-certificate-renovations-v1',
    'property_offerings',
    $1,
    'succeeded',
    now()
)
RETURNING property_dimension_projection_run_id`, offeringID).Scan(&runID); err != nil {
		return fmt.Errorf("create manager certificate renovation projection run: %w", err)
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM public.property_renovation_events
WHERE event_scope = 'source'
    AND target_type = 'housing_company'
    AND target_id = $1
    AND source_table = 'property_offerings'
    AND source_id = $2
    AND projection_version = 'manager-certificate-renovations-v1'`, housingCompanyID, offeringID); err != nil {
		return fmt.Errorf("delete previous manager certificate renovations: %w", err)
	}
	for _, item := range items {
		category := normalizeManagerCertificateRenovationCategory(item.Category)
		if category == "" || cleanDisplayString(item.Summary) == "" {
			continue
		}
		status := normalizeManagerCertificateRenovationStatus(item.Status)
		stage := normalizeManagerCertificateRenovationStage(item.Stage)
		scope := normalizeManagerCertificateRenovationScope(item.Scope)
		responsibility := normalizeRenovationResponsibility(item.Responsibility)
		confidence := 0.5
		switch normalizeConfidenceText(item.Confidence) {
		case "high":
			confidence = 0.9
		case "medium":
			confidence = 0.7
		case "low":
			confidence = 0.4
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO public.property_renovation_events (
    property_dimension_projection_run_id,
    projection_version,
    event_scope,
    target_type,
    target_id,
    source_table,
    source_id,
    source_field,
    category,
    component,
    status,
    stage,
    scope,
    responsibility,
    year,
    start_year,
    end_year,
    cost_estimate_eur,
    summary,
    evidence,
    confidence,
    source_reliability
) VALUES ($1, 'manager-certificate-renovations-v1', 'source', 'housing_company', $2, 'property_offerings', $3, 'manager_certificate', $4, NULL, $5, $6, $7, $8, $9, $10, $11, $12, $13, jsonb_build_object('evidence_level', 'manager_certificate'), $14, 0.9)
ON CONFLICT (
    event_scope,
    target_type,
    target_id,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    category,
    status,
    COALESCE(stage, ''),
    COALESCE(scope, ''),
    COALESCE(year, -1),
    COALESCE(start_year, -1),
    COALESCE(end_year, -1),
    md5(COALESCE(summary, '')),
    projection_version
) DO UPDATE SET
    responsibility = EXCLUDED.responsibility,
    start_year = EXCLUDED.start_year,
    end_year = EXCLUDED.end_year,
    cost_estimate_eur = EXCLUDED.cost_estimate_eur,
    confidence = EXCLUDED.confidence,
    evidence = EXCLUDED.evidence`, runID, housingCompanyID, offeringID, category, status, stage, scope, responsibility, item.Year, item.StartYear, item.EndYear, item.CostEstimateEUR, cleanDisplayString(firstNonEmpty(item.Summary, item.Evidence)), confidence); err != nil {
			return fmt.Errorf("insert manager certificate renovation: %w", err)
		}
	}
	return nil
}

func (s *Service) propertyDocumentSummary(ctx context.Context, documentID uuid.UUID) (PropertyDocumentSummary, error) {
	row, err := s.queries.GetPropertyDocumentForExtraction(ctx, documentID)
	if err != nil {
		return PropertyDocumentSummary{}, mapNotFound(err)
	}
	rows, err := s.queries.ListPropertyDocumentsForOffering(ctx, row.PropertyOfferingID)
	if err != nil {
		return PropertyDocumentSummary{}, err
	}
	for _, item := range rows {
		if item.PropertyDocumentID == documentID {
			return propertyDocumentSummaryFromListRow(item), nil
		}
	}
	return PropertyDocumentSummary{}, ErrNotFound
}

func (s *Service) enrichSaleListingDocuments(ctx context.Context, listing *SaleListing, offeringID uuid.UUID) error {
	rows, err := s.queries.ListPropertyDocumentsForOffering(ctx, offeringID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.Documents = append(listing.Documents, propertyDocumentSummaryFromListRow(row))
	}
	return nil
}

func propertyDocumentSummaryFromCreateRow(row db.CreatePropertyDocumentForOfferingRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: row.PropertyOfferingID.String(), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
}

func propertyDocumentSummaryFromListRow(row db.ListPropertyDocumentsForOfferingRow) PropertyDocumentSummary {
	return PropertyDocumentSummary{ID: row.PropertyDocumentID.String(), OfferingID: row.PropertyOfferingID.String(), UnitID: ptrUUIDString(row.PropertyUnitID), PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: ptrUUIDString(row.HousingCompanyID), Type: row.PropertyDocumentType, Filename: row.PropertyDocumentFilename, MimeType: row.PropertyDocumentMimeType, SizeBytes: row.PropertyDocumentSizeBytes, SHA256: row.PropertyDocumentSha256, ExtractionStatus: row.PropertyDocumentExtractionStatus, ExtractionError: valueOrEmpty(row.PropertyDocumentExtractionError), UploadedAt: row.PropertyDocumentUploadedAt.Format(time.RFC3339), ExtractedAt: timePtrString(row.PropertyDocumentExtractedAt), DownloadURL: propertyDocumentDownloadURL(row.PropertyDocumentID)}
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

func normalizeConfidenceText(value int32) string {
	switch {
	case value >= 80:
		return "high"
	case value >= 55:
		return "medium"
	default:
		return "low"
	}
}
