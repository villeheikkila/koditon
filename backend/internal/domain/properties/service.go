package properties

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/ads"
)

var ErrNotFound = errors.New("property not found")

type Service struct {
	db      db.DBTX
	queries *db.Queries
}

func NewService(dbtx db.DBTX) *Service {
	return &Service{db: dbtx, queries: db.New(dbtx)}
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
	offering, canonicalID, sourceListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return SaleListing{}, err
	}
	listing, err := s.saleListingByCanonicalID(ctx, canonicalID)
	if err != nil {
		return SaleListing{}, err
	}
	listing.ID = offeringID.String()
	listing.Canonical = offering
	if err := s.enrichSaleListingFromCanonicalBuilding(ctx, &listing, offeringID, sourceListingID); err != nil {
		return SaleListing{}, err
	}
	if err := s.enrichSaleListingFromSharedRow(ctx, &listing, offeringID, sourceListingID); err != nil {
		return SaleListing{}, err
	}
	records, err := s.saleOfferingSourceRecords(ctx, offeringID)
	if err != nil {
		return SaleListing{}, err
	}
	listing.SourceRecords = records
	return listing, nil
}

func (s *Service) saleListingByCanonicalID(ctx context.Context, canonicalID string) (SaleListing, error) {
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return SaleListing{}, err
	}
	switch source + ":" + kind {
	case "shortcut:ad":
		adID, err := strconv.ParseInt(nativeID, 10, 64)
		if err != nil {
			return SaleListing{}, fmt.Errorf("parse shortcut ad id: %w", err)
		}
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.ShortcutAdType != "listing" {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromShortcutAd(canonicalID, nativeID, row), nil
	case "frontdoor:ad":
		row, err := s.queries.GetFrontdoorAdUnifiedDetail(ctx, nativeID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		return saleFromFrontdoorAd(canonicalID, nativeID, row), nil
	case "frontdoor:announcement":
		announcementID, err := uuid.Parse(nativeID)
		if err != nil {
			return SaleListing{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
		}
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, announcementID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.FrontdoorBuildingAnnouncementRentPeriod != nil || row.FrontdoorBuildingAnnouncementRentalUniqueNo != nil {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromFrontdoorAnnouncement(canonicalID, nativeID, row), nil
	default:
		return SaleListing{}, fmt.Errorf("%w: unsupported sale listing id", ErrNotFound)
	}
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
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
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
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, announcementID)
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
		row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromShortcut(canonicalID, nativeID, row), nil
	case "frontdoor:building":
		buildingID, err := uuid.Parse(nativeID)
		if err != nil {
			return Building{}, fmt.Errorf("parse frontdoor building id: %w", err)
		}
		row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromFrontdoor(canonicalID, nativeID, row), nil
	default:
		return Building{}, fmt.Errorf("%w: unsupported building id", ErrNotFound)
	}
}

func (s *Service) resolveListingInput(ctx context.Context, input string, listingType string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	query := resolveSaleListingPublicIDSQL
	if listingType == "rental" {
		query = resolveRentalPublicIDSQL
	}
	var canonicalID string
	if err := s.db.QueryRow(ctx, query, strings.TrimSpace(input)).Scan(&canonicalID); err != nil {
		return "", mapNotFound(err)
	}
	return canonicalID, nil
}

func (s *Service) resolveBuildingInput(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	var canonicalID string
	if err := s.db.QueryRow(ctx, resolveBuildingPublicIDSQL, strings.TrimSpace(input)).Scan(&canonicalID); err != nil {
		return "", mapNotFound(err)
	}
	return canonicalID, nil
}

func (s *Service) saleOfferingSource(ctx context.Context, offeringID uuid.UUID) (CanonicalOffering, string, uuid.UUID, error) {
	var offering CanonicalOffering
	var sourceListingID uuid.UUID
	var canonicalID string
	err := s.db.QueryRow(ctx, `
SELECT
    po.property_offering_id::text,
    pb.property_building_id::text,
    pu.property_unit_id::text,
    selected.sale_listing_id,
    selected.sale_listing_canonical_id,
    count(pos.property_offering_source_id)::int4
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.property_buildings pb ON pb.property_building_id = pu.property_building_id
JOIN LATERAL (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_canonical_id
    FROM public.property_offering_sources linked
    JOIN public.sale_listings sl ON sl.sale_listing_id = linked.sale_listing_id
    WHERE linked.property_offering_id = po.property_offering_id
        AND linked.property_offering_source_link_status <> 'rejected'
    ORDER BY
        CASE
            WHEN sl.frontdoor_ad_id IS NOT NULL THEN 0
            WHEN sl.shortcut_ad_id IS NOT NULL THEN 1
            ELSE 2
        END,
        sl.sale_listing_last_seen_at DESC NULLS LAST,
        linked.property_offering_source_link_score DESC,
        sl.sale_listing_created_at DESC
    LIMIT 1
) selected ON true
LEFT JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
    AND pos.property_offering_source_link_status <> 'rejected'
WHERE po.property_offering_id = $1
GROUP BY po.property_offering_id, pb.property_building_id, pu.property_unit_id, selected.sale_listing_id, selected.sale_listing_canonical_id
LIMIT 1`, offeringID).Scan(&offering.OfferingID, &offering.BuildingID, &offering.UnitID, &sourceListingID, &canonicalID, &offering.SourceCount)
	if err != nil {
		return CanonicalOffering{}, "", uuid.UUID{}, mapNotFound(err)
	}
	offering.PrimarySourceListing = sourceListingID.String()
	return offering, canonicalID, sourceListingID, nil
}

func (s *Service) saleOfferingSourceRecords(ctx context.Context, offeringID uuid.UUID) ([]OfferingSourceRecord, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    sl.sale_listing_id::text,
    sl.sale_listing_public_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_canonical_id,
    sl.sale_listing_native_id,
    COALESCE(sl.sale_listing_url, ''),
    COALESCE(sl.sale_listing_headline, ''),
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    pos.property_offering_source_link_status,
    pos.property_offering_source_link_method,
    pos.property_offering_source_link_score::int4
FROM public.property_offering_sources pos
JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
WHERE pos.property_offering_id = $1
    AND pos.property_offering_source_link_status <> 'rejected'
ORDER BY sl.sale_listing_last_seen_at DESC NULLS LAST, sl.sale_listing_created_at DESC`, offeringID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OfferingSourceRecord{}
	for rows.Next() {
		var record OfferingSourceRecord
		if err := rows.Scan(&record.ID, &record.PublicID, &record.Provider, &record.Kind, &record.CanonicalID, &record.NativeID, &record.URL, &record.Headline, &record.FirstSeenAt, &record.LastSeenAt, &record.LinkStatus, &record.LinkMethod, &record.LinkScore); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
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
    sl.sale_listing_id::text,
    sl.sale_listing_public_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.sale_listing_canonical_id,
    COALESCE(
        CASE
            WHEN sl.shortcut_ad_id IS NOT NULL THEN sa.shortcut_ad_data
            WHEN sl.frontdoor_ad_id IS NOT NULL THEN fa.frontdoor_ad_data
            WHEN sl.frontdoor_building_announcement_id IS NOT NULL THEN to_jsonb(fba)
            ELSE NULL
        END,
        '{}'::jsonb
    ) AS payload
FROM public.property_offering_sources pos
JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
WHERE pos.property_offering_id = $1
    AND pos.sale_listing_id = $2
    AND pos.property_offering_source_link_status <> 'rejected'
LIMIT 1`, offeringID, sourceID).Scan(&out.ID, &out.PublicID, &out.Provider, &out.Kind, &out.NativeID, &out.Canonical, &out.Payload)
	if err != nil {
		return OfferingSourceRawPayload{}, mapNotFound(err)
	}
	return out, nil
}

