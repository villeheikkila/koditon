package properties

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/ads"
	"koditon/internal/domain/valuation"
)

var (
	ErrNotFound                                 = errors.New("property not found")
	ErrRenovationExtractorNotConfigured         = errors.New("renovation extractor not configured")
	ErrManagerCertificateExtractorNotConfigured = errors.New("manager certificate extractor not configured")
)

type Service struct {
	db                           db.DBTX
	queries                      *db.Queries
	renovationExtractorAPIKey    string
	renovationExtractorModelName string
	managerCertificateAPIKey     string
	managerCertificateModelName  string
}

type ServiceOption func(*Service)

func NewService(dbtx db.DBTX, opts ...ServiceOption) *Service {
	service := &Service{db: dbtx, queries: db.New(dbtx)}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func WithOpenRouterRenovationExtractor(apiKey string, modelName string) ServiceOption {
	return func(service *Service) {
		service.renovationExtractorAPIKey = strings.TrimSpace(apiKey)
		service.renovationExtractorModelName = strings.TrimSpace(modelName)
	}
}

func WithOpenAIManagerCertificateExtractor(apiKey string, modelName string) ServiceOption {
	return func(service *Service) {
		service.managerCertificateAPIKey = strings.TrimSpace(apiKey)
		service.managerCertificateModelName = strings.TrimSpace(modelName)
	}
}

func (s *Service) SearchSaleListings(ctx context.Context, params SearchParams) (Page[SaleListingSummary], error) {
	normalized := normalizeParams(params)
	count, err := s.countListings(ctx, normalized, "sale")
	if err != nil {
		return Page[SaleListingSummary]{}, err
	}
	rows, err := s.searchListings(ctx, normalized, "sale")
	if err != nil {
		return Page[SaleListingSummary]{}, err
	}
	out := make([]SaleListingSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toSaleSummary())
	}
	return Page[SaleListingSummary]{Rows: out, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) SearchRentals(ctx context.Context, params SearchParams) (Page[RentalSummary], error) {
	normalized := normalizeParams(params)
	count, err := s.countListings(ctx, normalized, "rental")
	if err != nil {
		return Page[RentalSummary]{}, err
	}
	rows, err := s.searchListings(ctx, normalized, "rental")
	if err != nil {
		return Page[RentalSummary]{}, err
	}
	out := make([]RentalSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toRentalSummary())
	}
	return Page[RentalSummary]{Rows: out, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) SaleListingByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (SaleListing, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return SaleListing{}, ErrNotFound
	}
	offering, sourceListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return SaleListing{}, err
	}
	listing, err := s.saleListingBySourceID(ctx, sourceListingID)
	if err != nil {
		return SaleListing{}, err
	}
	listing.ID = offeringID.String()
	listing.Canonical = offering
	records, err := s.saleOfferingSourceRecords(ctx, offeringID)
	if err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingFromOfferingSources(ctx, &listing, records, sourceListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingFromCanonicalBuilding(ctx, &listing, offeringID, sourceListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingFromSharedRow(ctx, &listing, offeringID, sourceListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingDocuments(ctx, &listing, offeringID); err != nil {
		return SaleListing{}, err
	}
	listing.SourceRecords = records
	listing.Valuation = valuation.Assess(valuationSaleListing(listing))
	return listing, nil
}

func (s *Service) saleListingBySourceID(ctx context.Context, saleListingID uuid.UUID) (SaleListing, error) {
	row, err := s.queries.GetPropertySourceOfferingDetail(ctx, &saleListingID)
	if err != nil {
		return SaleListing{}, mapNotFound(err)
	}
	var listing SaleListing
	listing.ID = row.SaleListingID.String()
	listing.Headline = valueOrEmpty(row.Headline)
	listing.Unit.Location = Location{
		StreetAddress: valueOrEmpty(row.StreetAddress),
		City:          valueOrEmpty(row.City),
		Postal:        valueOrEmpty(row.Postal),
		Latitude:      row.SaleListingLatitude,
		Longitude:     row.SaleListingLongitude,
	}
	listing.Unit.PropertyType = displayPropertyType(valueOrEmpty(row.PropertyTypeRaw))
	listing.Unit.RoomLayout = valueOrEmpty(row.RoomLayout)
	listing.Unit.RoomsCount = row.SaleListingRoomsCount
	listing.Unit.AreaM2 = row.SaleListingAreaValue
	listing.Unit.LivingAreaM2 = row.SaleListingLivingAreaValue
	listing.Unit.TotalAreaM2 = row.SaleListingTotalAreaValue
	listing.Unit.OtherAreaM2 = row.SaleListingOtherAreaValue
	listing.Unit.FloorLevel = row.SaleListingFloorLevel
	listing.Unit.Condition = displayCondition(valueOrEmpty(row.Condition))
	listing.Unit.BedroomsCount = row.SaleListingBedroomsCount
	listing.Unit.Sauna = row.SaleListingSauna
	listing.Unit.Balcony = row.SaleListingBalcony
	listing.Unit.Parking = valueOrEmpty(row.ParkingText)
	listing.Unit.KitchenDescription = valueOrEmpty(row.KitchenDescriptionText)
	listing.Unit.BathroomDescription = valueOrEmpty(row.BathroomDescriptionText)
	listing.Unit.StorageDescription = valueOrEmpty(row.StorageDescriptionText)
	listing.Unit.FloorMaterialsDescription = valueOrEmpty(row.FloorMaterialsDescriptionText)
	listing.Unit.WallMaterialsDescription = valueOrEmpty(row.WallMaterialsDescriptionText)
	listing.Unit.BalconyDescription = valueOrEmpty(row.BalconyDescriptionText)
	listing.Unit.SaunaDescription = valueOrEmpty(row.SaunaDescriptionText)
	listing.Unit.ViewsDescription = valueOrEmpty(row.ViewsDescriptionText)
	listing.Unit.Appliances = row.Appliances
	listing.Unit.Features = row.Features
	listing.Building.Location = listing.Unit.Location
	listing.Building.Elevator = row.SaleListingElevator
	listing.Building.HousingCompany = valueOrEmpty(row.HousingCompanyName)
	listing.Building.BusinessID = valueOrEmpty(row.HousingCompanyBusinessID)
	listing.Building.BuildYear = row.SaleListingBuildYear
	listing.Building.FloorCount = row.SaleListingTotalFloors
	listing.Building.ApartmentCount = row.SaleListingApartmentCount
	listing.Building.EnergyClass = displayEnergyClass(valueOrEmpty(row.EnergyEfficiencyLabel), valueOrEmpty(row.EnergyClass))
	listing.Building.BuildingMaterial = valueOrEmpty(row.BuildingMaterial)
	listing.Building.Heating = valueOrEmpty(row.HeatingSystem)
	listing.Building.RoofType = valueOrEmpty(row.RoofType)
	listing.Building.RoofMaterial = valueOrEmpty(row.RoofMaterial)
	listing.Building.CarStorage = valueOrEmpty(row.CarStorageText)
	listing.Building.OtherInfo = valueOrEmpty(row.BuildingOtherInfoText)
	listing.Site.PlotType = firstNonEmpty(valueOrEmpty(row.PlotTypeRaw), valueOrEmpty(row.PlotTypeCode))
	listing.Site.PlotAreaM2 = row.SaleListingPlotAreaValue
	listing.Site.Services = valueOrEmpty(row.ServicesText)
	listing.Site.Transport = valueOrEmpty(row.TransportText)
	if row.SaleListingPlotOwned != nil {
		if *row.SaleListingPlotOwned {
			listing.Site.PlotOwnershipType = "owned"
		} else {
			listing.Site.PlotOwnershipType = "rented"
		}
	}
	listing.Commercial.AskingPrice = row.SaleListingAskingPrice
	listing.Commercial.DebtFreePrice = row.SaleListingDebtFreePrice
	listing.Commercial.DebtShareAmount = row.SaleListingDebtShareAmount
	listing.Commercial.PricePerSquareMeter = row.SaleListingPricePerM2
	listing.Commercial.FirstSeenAt = row.SaleListingFirstSeenAt
	listing.Commercial.LastSeenAt = row.SaleListingLastSeenAt
	listing.Commercial.PublishedAt = row.SaleListingPublishedAt
	listing.Commercial.PreviousAskingPrice = row.SaleListingPreviousAskingPrice
	listing.Commercial.PreviousDebtFreePrice = row.SaleListingPreviousDebtFreePrice
	listing.Commercial.NewDevelopment = row.SaleListingNewDevelopment
	listing.Commercial.Charges = Charges{
		MaintenanceMonthly: row.SaleListingMaintenanceChargeMonthly,
		TotalMonthly:       row.SaleListingTotalChargeMonthly,
		Water:              row.SaleListingWaterCharge,
		Notes:              valueOrEmpty(row.ChargesText),
	}
	listing.Commercial.FeesInfo = valueOrEmpty(row.ChargesText)
	listing.Source = ListingSource{
		Provider:    row.SaleListingSourceProvider,
		Kind:        row.SaleListingSourceKind,
		URL:         valueOrEmpty(row.Url),
		OriginalURL: valueOrEmpty(row.Url),
		FirstSeenAt: row.SaleListingFirstSeenAt,
		LastSeenAt:  row.SaleListingLastSeenAt,
		PublishedAt: row.SaleListingPublishedAt,
	}
	listing.Texts = TextSections{
		Description:        valueOrEmpty(row.DescriptionText),
		Availability:       valueOrEmpty(row.AvailabilityText),
		RenovationsDone:    valueOrEmpty(row.RenovationsDoneText),
		RenovationsPlanned: valueOrEmpty(row.RenovationsPlannedText),
		AdditionalInfo:     valueOrEmpty(row.AdditionalInfoText),
		Charges:            valueOrEmpty(row.ChargesText),
		Building:           firstNonEmpty(valueOrEmpty(row.BuildingDescriptionText), valueOrEmpty(row.BuildingOtherInfoText)),
	}
	if row.SaleListingSourceKind == "announcement" {
		listing.Commercial.IsCompanyAnnouncement = new(true)
	}
	if err := s.enrichSaleListingMediaFromSource(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingRenovations(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingInsights(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingPropertyClaims(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingCanonicalApartmentProfile(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingCanonicalBuildingProfile(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingQualityScores(ctx, &listing, saleListingID); err != nil {
		return SaleListing{}, err
	}
	return listing, nil
}

func (s *Service) enrichSaleListingInsights(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	rows, err := s.queries.ListPropertySourceOfferingInsights(ctx, &saleListingID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.Insights.Items = append(listing.Insights.Items, Insight{Key: row.PropertySourceOfferingInsightKey, Value: valueOrEmpty(row.PropertySourceOfferingInsightValue), Direction: row.PropertySourceOfferingInsightDirection, Severity: row.PropertySourceOfferingInsightSeverity, Confidence: float64(ptrInt32Value(row.PropertySourceOfferingInsightConfidence)) / 100, Source: valueOrEmpty(row.PropertySourceOfferingInsightSourceField), Explanation: valueOrEmpty(row.PropertySourceOfferingInsightText)})
	}
	return nil
}

func (s *Service) enrichSaleListingPropertyClaims(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	targets, err := s.saleListingValuationClaimTargets(ctx, saleListingID)
	if err != nil {
		return err
	}
	for _, target := range targets {
		rows, err := s.queries.ListPropertyClaimsForEntity(ctx, db.ListPropertyClaimsForEntityParams{EntityType: &target.entityType, EntityID: &target.entityID})
		if err != nil {
			return err
		}
		for _, row := range rows {
			valueKind := valueOrEmpty(row.PropertyClaimValueKind)
			fact := valuation.ValuationFact{Section: valueOrEmpty(row.PropertyClaimNamespace), Key: valueOrEmpty(row.PropertyClaimKey), ValueKind: valueKind, ValueText: valueOrEmpty(row.PropertyClaimValueText), Confidence: float64(ptrInt32Value(row.PropertyClaimConfidence)) / 100, Source: valueOrEmpty(row.PropertyClaimSourceField), Evidence: valueOrEmpty(row.PropertyClaimEvidenceText), Model: valueOrEmpty(row.PropertyClaimModel), Prompt: valueOrEmpty(row.PropertyClaimPromptVersion)}
			if valueKind == "number" {
				fact.ValueNumber = row.PropertyClaimValueNumber
			}
			if valueKind == "bool" {
				fact.ValueBool = row.PropertyClaimValueBool
			}
			listing.ValuationInputs.Facts = append(listing.ValuationInputs.Facts, fact)
		}
	}
	return nil
}

type valuationClaimTarget struct {
	entityType string
	entityID   uuid.UUID
}

func (s *Service) saleListingValuationClaimTargets(ctx context.Context, saleListingID uuid.UUID) ([]valuationClaimTarget, error) {
	rows, err := s.queries.ListSaleListingValuationClaimTargets(ctx, &saleListingID)
	if err != nil {
		return nil, err
	}
	targets := make([]valuationClaimTarget, 0, len(rows))
	for _, row := range rows {
		targets = append(targets, valuationClaimTarget{entityType: row.EntityType, entityID: row.EntityID})
	}
	return targets, nil
}

func (s *Service) enrichSaleListingMediaFromSource(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row, err := s.queries.GetSaleListingSourceMediaData(ctx, &saleListingID)
	if err != nil {
		return err
	}
	var media Media
	provider := valueOrEmpty(row.SaleListingSourceProvider)
	kind := valueOrEmpty(row.SaleListingSourceKind)
	switch {
	case provider == "shortcut" && len(row.ShortcutAdData) > 2:
		media = shortcutMedia(parseShortcutRaw(row.ShortcutAdData))
	case provider == "frontdoor" && kind == "ad" && len(row.FrontdoorAdData) > 2:
		media = frontdoorMedia(parseFrontdoorRaw(row.FrontdoorAdData))
	case provider == "frontdoor" && kind == "announcement":
		media = frontdoorAnnouncementMedia(valueOrEmpty(row.FrontdoorBuildingAnnouncementMainImageUri))
	}
	mergeMedia(&listing.Media, media)
	return nil
}

func (s *Service) enrichSaleListingRenovations(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	rows, err := s.db.Query(ctx, `
WITH target AS (
    SELECT
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id,
        pu.physical_building_id
    FROM public.listing_search_documents doc
    JOIN public.property_offerings po ON po.property_offering_id = doc.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE doc.primary_source_listing_id = $1
        AND doc.listing_status = 'active'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
)
SELECT
    event.category,
    event.status,
    event.year,
    COALESCE(event.component, ''),
    COALESCE(event.scope, ''),
    COALESCE(event.stage, ''),
    COALESCE(event.responsibility, ''),
    event.cost_estimate_eur,
    COALESCE(event.summary, ''),
    round(COALESCE(event.confidence, 0.5) * 100)::integer,
    COALESCE(event.source_field, event.source_table),
    event.source_table,
    COALESCE(event.evidence #>> '{evidence_level}', ''),
    event.source_observed_at,
    COALESCE(event.confidence, 0.5),
    COALESCE(event.source_reliability, 0.5)
FROM target
JOIN public.property_renovation_events event
    ON event.event_scope = 'source'
    AND (
        (event.target_type = 'housing_company' AND event.target_id = target.housing_company_id)
        OR (event.target_type = 'building' AND event.target_id = target.physical_building_id)
    )`, saleListingID)
	if err != nil {
		return err
	}
	defer rows.Close()
	evidence := make([]renovationDisplayEvidence, 0)
	for rows.Next() {
		var category, status, component, scope, stage, responsibility, text, source, sourceTable, evidenceLevel string
		var year *int32
		var confidence *int32
		var costEstimateEUR *int64
		var sourceObservedAt *time.Time
		var confidenceScore, sourceReliability float64
		if err := rows.Scan(&category, &status, &year, &component, &scope, &stage, &responsibility, &costEstimateEUR, &text, &confidence, &source, &sourceTable, &evidenceLevel, &sourceObservedAt, &confidenceScore, &sourceReliability); err != nil {
			return err
		}
		evidence = append(evidence, renovationDisplayEvidence{Renovation: renovationFromEvidence(category, status, year, component, scope, stage, responsibility, costEstimateEUR, text, confidence, source), Status: status, SourceTable: sourceTable, EvidenceLevel: evidenceLevel, SourceObservedAt: sourceObservedAt, Confidence: confidenceScore, Reliability: sourceReliability})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(evidence) == 0 {
		return s.enrichSaleListingRenovationsFromFallbackRows(ctx, listing, saleListingID)
	}
	listing.Building.Renovations = compactRenovations(append(listing.Building.Renovations, resolveRenovationDisplayEvidence(evidence, time.Now())...))
	return nil
}

type renovationDisplayEvidence struct {
	Renovation       BuildingRenovation
	Status           string
	SourceTable      string
	EvidenceLevel    string
	SourceObservedAt *time.Time
	Confidence       float64
	Reliability      float64
}

func (s *Service) enrichSaleListingRenovationsFromFallbackRows(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	rows, err := s.queries.ListSaleListingFallbackRenovations(ctx, &saleListingID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.Building.Renovations = append(listing.Building.Renovations, renovationFromEvidence(row.PropertySourceOfferingRenovationCategory, row.PropertySourceOfferingRenovationStatus, row.PropertySourceOfferingRenovationYear, valueOrEmpty(row.PropertySourceOfferingRenovationComponent), valueOrEmpty(row.PropertySourceOfferingRenovationScope), valueOrEmpty(row.PropertySourceOfferingRenovationStage), valueOrEmpty(row.PropertySourceOfferingRenovationResponsibility), row.PropertySourceOfferingRenovationCostEstimateEur, valueOrEmpty(row.PropertySourceOfferingRenovationText), &row.PropertySourceOfferingRenovationConfidence, row.PropertySourceOfferingRenovationSourceField))
	}
	listing.Building.Renovations = compactRenovations(listing.Building.Renovations)
	return nil
}

func renovationFromEvidence(category string, status string, year *int32, component string, scope string, stage string, responsibility string, costEstimateEUR *int64, text string, confidence *int32, source string) BuildingRenovation {
	var done *bool
	switch status {
	case "done":
		done = new(true)
	case "planned":
		done = new(false)
	}
	renovation := buildingRenovation(category, done, year)
	renovation.Component = component
	renovation.Scope = firstNonEmpty(scope, inferRenovationScope(category+" "+component+" "+text))
	renovation.Stage = firstNonEmpty(stage, inferRenovationStage(text))
	renovation.Responsibility = firstNonEmpty(responsibility, inferRenovationResponsibility(text))
	renovation.CostEstimateEUR = costEstimateEUR
	renovation.Text = cleanDisplayString(text)
	renovation.Confidence = confidence
	renovation.Source = source
	return renovation
}

func resolveRenovationDisplayEvidence(rows []renovationDisplayEvidence, now time.Time) []BuildingRenovation {
	type selectedEvidence struct {
		row   renovationDisplayEvidence
		score float64
	}
	selected := map[string]selectedEvidence{}
	for _, row := range rows {
		key := fmt.Sprintf("%s:%s:%d:%s", row.Renovation.Kind, row.Status, ptrInt32Value(row.Renovation.Year), row.Renovation.Component)
		score := renovationEvidenceScore(row, now)
		if current, ok := selected[key]; !ok || score > current.score || score == current.score && renovationObservedAfter(row.SourceObservedAt, current.row.SourceObservedAt) {
			selected[key] = selectedEvidence{row: row, score: score}
		}
	}
	out := make([]BuildingRenovation, 0, len(selected))
	for _, item := range selected {
		out = append(out, item.row.Renovation)
	}
	sort.Slice(out, func(i int, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if ptrInt32Value(out[i].Year) != ptrInt32Value(out[j].Year) {
			if out[i].Year == nil {
				return false
			}
			if out[j].Year == nil {
				return true
			}
			return *out[i].Year < *out[j].Year
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func renovationEvidenceScore(row renovationDisplayEvidence, now time.Time) float64 {
	base := 60.0
	switch {
	case row.EvidenceLevel == "manager_certificate":
		base = 120
	case row.SourceTable == "property_source_offerings":
		base = 80
	}
	decay := 1.0
	if (row.Status == "planned" || row.Status == "suspected" || row.Status == "forecast") && row.SourceObservedAt != nil {
		ageDays := now.Sub(*row.SourceObservedAt).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
		decay = math.Pow(0.5, ageDays/365)
	}
	return base * row.Confidence * row.Reliability * decay
}

func renovationObservedAfter(left *time.Time, right *time.Time) bool {
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	return left.After(*right)
}

func (s *Service) RentalByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (Rental, error) {
	canonicalID, err := s.resolveListingInput(ctx, input, "rental", shortcutBase, frontdoorBase)
	if err != nil {
		return Rental{}, err
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return Rental{}, err
	}
	switch source + ":" + kind {
	case "shortcut:ad":
		adID, err := strconv.ParseInt(nativeID, 10, 64)
		if err != nil {
			return Rental{}, fmt.Errorf("parse shortcut ad id: %w", err)
		}
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, &adID)
		if err != nil {
			return Rental{}, mapNotFound(err)
		}
		if row.ShortcutAdType != "rental" {
			return Rental{}, fmt.Errorf("%w: not a rental", ErrNotFound)
		}
		return rentalFromShortcutAd(canonicalID, nativeID, row), nil
	case "frontdoor:announcement":
		announcementID, err := uuid.Parse(nativeID)
		if err != nil {
			return Rental{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
		}
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, &announcementID)
		if err != nil {
			return Rental{}, mapNotFound(err)
		}
		if row.FrontdoorBuildingAnnouncementRentPeriod == nil && row.FrontdoorBuildingAnnouncementRentalUniqueNo == nil {
			return Rental{}, fmt.Errorf("%w: not a rental", ErrNotFound)
		}
		return rentalFromFrontdoorAnnouncement(canonicalID, nativeID, row), nil
	default:
		return Rental{}, fmt.Errorf("%w: unsupported rental id", ErrNotFound)
	}
}

func (s *Service) BuildingByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (Building, error) {
	if buildingID, err := uuid.Parse(strings.TrimSpace(input)); err == nil {
		building, err := s.buildingByHousingCompanyID(ctx, buildingID)
		if err == nil {
			return building, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Building{}, err
		}
	}
	canonicalID, err := s.resolveBuildingInput(ctx, input, shortcutBase, frontdoorBase)
	if err != nil {
		return Building{}, err
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return Building{}, err
	}
	switch source + ":" + kind {
	case "shortcut:building":
		buildingID, err := uuid.Parse(nativeID)
		if err != nil {
			return Building{}, fmt.Errorf("parse shortcut building id: %w", err)
		}
		row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, &buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromShortcut(canonicalID, nativeID, row), nil
	case "frontdoor:building":
		buildingID, err := uuid.Parse(nativeID)
		if err != nil {
			return Building{}, fmt.Errorf("parse frontdoor building id: %w", err)
		}
		row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, &buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromFrontdoor(canonicalID, nativeID, row), nil
	default:
		return Building{}, fmt.Errorf("%w: unsupported building id", ErrNotFound)
	}
}

func (s *Service) buildingByHousingCompanyID(ctx context.Context, housingCompanyID uuid.UUID) (Building, error) {
	var building Building
	var energyRaw string
	var latitude, longitude *float64
	var mergeDecisionCount int32
	var mergedFrom []string
	err := s.db.QueryRow(ctx, `
SELECT
    pb.housing_company_id::text,
    pb.housing_company_identity_key,
    COALESCE(pb.housing_company_address_norm, ''),
    COALESCE(pb.housing_company_city_norm, ''),
    COALESCE(pb.housing_company_postal_norm, ''),
    COALESCE(pb.housing_company_name, ''),
    COALESCE(pb.housing_company_business_id, ''),
    pb.housing_company_build_year,
    pb.housing_company_floor_count,
    pb.housing_company_apartment_count,
    pb.housing_company_elevator,
    COALESCE(pb.housing_company_energy_efficiency_label, ''),
    CASE WHEN pb.housing_company_geom IS NULL THEN NULL ELSE postgis.ST_Y(pb.housing_company_geom)::double precision END,
    CASE WHEN pb.housing_company_geom IS NULL THEN NULL ELSE postgis.ST_X(pb.housing_company_geom)::double precision END,
    count(DISTINCT merges.source_housing_company_id)::int4,
    COALESCE(array_agg(DISTINCT merges.source_housing_company_id::text) FILTER (WHERE merges.source_housing_company_id IS NOT NULL), ARRAY[]::text[])
FROM public.housing_companies pb
LEFT JOIN public.housing_company_merge_decisions merges ON merges.target_housing_company_id = pb.housing_company_id
    AND merges.housing_company_merge_decision_status = 'accepted'
WHERE pb.housing_company_id = $1
GROUP BY pb.housing_company_id
LIMIT 1`, housingCompanyID).Scan(&building.ID, &building.Details.Identity.Key, &building.Details.Location.StreetAddress, &building.Details.Location.City, &building.Details.Location.Postal, &building.Details.HousingCompany, &building.Details.BusinessID, &building.Details.BuildYear, &building.Details.FloorCount, &building.Details.ApartmentCount, &building.Details.Elevator, &energyRaw, &latitude, &longitude, &mergeDecisionCount, &mergedFrom)
	if err != nil {
		return Building{}, mapNotFound(err)
	}
	building.Details.Identity.Strategy = "canonical_housing_company"
	building.Details.Identity.Confidence = 1
	building.Details.Location.Latitude = latitude
	building.Details.Location.Longitude = longitude
	building.Details.EnergyClass = displayEnergyClass(energyRaw)
	if mergeDecisionCount > 0 {
		building.Metadata = map[string]any{
			"merge_decision_count": mergeDecisionCount,
			"merged_from":          mergedFrom,
		}
	}
	if err := s.enrichBuildingFromProfile(ctx, &building, housingCompanyID); err != nil {
		return Building{}, err
	}
	if err := s.enrichBuildingFromOfferingSources(ctx, &building, housingCompanyID); err != nil {
		return Building{}, err
	}
	related, err := s.relatedListingsForBuilding(ctx, housingCompanyID)
	if err != nil {
		return Building{}, err
	}
	building.Related.Items = related
	return building, nil
}

func (s *Service) enrichBuildingFromProfile(ctx context.Context, building *Building, buildingID uuid.UUID) error {
	var housingCompany, businessID, energyLabel, buildingMaterial, heating, roofType, roofMaterial, carStorage, description, otherInfo *string
	var buildYear, floorCount, apartmentCount *int32
	var elevator *bool
	err := s.db.QueryRow(ctx, `
SELECT
    NULLIF(profile.resolved_values #>> '{housing_company,name}', ''),
    NULLIF(profile.resolved_values #>> '{housing_company,business_id}', ''),
    NULLIF(profile.resolved_values #>> '{building,build_year}', '')::int4,
    NULLIF(profile.resolved_values #>> '{building,floor_count}', '')::int4,
    NULLIF(profile.resolved_values #>> '{building,apartment_count}', '')::int4,
    NULLIF(profile.resolved_values #>> '{building,elevator}', '')::bool,
    NULLIF(profile.resolved_values #>> '{building,energy_class}', ''),
    NULLIF(profile.resolved_values #>> '{building,material}', ''),
    NULLIF(profile.resolved_values #>> '{building,heating_method}', ''),
    NULLIF(profile.resolved_values #>> '{building,roof_type}', ''),
    NULLIF(profile.resolved_values #>> '{building,roof_material}', ''),
    NULLIF(profile.resolved_values #>> '{building,car_storage}', ''),
    NULLIF(profile.resolved_values #>> '{building,description}', ''),
    NULLIF(profile.resolved_values #>> '{building,other_info}', '')
FROM public.dimension_profiles profile
WHERE profile.target_type = 'housing_company'
    AND profile.target_id = $1
LIMIT 1`, buildingID).Scan(&housingCompany, &businessID, &buildYear, &floorCount, &apartmentCount, &elevator, &energyLabel, &buildingMaterial, &heating, &roofType, &roofMaterial, &carStorage, &description, &otherInfo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	building.Details.HousingCompany = firstNonEmpty(building.Details.HousingCompany, valueOrEmpty(housingCompany))
	building.Details.BusinessID = firstNonEmpty(building.Details.BusinessID, valueOrEmpty(businessID))
	building.Details.BuildYear = firstInt32(building.Details.BuildYear, buildYear)
	building.Details.FloorCount = firstInt32(building.Details.FloorCount, floorCount)
	building.Details.ApartmentCount = firstInt32(building.Details.ApartmentCount, apartmentCount)
	building.Details.Elevator = firstBool(building.Details.Elevator, elevator)
	building.Details.EnergyClass = firstNonEmpty(building.Details.EnergyClass, displayEnergyClass(valueOrEmpty(energyLabel)))
	building.Details.BuildingMaterial = firstNonEmpty(building.Details.BuildingMaterial, valueOrEmpty(buildingMaterial))
	building.Details.Heating = firstNonEmpty(building.Details.Heating, valueOrEmpty(heating))
	building.Details.RoofType = firstNonEmpty(building.Details.RoofType, valueOrEmpty(roofType))
	building.Details.RoofMaterial = firstNonEmpty(building.Details.RoofMaterial, valueOrEmpty(roofMaterial))
	building.Details.CarStorage = firstNonEmpty(building.Details.CarStorage, valueOrEmpty(carStorage))
	building.Details.OtherInfo = firstNonEmpty(building.Details.OtherInfo, valueOrEmpty(otherInfo))
	building.Texts.Building = firstNonEmpty(building.Texts.Building, valueOrEmpty(description), valueOrEmpty(otherInfo))
	rows, err := s.queries.ListHousingCompanyRenovationEvents(ctx, &buildingID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		building.Details.Renovations = append(building.Details.Renovations, buildingRenovation(row.Category, new(true), row.Year))
	}
	building.Details.Renovations = compactRenovations(building.Details.Renovations)
	return nil
}

func (s *Service) enrichBuildingFromOfferingSources(ctx context.Context, building *Building, buildingID uuid.UUID) error {
	rows, err := s.queries.ListBuildingOfferingSourceListingIDs(ctx, &buildingID)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	var renovationCandidate *buildingRenovationSourceCandidate
	for _, saleListingID := range rows {
		if saleListingID == nil {
			continue
		}
		sourceKey := saleListingID.String()
		if _, ok := seen[sourceKey]; ok {
			continue
		}
		seen[sourceKey] = struct{}{}
		listing, err := s.saleListingBySourceID(ctx, *saleListingID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return err
		}
		mergeBuildingDetails(&building.Details, listing.Building)
		mergeSiteDetails(&building.Site, listing.Site)
		mergeTextSections(&building.Texts, listing.Texts)
		renovationCandidate = selectBuildingRenovationCandidate(renovationCandidate, listing)
		building.SourceRecords = appendUniqueListingSources(building.SourceRecords, []ListingSource{listing.Source})
	}
	if renovationCandidate != nil {
		building.Details.Renovations = renovationCandidate.renovations
		building.Texts.RenovationsDone = renovationCandidate.doneText
		building.Texts.RenovationsPlanned = renovationCandidate.plannedText
	}
	return nil
}

type buildingRenovationSourceCandidate struct {
	renovations []BuildingRenovation
	doneText    string
	plannedText string
	recency     *time.Time
	score       int
}

func selectBuildingRenovationCandidate(current *buildingRenovationSourceCandidate, listing SaleListing) *buildingRenovationSourceCandidate {
	candidate := buildingRenovationCandidateFromListing(listing)
	if candidate == nil {
		return current
	}
	if current == nil {
		return candidate
	}
	if current.recency == nil && candidate.recency != nil {
		return candidate
	}
	if current.recency != nil && candidate.recency != nil {
		if candidate.recency.After(*current.recency) {
			return candidate
		}
		if candidate.recency.Equal(*current.recency) && candidate.score > current.score {
			return candidate
		}
		return current
	}
	if candidate.score > current.score {
		return candidate
	}
	return current
}

func buildingRenovationCandidateFromListing(listing SaleListing) *buildingRenovationSourceCandidate {
	renovations := compactRenovations(listing.Building.Renovations)
	doneText := cleanDisplayString(listing.Texts.RenovationsDone)
	plannedText := cleanDisplayString(listing.Texts.RenovationsPlanned)
	if len(renovations) == 0 && doneText == "" && plannedText == "" {
		return nil
	}
	return &buildingRenovationSourceCandidate{
		renovations: renovations,
		doneText:    doneText,
		plannedText: plannedText,
		recency:     sourceRenovationRecency(listing),
		score:       len(renovations)*10 + boolScore(doneText != "") + boolScore(plannedText != ""),
	}
}

func sourceRenovationRecency(listing SaleListing) *time.Time {
	return latestTime(listing.Source.LastSeenAt, listing.Commercial.LastSeenAt, listing.Source.PublishedAt, listing.Commercial.PublishedAt, listing.Source.FirstSeenAt, listing.Commercial.FirstSeenAt)
}

func boolScore(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *Service) relatedListingsForBuilding(ctx context.Context, buildingID uuid.UUID) ([]RelatedListing, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    po.property_offering_id::text,
    po.property_offering_type,
    COALESCE(po.property_offering_headline, source_summary.headline, ''),
    COALESCE(pb.housing_company_address_norm, pu.property_unit_address_norm, ''),
    COALESCE(pu.property_unit_room_layout, ''),
    pu.property_unit_area_value,
    COALESCE(po.property_offering_asking_price, source_summary.asking_price),
    COALESCE(po.property_offering_price_per_m2, source_summary.price_per_m2),
    transaction_summary.sold_price,
    transaction_summary.sold_at,
    pb.housing_company_build_year,
    COALESCE(po.property_offering_last_seen_at, source_summary.last_seen_at),
    COALESCE(source_summary.providers, ARRAY[]::text[]),
    COALESCE(source_summary.kinds, ARRAY[]::text[])
FROM public.property_units pu
JOIN public.housing_companies pb ON pb.housing_company_id = pu.housing_company_id
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
LEFT JOIN LATERAL (
    SELECT
        max(NULLIF(doc.headline, '')) AS headline,
        min(doc.asking_price) FILTER (WHERE doc.asking_price IS NOT NULL) AS asking_price,
        min(doc.price_per_m2) FILTER (WHERE doc.price_per_m2 IS NOT NULL) AS price_per_m2,
        max(doc.last_seen_at) AS last_seen_at,
        array_agg(DISTINCT provider ORDER BY provider) AS providers,
        array_agg(DISTINCT kind ORDER BY kind) AS kinds
    FROM public.listing_search_documents doc
    LEFT JOIN LATERAL unnest(doc.source_providers) provider ON true
    LEFT JOIN LATERAL unnest(doc.source_kinds) kind ON true
    WHERE doc.property_offering_id = po.property_offering_id
        AND doc.listing_status = 'active'
) source_summary ON true
LEFT JOIN LATERAL (
    SELECT
        pt.prices_transaction_price::bigint AS sold_price,
        pt.prices_transaction_created_at AS sold_at
    FROM public.price_links pl
    JOIN origin.prices_transactions pt ON pt.prices_transaction_id = pl.prices_transaction_id
    WHERE pl.target_type = 'listing'
        AND pl.target_id = po.property_offering_id
        AND pl.link_status <> 'rejected'
    ORDER BY pl.link_score DESC, pt.prices_transaction_created_at DESC
    LIMIT 1
) transaction_summary ON true
WHERE pu.housing_company_id = $1
ORDER BY po.property_offering_last_seen_at DESC NULLS LAST, po.property_offering_asking_price ASC NULLS LAST
LIMIT 60`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RelatedListing{}
	for rows.Next() {
		var item RelatedListing
		if err := rows.Scan(&item.ID, &item.Kind, &item.FriendlyID, &item.Address, &item.RoomLayout, &item.AreaM2, &item.Price, &item.PricePerM2, &item.SoldPrice, &item.SoldAt, &item.BuildYear, &item.LastSeenAt, &item.Providers, &item.Kinds); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) resolveListingInput(ctx context.Context, input string, listingType string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	if listingType != "rental" {
		return "", ErrNotFound
	}
	publicID := strings.TrimSpace(input)
	canonicalID, err := s.queries.ResolveRentalPublicID(ctx, &publicID)
	if err != nil {
		return "", mapNotFound(err)
	}
	return canonicalID, nil
}

func (s *Service) resolveBuildingInput(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	publicID := strings.TrimSpace(input)
	canonicalID, err := s.queries.ResolveBuildingPublicID(ctx, &publicID)
	if err != nil {
		return "", mapNotFound(err)
	}
	if canonicalID == nil {
		return "", ErrNotFound
	}
	return *canonicalID, nil
}

func (s *Service) saleOfferingSource(ctx context.Context, offeringID uuid.UUID) (CanonicalOffering, uuid.UUID, error) {
	var offering CanonicalOffering
	var sourceListingID uuid.UUID
	err := s.db.QueryRow(ctx, `
SELECT
    po.property_offering_id::text,
    COALESCE(pb.housing_company_id::text, '')::text,
    pu.property_unit_id::text,
    selected.primary_source_listing_id,
    COALESCE(selected.source_count, 0)::int4,
    count(DISTINCT merges.source_property_offering_id)::int4,
    COALESCE(array_agg(DISTINCT merges.source_property_offering_id::text) FILTER (WHERE merges.source_property_offering_id IS NOT NULL), ARRAY[]::text[])
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.housing_companies pb ON pb.housing_company_id = pu.housing_company_id
JOIN LATERAL (
    SELECT
        doc.primary_source_listing_id,
        cardinality(doc.source_providers)::int4 AS source_count
    FROM public.listing_search_documents doc
    WHERE doc.property_offering_id = po.property_offering_id
        AND doc.listing_status = 'active'
    ORDER BY
        CASE WHEN doc.kind = 'ad' THEN 0 ELSE 1 END,
        CASE WHEN doc.asking_price IS NOT NULL THEN 0 ELSE 1 END,
        doc.last_seen_at DESC NULLS LAST,
        doc.refreshed_at DESC
    LIMIT 1
) selected ON true
LEFT JOIN public.property_offering_merge_decisions merges ON merges.target_property_offering_id = po.property_offering_id
    AND merges.property_offering_merge_decision_status = 'accepted'
WHERE po.property_offering_id = $1
GROUP BY po.property_offering_id, pb.housing_company_id, pu.property_unit_id, selected.primary_source_listing_id, selected.source_count
LIMIT 1`, offeringID).Scan(&offering.OfferingID, &offering.HousingCompanyID, &offering.UnitID, &sourceListingID, &offering.SourceCount, &offering.MergeDecisionCount, &offering.MergedFrom)
	if err != nil {
		return CanonicalOffering{}, uuid.UUID{}, mapNotFound(err)
	}
	return offering, sourceListingID, nil
}

func (s *Service) saleOfferingSourceRecords(ctx context.Context, offeringID uuid.UUID) ([]OfferingSourceRecord, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    evidence.evidence_source_id::text,
    CASE
        WHEN evidence.source_kind IN ('frontdoor_ad', 'frontdoor_building_announcement') THEN 'frontdoor'
        WHEN evidence.source_kind = 'shortcut_ad' THEN 'shortcut'
        ELSE evidence.source_kind
    END,
    CASE
        WHEN evidence.source_kind = 'frontdoor_building_announcement' THEN 'announcement'
        WHEN evidence.source_kind = 'frontdoor_ad' THEN 'ad'
        WHEN evidence.source_kind = 'shortcut_ad' THEN 'ad'
        ELSE evidence.source_kind
    END,
    COALESCE(evidence.external_id, ''),
    COALESCE(evidence.url, doc.url, ''),
    COALESCE(doc.headline, ''),
    doc.first_seen_at,
    COALESCE(evidence.observed_at, doc.last_seen_at),
    entity_evidence.link_status,
    entity_evidence.link_method,
    (entity_evidence.confidence * 100)::int4
FROM public.listing_search_documents doc
JOIN public.entity_evidence entity_evidence ON entity_evidence.listing_id = doc.listing_id
    AND entity_evidence.link_status <> 'rejected'
JOIN public.evidence_sources evidence ON evidence.evidence_source_id = entity_evidence.evidence_source_id
WHERE doc.property_offering_id = $1
    AND doc.listing_status = 'active'
ORDER BY COALESCE(evidence.observed_at, doc.last_seen_at) DESC NULLS LAST, evidence.provider, evidence.external_id`, offeringID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OfferingSourceRecord{}
	for rows.Next() {
		var record OfferingSourceRecord
		if err := rows.Scan(&record.ID, &record.Provider, &record.Kind, &record.NativeID, &record.URL, &record.Headline, &record.FirstSeenAt, &record.LastSeenAt, &record.LinkStatus, &record.LinkMethod, &record.LinkScore); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) enrichSaleListingFromOfferingSources(ctx context.Context, listing *SaleListing, records []OfferingSourceRecord, primarySourceListingID uuid.UUID) error {
	for _, record := range records {
		if record.ID == "" || record.ID == primarySourceListingID.String() {
			continue
		}
		sourceListingID, err := uuid.Parse(record.ID)
		if err != nil {
			continue
		}
		sourceListing, err := s.saleListingBySourceID(ctx, sourceListingID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return err
		}
		mergeSaleListingDetails(listing, sourceListing)
	}
	return nil
}

func mergeSaleListingDetails(dst *SaleListing, src SaleListing) {
	dst.Headline = firstNonEmpty(dst.Headline, src.Headline)
	unitCompatible := compatibleUnitFacts(dst, src)
	if unitCompatible {
		mergeUnitDetails(&dst.Unit, src.Unit)
		mergeCommercialDetails(&dst.Commercial, src.Commercial)
		mergeTextSections(&dst.Texts, src.Texts)
		mergeMedia(&dst.Media, src.Media)
		dst.Contacts = appendUniqueContacts(dst.Contacts, src.Contacts)
		dst.Showings = append(dst.Showings, src.Showings...)
		dst.Links = appendUniqueLinks(dst.Links, src.Links)
	}
	mergeBuildingDetails(&dst.Building, src.Building)
	mergeSiteDetails(&dst.Site, src.Site)
}

func compatibleUnitFacts(dst *SaleListing, src SaleListing) bool {
	if !sameFloat(dst.Unit.AreaM2, src.Unit.AreaM2) {
		return false
	}
	if !sameInt32(dst.Unit.FloorLevel, src.Unit.FloorLevel) {
		return false
	}
	if !sameInt32(dst.Unit.RoomsCount, src.Unit.RoomsCount) {
		return false
	}
	if !sameInt32(dst.Building.BuildYear, src.Building.BuildYear) {
		return false
	}
	return true
}

func mergeUnitDetails(dst *UnitDetails, src UnitDetails) {
	mergeLocation(&dst.Location, src.Location)
	dst.PropertyType = firstNonEmpty(dst.PropertyType, src.PropertyType)
	dst.PropertySubtype = firstNonEmpty(dst.PropertySubtype, src.PropertySubtype)
	dst.RoomLayout = firstNonEmpty(dst.RoomLayout, src.RoomLayout)
	dst.RoomsCount = firstInt32(dst.RoomsCount, src.RoomsCount)
	dst.BedroomsCount = firstInt32(dst.BedroomsCount, src.BedroomsCount)
	dst.AreaM2 = firstFloat64(dst.AreaM2, src.AreaM2)
	dst.LivingAreaM2 = firstFloat64(dst.LivingAreaM2, src.LivingAreaM2)
	dst.TotalAreaM2 = firstFloat64(dst.TotalAreaM2, src.TotalAreaM2)
	dst.OtherAreaM2 = firstFloat64(dst.OtherAreaM2, src.OtherAreaM2)
	dst.FloorLevel = firstInt32(dst.FloorLevel, src.FloorLevel)
	dst.Condition = firstNonEmpty(dst.Condition, displayCondition(src.Condition))
	dst.Sauna = trueWinsBool(dst.Sauna, src.Sauna)
	dst.Balcony = trueWinsBool(dst.Balcony, src.Balcony)
	dst.Parking = richerText(dst.Parking, src.Parking)
	dst.Availability = firstNonEmpty(dst.Availability, src.Availability)
	dst.KitchenDescription = richerText(dst.KitchenDescription, src.KitchenDescription)
	dst.BathroomDescription = richerText(dst.BathroomDescription, src.BathroomDescription)
	dst.StorageDescription = richerText(dst.StorageDescription, src.StorageDescription)
	dst.FloorMaterialsDescription = richerText(dst.FloorMaterialsDescription, src.FloorMaterialsDescription)
	dst.WallMaterialsDescription = richerText(dst.WallMaterialsDescription, src.WallMaterialsDescription)
	dst.BalconyDescription = richerText(dst.BalconyDescription, src.BalconyDescription)
	dst.SaunaDescription = richerText(dst.SaunaDescription, src.SaunaDescription)
	dst.ViewsDescription = richerText(dst.ViewsDescription, src.ViewsDescription)
	dst.Appliances = compactStrings(append(dst.Appliances, src.Appliances...))
	dst.Features = compactStrings(append(dst.Features, src.Features...))
}

func mergeBuildingDetails(dst *BuildingDetails, src BuildingDetails) {
	mergeLocation(&dst.Location, src.Location)
	dst.HousingCompany = firstNonEmpty(dst.HousingCompany, src.HousingCompany)
	dst.BusinessID = firstNonEmpty(dst.BusinessID, src.BusinessID)
	dst.BuildingType = firstNonEmpty(dst.BuildingType, src.BuildingType)
	dst.BuildingSubtype = firstNonEmpty(dst.BuildingSubtype, src.BuildingSubtype)
	dst.BuildYear = firstInt32(dst.BuildYear, src.BuildYear)
	dst.ConstructionYear = firstInt32(dst.ConstructionYear, src.ConstructionYear)
	dst.FloorCount = firstInt32(dst.FloorCount, src.FloorCount)
	dst.ApartmentCount = firstInt32(dst.ApartmentCount, src.ApartmentCount)
	dst.BusinessPremiseCount = firstInt32(dst.BusinessPremiseCount, src.BusinessPremiseCount)
	dst.EnergyClass = firstNonEmpty(dst.EnergyClass, displayEnergyClass(src.EnergyClass))
	dst.Heating = firstNonEmpty(dst.Heating, src.Heating)
	dst.HeatingDescription = firstNonEmpty(dst.HeatingDescription, src.HeatingDescription)
	dst.HeatingFuel = firstNonEmpty(dst.HeatingFuel, src.HeatingFuel)
	dst.BuildingMaterial = firstNonEmpty(dst.BuildingMaterial, src.BuildingMaterial)
	dst.WallStructure = firstNonEmpty(dst.WallStructure, src.WallStructure)
	dst.FrameConstructionMethod = firstNonEmpty(dst.FrameConstructionMethod, src.FrameConstructionMethod)
	dst.RoofType = firstNonEmpty(dst.RoofType, src.RoofType)
	dst.RoofMaterial = firstNonEmpty(dst.RoofMaterial, src.RoofMaterial)
	dst.CommonAreas = firstNonEmpty(dst.CommonAreas, src.CommonAreas)
	dst.CarStorage = firstNonEmpty(dst.CarStorage, src.CarStorage)
	dst.Connectivity = firstNonEmpty(dst.Connectivity, src.Connectivity)
	dst.OtherInfo = firstNonEmpty(dst.OtherInfo, src.OtherInfo)
	dst.Elevator = firstBool(dst.Elevator, src.Elevator)
	dst.Sauna = trueWinsBool(dst.Sauna, src.Sauna)
	dst.Renovations = compactRenovations(append(dst.Renovations, src.Renovations...))
	dst.ManagementMethod = firstNonEmpty(dst.ManagementMethod, src.ManagementMethod)
	dst.PropertyManager = firstNonEmpty(dst.PropertyManager, src.PropertyManager)
	dst.MaintenanceResponsibility = firstNonEmpty(dst.MaintenanceResponsibility, src.MaintenanceResponsibility)
}

func mergeSiteDetails(dst *SiteDetails, src SiteDetails) {
	dst.PlotType = firstNonEmpty(dst.PlotType, src.PlotType)
	dst.PlotOwnershipType = firstNonEmpty(dst.PlotOwnershipType, src.PlotOwnershipType)
	dst.PlotAreaM2 = firstFloat64(dst.PlotAreaM2, src.PlotAreaM2)
	dst.LotRedemptionInfo = firstNonEmpty(dst.LotRedemptionInfo, src.LotRedemptionInfo)
	dst.LotRentalAgreement = firstNonEmpty(dst.LotRentalAgreement, src.LotRentalAgreement)
	dst.Yard = firstNonEmpty(dst.Yard, src.Yard)
	dst.Shore = firstNonEmpty(dst.Shore, src.Shore)
	dst.WaterSupply = firstNonEmpty(dst.WaterSupply, src.WaterSupply)
	dst.Sewer = firstNonEmpty(dst.Sewer, src.Sewer)
	dst.RoadAccess = firstNonEmpty(dst.RoadAccess, src.RoadAccess)
	dst.Zoning = firstNonEmpty(dst.Zoning, src.Zoning)
	dst.DrivingDirections = firstNonEmpty(dst.DrivingDirections, src.DrivingDirections)
	dst.Services = richerText(dst.Services, src.Services)
	dst.Transport = richerText(dst.Transport, src.Transport)
	dst.WaterSupplyTypes = compactStrings(append(dst.WaterSupplyTypes, src.WaterSupplyTypes...))
}

func mergeCommercialDetails(dst *CommercialDetails, src CommercialDetails) {
	matchedTransaction := dst.MatchedTransaction
	dst.Status = firstNonEmpty(dst.Status, src.Status)
	dst.BookingStatus = firstNonEmpty(dst.BookingStatus, src.BookingStatus)
	dst.PublishedAt = earliestTime(dst.PublishedAt, src.PublishedAt)
	dst.UnpublishedAt = latestTime(dst.UnpublishedAt, src.UnpublishedAt)
	dst.FirstSeenAt = earliestTime(dst.FirstSeenAt, src.FirstSeenAt)
	dst.LastSeenAt = latestTime(dst.LastSeenAt, src.LastSeenAt)
	dst.DaysOnMarket = firstInt32(dst.DaysOnMarket, src.DaysOnMarket)
	dst.MapVisible = firstBool(dst.MapVisible, src.MapVisible)
	dst.CanReceiveLeads = firstBool(dst.CanReceiveLeads, src.CanReceiveLeads)
	dst.AskingPrice = firstInt64(dst.AskingPrice, src.AskingPrice)
	dst.DebtFreePrice = firstInt64(dst.DebtFreePrice, src.DebtFreePrice)
	dst.DebtShareAmount = firstInt64(dst.DebtShareAmount, src.DebtShareAmount)
	dst.PreviousAskingPrice = firstInt64(dst.PreviousAskingPrice, src.PreviousAskingPrice)
	dst.PreviousDebtFreePrice = firstInt64(dst.PreviousDebtFreePrice, src.PreviousDebtFreePrice)
	dst.PricePerSquareMeter = firstFloat64(dst.PricePerSquareMeter, src.PricePerSquareMeter)
	dst.OwnershipType = firstNonEmpty(dst.OwnershipType, src.OwnershipType)
	dst.DebtShareAdditionalInfo = firstNonEmpty(dst.DebtShareAdditionalInfo, src.DebtShareAdditionalInfo)
	dst.FeesInfo = firstNonEmpty(dst.FeesInfo, src.FeesInfo)
	dst.FinancingFeeInterestOnlyPeriod = firstNonEmpty(dst.FinancingFeeInterestOnlyPeriod, src.FinancingFeeInterestOnlyPeriod)
	dst.FinancingFeeInterestOnlyStartDate = firstNonEmpty(dst.FinancingFeeInterestOnlyStartDate, src.FinancingFeeInterestOnlyStartDate)
	dst.FinancingFeeInterestOnlyEndDate = firstNonEmpty(dst.FinancingFeeInterestOnlyEndDate, src.FinancingFeeInterestOnlyEndDate)
	dst.OpenBiddingInUse = firstBool(dst.OpenBiddingInUse, src.OpenBiddingInUse)
	dst.OpenBiddingStartingSellingPrice = firstInt64(dst.OpenBiddingStartingSellingPrice, src.OpenBiddingStartingSellingPrice)
	dst.OpenBiddingStartingDebtFreePrice = firstInt64(dst.OpenBiddingStartingDebtFreePrice, src.OpenBiddingStartingDebtFreePrice)
	dst.OpenBiddingLatestOffer = firstInt64(dst.OpenBiddingLatestOffer, src.OpenBiddingLatestOffer)
	dst.OpenBiddingTargetURL = firstNonEmpty(dst.OpenBiddingTargetURL, src.OpenBiddingTargetURL)
	dst.DevelopmentPhase = firstNonEmpty(dst.DevelopmentPhase, src.DevelopmentPhase)
	dst.NewDevelopment = trueWinsBool(dst.NewDevelopment, src.NewDevelopment)
	dst.NotifyPriceChanged = firstBool(dst.NotifyPriceChanged, src.NotifyPriceChanged)
	dst.MainImageHidden = firstBool(dst.MainImageHidden, src.MainImageHidden)
	dst.IsCompanyAnnouncement = firstBool(dst.IsCompanyAnnouncement, src.IsCompanyAnnouncement)
	dst.ShowBiddingIndicators = firstBool(dst.ShowBiddingIndicators, src.ShowBiddingIndicators)
	mergeCharges(&dst.Charges, src.Charges)
	dst.MatchedTransaction = matchedTransaction
}

func mergeCharges(dst *Charges, src Charges) {
	dst.MaintenanceMonthly = firstFloat64(dst.MaintenanceMonthly, src.MaintenanceMonthly)
	dst.TotalMonthly = firstFloat64(dst.TotalMonthly, src.TotalMonthly)
	dst.Water = firstFloat64(dst.Water, src.Water)
	dst.Parking = firstFloat64(dst.Parking, src.Parking)
	dst.Sauna = firstFloat64(dst.Sauna, src.Sauna)
	dst.Electricity = firstNonEmpty(dst.Electricity, src.Electricity)
	dst.Heating = firstNonEmpty(dst.Heating, src.Heating)
	dst.Notes = richerText(dst.Notes, src.Notes)
}

func mergeTextSections(dst *TextSections, src TextSections) {
	dst.Description = richerText(dst.Description, src.Description)
	dst.Availability = firstNonEmpty(dst.Availability, src.Availability)
	dst.RenovationsDone = combineText(dst.RenovationsDone, src.RenovationsDone)
	dst.RenovationsPlanned = combineText(dst.RenovationsPlanned, src.RenovationsPlanned)
	dst.AdditionalInfo = richerText(dst.AdditionalInfo, src.AdditionalInfo)
	dst.Area = richerText(dst.Area, src.Area)
	dst.Building = richerText(dst.Building, src.Building)
	dst.Transport = richerText(dst.Transport, src.Transport)
	dst.Amenities = richerText(dst.Amenities, src.Amenities)
	dst.Charges = richerText(dst.Charges, src.Charges)
	dst.Kitchen = richerText(dst.Kitchen, src.Kitchen)
	dst.Bathroom = richerText(dst.Bathroom, src.Bathroom)
	dst.Storage = richerText(dst.Storage, src.Storage)
	dst.Materials = richerText(dst.Materials, src.Materials)
}

func mergeMedia(dst *Media, src Media) {
	if dst.MainImage == nil {
		dst.MainImage = src.MainImage
	}
	dst.Images = appendUniqueImages(dst.Images, src.Images)
}

func mergeLocation(dst *Location, src Location) {
	dst.StreetAddress = firstNonEmpty(dst.StreetAddress, src.StreetAddress)
	dst.City = firstNonEmpty(dst.City, src.City)
	dst.Postal = firstNonEmpty(dst.Postal, src.Postal)
	dst.District = firstNonEmpty(dst.District, src.District)
	dst.Latitude = firstFloat64(dst.Latitude, src.Latitude)
	dst.Longitude = firstFloat64(dst.Longitude, src.Longitude)
}

func earliestTime(values ...*time.Time) *time.Time {
	var out *time.Time
	for _, value := range values {
		if value != nil && (out == nil || value.Before(*out)) {
			out = value
		}
	}
	return out
}

func latestTime(values ...*time.Time) *time.Time {
	var out *time.Time
	for _, value := range values {
		if value != nil && (out == nil || value.After(*out)) {
			out = value
		}
	}
	return out
}

func trueWinsBool(dst *bool, src *bool) *bool {
	if dst != nil && *dst {
		return dst
	}
	if src != nil && *src {
		return src
	}
	return firstBool(dst, src)
}

func richerText(dst string, src string) string {
	dst = cleanDisplayString(dst)
	src = cleanDisplayString(src)
	if dst == "" {
		return src
	}
	if src == "" {
		return dst
	}
	if strings.Contains(strings.ToLower(dst), strings.ToLower(src)) {
		return dst
	}
	if strings.Contains(strings.ToLower(src), strings.ToLower(dst)) {
		return src
	}
	if len([]rune(src)) > len([]rune(dst))*3/2 {
		return src
	}
	return dst
}

func combineText(dst string, src string) string {
	dst = cleanDisplayString(dst)
	src = cleanDisplayString(src)
	if dst == "" {
		return src
	}
	if src == "" || strings.Contains(strings.ToLower(dst), strings.ToLower(src)) {
		return dst
	}
	if strings.Contains(strings.ToLower(src), strings.ToLower(dst)) {
		return src
	}
	return dst + "\n\n" + src
}

func sameInt32(a *int32, b *int32) bool {
	if a == nil || b == nil {
		return true
	}
	return *a == *b
}

func sameFloat(a *float64, b *float64) bool {
	if a == nil || b == nil {
		return true
	}
	delta := *a - *b
	if delta < 0 {
		delta = -delta
	}
	return delta < 0.01
}

func appendUniqueImages(dst []Image, src []Image) []Image {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := make([]Image, 0, len(dst)+len(src))
	for _, image := range append(dst, src...) {
		key := image.URL
		if key == "" {
			key = image.ID
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, image)
	}
	return out
}

func appendUniqueContacts(dst []Contact, src []Contact) []Contact {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := make([]Contact, 0, len(dst)+len(src))
	for _, contact := range append(dst, src...) {
		key := strings.ToLower(contact.Name + "|" + contact.Email + "|" + contact.Phone)
		if key == "||" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, contact)
	}
	return out
}

func appendUniqueLinks(dst []Link, src []Link) []Link {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := make([]Link, 0, len(dst)+len(src))
	for _, link := range append(dst, src...) {
		if link.URL == "" {
			continue
		}
		if _, ok := seen[link.URL]; ok {
			continue
		}
		seen[link.URL] = struct{}{}
		out = append(out, link)
	}
	return out
}

func appendUniqueListingSources(dst []ListingSource, src []ListingSource) []ListingSource {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := make([]ListingSource, 0, len(dst)+len(src))
	for _, source := range append(dst, src...) {
		key := source.Provider + "|" + source.Kind + "|" + source.CanonicalID + "|" + source.NativeID
		if key == "|||" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, source)
	}
	return out
}

func (s *Service) SaleOfferingSourceRawPayload(ctx context.Context, offeringIDInput, sourceIDInput string) (OfferingSourceRawPayload, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(offeringIDInput))
	if err != nil {
		return OfferingSourceRawPayload{}, ErrNotFound
	}
	sourceID, err := uuid.Parse(strings.TrimSpace(sourceIDInput))
	if err != nil {
		return OfferingSourceRawPayload{}, ErrNotFound
	}
	var out OfferingSourceRawPayload
	err = s.db.QueryRow(ctx, `
SELECT
    evidence.evidence_source_id::text,
    CASE
        WHEN evidence.source_kind IN ('frontdoor_ad', 'frontdoor_building_announcement') THEN 'frontdoor'
        WHEN evidence.source_kind = 'shortcut_ad' THEN 'shortcut'
        ELSE evidence.source_kind
    END,
    CASE
        WHEN evidence.source_kind = 'frontdoor_building_announcement' THEN 'announcement'
        WHEN evidence.source_kind = 'frontdoor_ad' THEN 'ad'
        WHEN evidence.source_kind = 'shortcut_ad' THEN 'ad'
        ELSE evidence.source_kind
    END,
    COALESCE(evidence.external_id, doc.native_id),
    COALESCE(
        CASE
            WHEN evidence.shortcut_ad_id IS NOT NULL THEN sa.shortcut_ad_data
            WHEN evidence.frontdoor_ad_id IS NOT NULL THEN fa.frontdoor_ad_data
            WHEN evidence.frontdoor_building_announcement_id IS NOT NULL THEN to_jsonb(fba)
            ELSE NULL
        END,
        '{}'::jsonb
    ) AS payload
FROM public.listing_search_documents doc
JOIN public.entity_evidence entity_evidence ON entity_evidence.listing_id = doc.listing_id
    AND entity_evidence.evidence_source_id = $2
    AND entity_evidence.link_status <> 'rejected'
JOIN public.evidence_sources evidence ON evidence.evidence_source_id = entity_evidence.evidence_source_id
LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = evidence.shortcut_ad_id
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = evidence.frontdoor_ad_id
LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = evidence.frontdoor_building_announcement_id
WHERE doc.property_offering_id = $1
    AND doc.listing_status = 'active'
LIMIT 1`, offeringID, sourceID).Scan(&out.ID, &out.Provider, &out.Kind, &out.NativeID, &out.Payload)
	if err != nil {
		return OfferingSourceRawPayload{}, mapNotFound(err)
	}
	return out, nil
}

func (s *Service) enrichSaleListingFromCanonicalBuilding(ctx context.Context, listing *SaleListing, offeringID uuid.UUID, saleListingID uuid.UUID) error {
	var housingCompany, businessID, address, postal, city, energyLabel, buildingMaterial, heating, roofMaterial, roofType, carStorage, description, otherInfo *string
	var buildYear, floorCount, apartmentCount *int32
	var elevator *bool
	var latitude, longitude *float64
	err := s.db.QueryRow(ctx, `
SELECT
    COALESCE(NULLIF(hcp.resolved_values #>> '{housing_company,name}', ''), hc.housing_company_name),
    COALESCE(NULLIF(hcp.resolved_values #>> '{housing_company,business_id}', ''), hc.housing_company_business_id),
    hc.housing_company_address_norm,
    hc.housing_company_postal_norm,
    hc.housing_company_city_norm,
    COALESCE(NULLIF(hcp.resolved_values #>> '{building,build_year}', '')::int4, hc.housing_company_build_year),
    COALESCE(NULLIF(hcp.resolved_values #>> '{building,floor_count}', '')::int4, hc.housing_company_floor_count),
    COALESCE(NULLIF(hcp.resolved_values #>> '{building,apartment_count}', '')::int4, hc.housing_company_apartment_count),
    COALESCE(NULLIF(hcp.resolved_values #>> '{building,elevator}', '')::bool, hc.housing_company_elevator),
    COALESCE(NULLIF(hcp.resolved_values #>> '{building,energy_class}', ''), hc.housing_company_energy_efficiency_label),
    NULLIF(hcp.resolved_values #>> '{building,material}', ''),
    NULLIF(hcp.resolved_values #>> '{building,heating_method}', ''),
    NULLIF(hcp.resolved_values #>> '{building,roof_material}', ''),
    NULLIF(hcp.resolved_values #>> '{building,roof_type}', ''),
    NULLIF(hcp.resolved_values #>> '{building,car_storage}', ''),
    NULLIF(hcp.resolved_values #>> '{building,description}', ''),
    NULLIF(hcp.resolved_values #>> '{building,other_info}', ''),
    CASE WHEN hc.housing_company_geom IS NULL THEN NULL ELSE postgis.ST_Y(hc.housing_company_geom)::double precision END,
    CASE WHEN hc.housing_company_geom IS NULL THEN NULL ELSE postgis.ST_X(hc.housing_company_geom)::double precision END
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
LEFT JOIN public.dimension_profiles hcp ON hcp.target_type = 'housing_company'
    AND hcp.target_id = hc.housing_company_id
WHERE po.property_offering_id = $1
LIMIT 1`, offeringID).Scan(&housingCompany, &businessID, &address, &postal, &city, &buildYear, &floorCount, &apartmentCount, &elevator, &energyLabel, &buildingMaterial, &heating, &roofMaterial, &roofType, &carStorage, &description, &otherInfo, &latitude, &longitude)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	listing.Building.HousingCompany = firstNonEmpty(listing.Building.HousingCompany, valueOrEmpty(housingCompany))
	listing.Building.BusinessID = firstNonEmpty(listing.Building.BusinessID, valueOrEmpty(businessID))
	listing.Building.Location.StreetAddress = firstNonEmpty(listing.Building.Location.StreetAddress, valueOrEmpty(address))
	listing.Building.Location.Postal = firstNonEmpty(listing.Building.Location.Postal, valueOrEmpty(postal))
	listing.Building.Location.City = firstNonEmpty(listing.Building.Location.City, valueOrEmpty(city))
	listing.Building.Location.Latitude = firstFloat64(listing.Building.Location.Latitude, latitude)
	listing.Building.Location.Longitude = firstFloat64(listing.Building.Location.Longitude, longitude)
	listing.Unit.Location.Latitude = firstFloat64(listing.Unit.Location.Latitude, latitude)
	listing.Unit.Location.Longitude = firstFloat64(listing.Unit.Location.Longitude, longitude)
	listing.Building.BuildYear = firstInt32(listing.Building.BuildYear, buildYear)
	listing.Building.FloorCount = firstInt32(listing.Building.FloorCount, floorCount)
	listing.Building.ApartmentCount = firstInt32(listing.Building.ApartmentCount, apartmentCount)
	listing.Building.Elevator = firstBool(listing.Building.Elevator, elevator)
	energy := displayEnergyClass(valueOrEmpty(energyLabel))
	listing.Building.EnergyClass = firstNonEmpty(listing.Building.EnergyClass, energy)
	listing.Building.BuildingMaterial = firstNonEmpty(listing.Building.BuildingMaterial, valueOrEmpty(buildingMaterial))
	listing.Building.Heating = firstNonEmpty(listing.Building.Heating, valueOrEmpty(heating))
	listing.Building.RoofMaterial = firstNonEmpty(listing.Building.RoofMaterial, valueOrEmpty(roofMaterial))
	listing.Building.RoofType = firstNonEmpty(listing.Building.RoofType, valueOrEmpty(roofType))
	listing.Building.CarStorage = firstNonEmpty(listing.Building.CarStorage, valueOrEmpty(carStorage))
	listing.Building.OtherInfo = firstNonEmpty(listing.Building.OtherInfo, valueOrEmpty(otherInfo))
	listing.Texts.Building = firstNonEmpty(listing.Texts.Building, valueOrEmpty(description), valueOrEmpty(otherInfo))
	if err := s.enrichSaleListingFromHousingCompanyRenovations(ctx, listing, offeringID); err != nil {
		return err
	}
	return nil
}

func (s *Service) enrichSaleListingFromHousingCompanyRenovations(ctx context.Context, listing *SaleListing, offeringID uuid.UUID) error {
	rows, err := s.db.Query(ctx, `
SELECT
    event.category,
    COALESCE(event.year, event.start_year)::int4
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.property_renovation_events event ON event.target_type = 'housing_company'
    AND event.target_id = pu.housing_company_id
WHERE po.property_offering_id = $1
    AND event.category <> ''
    AND event.status = 'done'
ORDER BY event.category, COALESCE(event.year, event.start_year) NULLS LAST`, offeringID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var category string
		var year *int32
		if err := rows.Scan(&category, &year); err != nil {
			return err
		}
		listing.Building.Renovations = append(listing.Building.Renovations, buildingRenovation(category, new(true), year))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	listing.Building.Renovations = compactRenovations(listing.Building.Renovations)
	return nil
}

func (s *Service) enrichSaleListingFromSharedRow(ctx context.Context, listing *SaleListing, offeringID uuid.UUID, saleListingID uuid.UUID) error {
	var transactionID *uuid.UUID
	var transactionFirstSeenAt, transactionUpdatedAt *time.Time
	var description, transactionType, category, period string
	var area float64
	var price, pricePerM2, buildYear int32
	var floor, condition, plot, energyClass *string
	var city, neighborhood, postalCode *string
	var elevator bool
	var plotOwned *bool
	var matchStatus *string
	var matchScore *int32
	var matchConfidence *string
	err := s.db.QueryRow(ctx, `
SELECT
    pt.prices_transaction_id,
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    COALESCE(pt.prices_transaction_area, 0),
    COALESCE(pt.prices_transaction_price, 0),
    COALESCE(pt.prices_transaction_price_per_square_meter, 0),
    COALESCE(pt.prices_transaction_build_year, 0),
    pt.prices_transaction_floor,
    COALESCE(pt.prices_transaction_elevator, false),
    pt.prices_transaction_condition,
    pt.prices_transaction_plot,
    pt.prices_transaction_plot_owned,
    pt.prices_transaction_energy_class,
    COALESCE(pt.prices_transaction_period_identifier, ''),
    pc.prices_city_name,
    pn.prices_neighborhood_name,
    ppc.prices_postal_code_code,
    pl.link_status,
    COALESCE(c.sale_listing_prices_transaction_match_score, pl.link_score),
    c.sale_listing_prices_transaction_match_confidence
FROM public.price_links pl
JOIN origin.prices_transactions pt ON pt.prices_transaction_id = pl.prices_transaction_id
LEFT JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN origin.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN LATERAL (
    SELECT
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.prices_transaction_id = pl.prices_transaction_id
        AND c.sale_listing_id = $2
    ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
    LIMIT 1
) c ON true
WHERE pl.target_type = 'listing'
    AND pl.target_id = $1
    AND pl.link_status <> 'rejected'
ORDER BY pl.link_score DESC, pl.updated_at DESC
LIMIT 1`, offeringID, saleListingID).Scan(&transactionID, &transactionFirstSeenAt, &transactionUpdatedAt, &description, &transactionType, &category, &area, &price, &pricePerM2, &buildYear, &floor, &elevator, &condition, &plot, &plotOwned, &energyClass, &period, &city, &neighborhood, &postalCode, &matchStatus, &matchScore, &matchConfidence)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return s.enrichSaleListingFromSourceTransaction(ctx, listing, saleListingID)
	}
	if transactionID == nil {
		return nil
	}
	areaPtr := area
	priceValue := int64(price)
	pricePerM2Value := int64(pricePerM2)
	buildYearValue := buildYear
	listing.Commercial.MatchedTransaction = &PriceTransactionMatch{ID: transactionID.String(), FirstSeenAt: transactionFirstSeenAt, UpdatedAt: transactionUpdatedAt, Description: description, Type: transactionType, Category: category, AreaM2: &areaPtr, Price: &priceValue, PricePerSquareMeter: &pricePerM2Value, BuildYear: &buildYearValue, Floor: valueOrEmpty(floor), Elevator: &elevator, Condition: displayCondition(valueOrEmpty(condition)), Plot: valueOrEmpty(plot), PlotOwned: plotOwned, EnergyClass: displayEnergyClass(valueOrEmpty(energyClass)), PeriodIdentifier: period, City: valueOrEmpty(city), Neighborhood: valueOrEmpty(neighborhood), PostalCode: valueOrEmpty(postalCode), MatchStatus: valueOrEmpty(matchStatus), MatchScore: matchScore, MatchConfidence: valueOrEmpty(matchConfidence)}
	return nil
}

func (s *Service) enrichSaleListingFromSourceTransaction(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	var transactionID *uuid.UUID
	var transactionFirstSeenAt, transactionUpdatedAt *time.Time
	var description, transactionType, category, period string
	var area float64
	var price, pricePerM2, buildYear int32
	var floor, condition, plot, energyClass *string
	var city, neighborhood, postalCode *string
	var elevator bool
	var plotOwned *bool
	var matchStatus *string
	var matchScore *int32
	var matchConfidence *string
	err := s.db.QueryRow(ctx, `
SELECT
    pt.prices_transaction_id,
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    COALESCE(pt.prices_transaction_area, 0),
    COALESCE(pt.prices_transaction_price, 0),
    COALESCE(pt.prices_transaction_price_per_square_meter, 0),
    COALESCE(pt.prices_transaction_build_year, 0),
    pt.prices_transaction_floor,
    COALESCE(pt.prices_transaction_elevator, false),
    pt.prices_transaction_condition,
    pt.prices_transaction_plot,
    pt.prices_transaction_plot_owned,
    pt.prices_transaction_energy_class,
    COALESCE(pt.prices_transaction_period_identifier, ''),
    pc.prices_city_name,
    pn.prices_neighborhood_name,
    ppc.prices_postal_code_code,
    pl.link_status,
    c.sale_listing_prices_transaction_match_score,
    c.sale_listing_prices_transaction_match_confidence
FROM public.price_links pl
JOIN origin.prices_transactions pt ON pt.prices_transaction_id = pl.prices_transaction_id
LEFT JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN origin.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN LATERAL (
    SELECT
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.sale_listing_id = pl.target_id
        AND c.prices_transaction_id = pl.prices_transaction_id
    ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
    LIMIT 1
) c ON true
WHERE pl.target_type = 'source_listing'
    AND pl.target_id = $1
    AND pl.link_status <> 'rejected'
ORDER BY pl.link_score DESC, pl.updated_at DESC
LIMIT 1`, saleListingID).Scan(&transactionID, &transactionFirstSeenAt, &transactionUpdatedAt, &description, &transactionType, &category, &area, &price, &pricePerM2, &buildYear, &floor, &elevator, &condition, &plot, &plotOwned, &energyClass, &period, &city, &neighborhood, &postalCode, &matchStatus, &matchScore, &matchConfidence)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if transactionID == nil {
		return nil
	}
	areaPtr := area
	priceValue := int64(price)
	pricePerM2Value := int64(pricePerM2)
	buildYearValue := buildYear
	listing.Commercial.MatchedTransaction = &PriceTransactionMatch{ID: transactionID.String(), FirstSeenAt: transactionFirstSeenAt, UpdatedAt: transactionUpdatedAt, Description: description, Type: transactionType, Category: category, AreaM2: &areaPtr, Price: &priceValue, PricePerSquareMeter: &pricePerM2Value, BuildYear: &buildYearValue, Floor: valueOrEmpty(floor), Elevator: &elevator, Condition: displayCondition(valueOrEmpty(condition)), Plot: valueOrEmpty(plot), PlotOwned: plotOwned, EnergyClass: displayEnergyClass(valueOrEmpty(energyClass)), PeriodIdentifier: period, City: valueOrEmpty(city), Neighborhood: valueOrEmpty(neighborhood), PostalCode: valueOrEmpty(postalCode), MatchStatus: valueOrEmpty(matchStatus), MatchScore: matchScore, MatchConfidence: valueOrEmpty(matchConfidence)}
	return nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

type listingSearchRow struct {
	Source                string
	Kind                  string
	NativeID              string
	CanonicalID           string
	PublicID              string
	URL                   string
	Headline              string
	Address               string
	City                  string
	Postal                string
	Price                 *int64
	Area                  *float64
	RoomLayout            string
	PricePerM2            *float64
	DebtFreePrice         *int64
	DebtShareAmount       *int64
	RoomsCount            *int32
	FloorLevel            *int32
	TotalFloors           *int32
	BuildYear             *int32
	Condition             *string
	EnergyClass           *string
	EnergyEfficiencyLabel *string // used only for energy class computation, not exposed in API
	LastSeenAt            *string
	PublishedAt           *string
	BuildingKeyAddress    string
	SourceProviders       []string
}

func (r listingSearchRow) toSaleSummary() SaleListingSummary {
	source := ListingSource{Provider: r.Source, Kind: r.Kind, CanonicalID: r.CanonicalID, NativeID: r.NativeID, URL: r.URL, OriginalURL: r.URL}
	location := Location{StreetAddress: r.Address, City: r.City, Postal: r.Postal}
	identity := computedBuildingIdentity(r.Source, r.Kind, r.NativeID, location, "", "", "")
	energy := displayEnergyClass(valueOrEmpty(r.EnergyEfficiencyLabel), valueOrEmpty(r.EnergyClass))
	return SaleListingSummary{ID: r.PublicID, Source: source, SourceProviders: r.SourceProviders, Headline: r.Headline, Unit: UnitDetails{Location: location, RoomLayout: r.RoomLayout, RoomsCount: r.RoomsCount, AreaM2: r.Area, FloorLevel: r.FloorLevel, Condition: displayCondition(valueOrEmpty(r.Condition))}, Building: BuildingDetails{Identity: identity, Location: location, BuildYear: r.BuildYear, FloorCount: r.TotalFloors, EnergyClass: energy}, Commercial: CommercialDetails{AskingPrice: r.Price, DebtFreePrice: r.DebtFreePrice, DebtShareAmount: r.DebtShareAmount, PricePerSquareMeter: r.PricePerM2, LastSeenAt: parseTimeString(r.LastSeenAt), PublishedAt: parseTimeString(r.PublishedAt)}}
}

func (r listingSearchRow) toRentalSummary() RentalSummary {
	source := ListingSource{Provider: r.Source, Kind: r.Kind, CanonicalID: r.CanonicalID, NativeID: r.NativeID, URL: r.URL, OriginalURL: r.URL}
	location := Location{StreetAddress: r.Address, City: r.City, Postal: r.Postal}
	identity := computedBuildingIdentity(r.Source, r.Kind, r.NativeID, location, "", "", "")
	return RentalSummary{ID: r.PublicID, Source: source, Headline: r.Headline, Unit: UnitDetails{Location: location, RoomLayout: r.RoomLayout, AreaM2: r.Area}, Building: BuildingDetails{Identity: identity, Location: location}, Commercial: CommercialDetails{Rent: r.Price, RentPeriod: "month", LastSeenAt: parseTimeString(r.LastSeenAt), PublishedAt: parseTimeString(r.PublishedAt)}}
}

func parseTimeString(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(*value))
	if err != nil {
		return nil
	}
	return &t
}