func (s *Service) enrichSaleListingFromCanonicalBuilding(ctx context.Context, listing *SaleListing, offeringID uuid.UUID, saleListingID uuid.UUID) error {
	var housingCompany, businessID, address, postal, city, energyLabel, heating, heatingFuel, roofMaterial, roofType, carStorage, otherInfo *string
	var buildYear, floorCount, apartmentCount *int32
	var elevator, sauna *bool
	var elevatorRenovated, facadeRenovated, windowRenovated, roofRenovated, pipeRenovated, balconyRenovated, electricityRenovated *bool
	var elevatorRenovatedYear, facadeRenovatedYear, windowRenovatedYear, roofRenovatedYear, pipeRenovatedYear, balconyRenovatedYear, electricityRenovatedYear *int32
	err := s.db.QueryRow(ctx, `
SELECT
    COALESCE(NULLIF(fb.frontdoor_building_company_name, ''), pb.property_building_housing_company),
    COALESCE(NULLIF(fb.frontdoor_building_business_id, ''), pb.property_building_business_id),
    COALESCE(NULLIF(trim(concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number)), ''), pb.property_building_address_norm),
    COALESCE(NULLIF(fb.frontdoor_building_postcode, ''), pb.property_building_postal_norm),
    COALESCE(NULLIF(fb.frontdoor_building_municipality, ''), pb.property_building_city_norm),
    COALESCE(fb.frontdoor_building_build_year, fb.frontdoor_building_construction_end_year, pb.property_building_build_year)::int4,
    COALESCE(fb.frontdoor_building_floor_count, pb.property_building_floor_count)::int4,
    COALESCE(fb.frontdoor_building_apartment_count, pb.property_building_apartment_count)::int4,
    COALESCE(fb.frontdoor_building_has_elevator, pb.property_building_elevator),
    fb.frontdoor_building_has_sauna,
    COALESCE(NULLIF(fb.frontdoor_building_energy_certificate_code, ''), pb.property_building_energy_efficiency_label),
    fb.frontdoor_building_heating,
    fb.frontdoor_building_heating_fuel,
    fb.frontdoor_building_outer_roof_material,
    fb.frontdoor_building_outer_roof_type,
    fb.frontdoor_building_car_storage_description,
    fb.frontdoor_building_other_info,
    fb.frontdoor_building_elevator_renovated,
    fb.frontdoor_building_elevator_renovated_year,
    fb.frontdoor_building_facade_renovated,
    fb.frontdoor_building_facade_renovated_year,
    fb.frontdoor_building_window_renovated,
    fb.frontdoor_building_window_renovated_year,
    fb.frontdoor_building_roof_renovated,
    fb.frontdoor_building_roof_renovated_year,
    fb.frontdoor_building_pipe_renovated,
    fb.frontdoor_building_pipe_renovated_year,
    fb.frontdoor_building_balcony_renovated,
    fb.frontdoor_building_balcony_renovated_year,
    fb.frontdoor_building_electricity_renovated,
    fb.frontdoor_building_electricity_renovated_year
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.property_buildings pb ON pb.property_building_id = pu.property_building_id
LEFT JOIN public.sale_listings sl ON sl.sale_listing_id = $2
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE po.property_offering_id = $1
LIMIT 1`, offeringID, saleListingID).Scan(&housingCompany, &businessID, &address, &postal, &city, &buildYear, &floorCount, &apartmentCount, &elevator, &sauna, &energyLabel, &heating, &heatingFuel, &roofMaterial, &roofType, &carStorage, &otherInfo, &elevatorRenovated, &elevatorRenovatedYear, &facadeRenovated, &facadeRenovatedYear, &windowRenovated, &windowRenovatedYear, &roofRenovated, &roofRenovatedYear, &pipeRenovated, &pipeRenovatedYear, &balconyRenovated, &balconyRenovatedYear, &electricityRenovated, &electricityRenovatedYear)
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
	listing.Building.BuildYear = firstInt32(listing.Building.BuildYear, buildYear)
	listing.Building.FloorCount = firstInt32(listing.Building.FloorCount, floorCount)
	listing.Building.ApartmentCount = firstInt32(listing.Building.ApartmentCount, apartmentCount)
	listing.Building.Elevator = firstBool(listing.Building.Elevator, elevator)
	listing.Building.Sauna = firstBool(listing.Building.Sauna, sauna)
	listing.Building.EnergyClass = firstNonEmpty(listing.Building.EnergyClass, valueOrEmpty(energyLabel))
	listing.Building.EnergyEfficiencyLabel = firstNonEmpty(listing.Building.EnergyEfficiencyLabel, valueOrEmpty(energyLabel))
	listing.Building.Heating = firstNonEmpty(listing.Building.Heating, valueOrEmpty(heating))
	listing.Building.HeatingFuel = firstNonEmpty(listing.Building.HeatingFuel, valueOrEmpty(heatingFuel))
	listing.Building.RoofMaterial = firstNonEmpty(listing.Building.RoofMaterial, valueOrEmpty(roofMaterial))
	listing.Building.RoofType = firstNonEmpty(listing.Building.RoofType, valueOrEmpty(roofType))
	listing.Building.CarStorage = firstNonEmpty(listing.Building.CarStorage, valueOrEmpty(carStorage))
	listing.Building.OtherInfo = firstNonEmpty(listing.Building.OtherInfo, valueOrEmpty(otherInfo))
	listing.Building.Renovations = append(listing.Building.Renovations,
		buildingRenovation("Elevator", elevatorRenovated, elevatorRenovatedYear),
		buildingRenovation("Facade", facadeRenovated, facadeRenovatedYear),
		buildingRenovation("Windows", windowRenovated, windowRenovatedYear),
		buildingRenovation("Roof", roofRenovated, roofRenovatedYear),
		buildingRenovation("Pipes", pipeRenovated, pipeRenovatedYear),
		buildingRenovation("Balcony", balconyRenovated, balconyRenovatedYear),
		buildingRenovation("Electricity", electricityRenovated, electricityRenovatedYear),
	)
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
    pot.property_offering_transaction_link_status,
    COALESCE(c.sale_listing_prices_transaction_match_score, pot.property_offering_transaction_link_score),
    c.sale_listing_prices_transaction_match_confidence
FROM public.property_offering_transactions pot
JOIN public.prices_transactions pt ON pt.prices_transaction_id = pot.prices_transaction_id
LEFT JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN public.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN public.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN LATERAL (
    SELECT
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence
    FROM public.sale_listing_prices_transaction_match_candidates c
    JOIN public.property_offering_sources pos ON pos.sale_listing_id = c.sale_listing_id
    WHERE pos.property_offering_id = pot.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
        AND c.prices_transaction_id = pot.prices_transaction_id
    ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
    LIMIT 1
) c ON true
WHERE pot.property_offering_id = $1
    AND pot.property_offering_transaction_link_status <> 'rejected'
ORDER BY pot.property_offering_transaction_link_score DESC, pot.property_offering_transaction_updated_at DESC
LIMIT 1`, offeringID).Scan(&transactionID, &transactionFirstSeenAt, &transactionUpdatedAt, &description, &transactionType, &category, &area, &price, &pricePerM2, &buildYear, &floor, &elevator, &condition, &plot, &plotOwned, &energyClass, &period, &city, &neighborhood, &postalCode, &matchStatus, &matchScore, &matchConfidence)
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
	listing.Commercial.MatchedTransaction = &PriceTransactionMatch{ID: transactionID.String(), FirstSeenAt: transactionFirstSeenAt, UpdatedAt: transactionUpdatedAt, Description: description, Type: transactionType, Category: category, AreaM2: &areaPtr, Price: &priceValue, PricePerSquareMeter: &pricePerM2Value, BuildYear: &buildYearValue, Floor: valueOrEmpty(floor), Elevator: &elevator, Condition: valueOrEmpty(condition), Plot: valueOrEmpty(plot), PlotOwned: plotOwned, EnergyClass: valueOrEmpty(energyClass), PeriodIdentifier: period, City: valueOrEmpty(city), Neighborhood: valueOrEmpty(neighborhood), PostalCode: valueOrEmpty(postalCode), MatchStatus: valueOrEmpty(matchStatus), MatchScore: matchScore, MatchConfidence: valueOrEmpty(matchConfidence)}
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
    sl.sale_listing_prices_match_status,
    c.sale_listing_prices_transaction_match_score,
    c.sale_listing_prices_transaction_match_confidence
FROM public.sale_listings sl
LEFT JOIN public.prices_transactions pt ON pt.prices_transaction_id = sl.prices_transaction_id
LEFT JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN public.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN public.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN LATERAL (
    SELECT
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.sale_listing_id = sl.sale_listing_id
        AND c.prices_transaction_id = sl.prices_transaction_id
    ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
    LIMIT 1
) c ON true
WHERE sl.sale_listing_id = $1
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
	listing.Commercial.MatchedTransaction = &PriceTransactionMatch{ID: transactionID.String(), FirstSeenAt: transactionFirstSeenAt, UpdatedAt: transactionUpdatedAt, Description: description, Type: transactionType, Category: category, AreaM2: &areaPtr, Price: &priceValue, PricePerSquareMeter: &pricePerM2Value, BuildYear: &buildYearValue, Floor: valueOrEmpty(floor), Elevator: &elevator, Condition: valueOrEmpty(condition), Plot: valueOrEmpty(plot), PlotOwned: plotOwned, EnergyClass: valueOrEmpty(energyClass), PeriodIdentifier: period, City: valueOrEmpty(city), Neighborhood: valueOrEmpty(neighborhood), PostalCode: valueOrEmpty(postalCode), MatchStatus: valueOrEmpty(matchStatus), MatchScore: matchScore, MatchConfidence: valueOrEmpty(matchConfidence)}
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
	EnergyEfficiencyLabel *string
	LastSeenAt            *string
	PublishedAt           *string
	BuildingKeyAddress    string
	SourceProviders       []string
}

func (r listingSearchRow) toSaleSummary() SaleListingSummary {
	source := ListingSource{Provider: r.Source, Kind: r.Kind, CanonicalID: r.CanonicalID, NativeID: r.NativeID, URL: r.URL, OriginalURL: r.URL}
	location := Location{StreetAddress: r.Address, City: r.City, Postal: r.Postal}
	identity := computedBuildingIdentity(r.Source, r.Kind, r.NativeID, location, "", "", "")
	return SaleListingSummary{ID: r.PublicID, Source: source, SourceProviders: r.SourceProviders, Headline: r.Headline, Unit: UnitDetails{Location: location, RoomLayout: r.RoomLayout, RoomsCount: r.RoomsCount, AreaM2: r.Area, FloorLevel: r.FloorLevel, Condition: valueOrEmpty(r.Condition)}, Building: BuildingDetails{Identity: identity, Location: location, BuildYear: r.BuildYear, FloorCount: r.TotalFloors, EnergyClass: valueOrEmpty(r.EnergyClass), EnergyEfficiencyLabel: valueOrEmpty(r.EnergyEfficiencyLabel)}, Commercial: CommercialDetails{AskingPrice: r.Price, DebtFreePrice: r.DebtFreePrice, DebtShareAmount: r.DebtShareAmount, PricePerSquareMeter: r.PricePerM2, LastSeenAt: parseTimeString(r.LastSeenAt), PublishedAt: parseTimeString(r.PublishedAt)}}
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
