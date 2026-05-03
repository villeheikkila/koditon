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

type saleListingSourceRow struct {
	SaleListingID                   uuid.UUID
	Provider                        string
	Kind                            string
	NativeID                        string
	CanonicalID                     string
	ShortcutAdID                    *int64
	FrontdoorAdID                   *uuid.UUID
	FrontdoorBuildingAnnouncementID *uuid.UUID
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
	listing.SourceRecords = records
	return listing, nil
}

func (s *Service) saleListingBySourceID(ctx context.Context, saleListingID uuid.UUID) (SaleListing, error) {
	source, err := s.saleListingSourceRow(ctx, saleListingID)
	if err != nil {
		return SaleListing{}, err
	}
	switch {
	case source.ShortcutAdID != nil:
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, *source.ShortcutAdID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.ShortcutAdType != "listing" {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromShortcutAd(source.CanonicalID, source.NativeID, row), nil
	case source.FrontdoorAdID != nil:
		row, err := s.queries.GetFrontdoorAdUnifiedDetail(ctx, source.NativeID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		return saleFromFrontdoorAd(source.CanonicalID, source.NativeID, row), nil
	case source.FrontdoorBuildingAnnouncementID != nil:
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, *source.FrontdoorBuildingAnnouncementID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.FrontdoorBuildingAnnouncementRentPeriod != nil || row.FrontdoorBuildingAnnouncementRentalUniqueNo != nil {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromFrontdoorAnnouncement(source.CanonicalID, source.NativeID, row), nil
	default:
		return SaleListing{}, fmt.Errorf("%w: sale listing has no source row", ErrNotFound)
	}
}

func (s *Service) saleListingSourceRow(ctx context.Context, saleListingID uuid.UUID) (saleListingSourceRow, error) {
	var source saleListingSourceRow
	err := s.db.QueryRow(ctx, `
SELECT
    sale_listing_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    shortcut_ad_id,
    frontdoor_ad_id,
    frontdoor_building_announcement_id
FROM public.property_source_offerings
WHERE sale_listing_id = $1
LIMIT 1`, saleListingID).Scan(&source.SaleListingID, &source.Provider, &source.Kind, &source.NativeID, &source.CanonicalID, &source.ShortcutAdID, &source.FrontdoorAdID, &source.FrontdoorBuildingAnnouncementID)
	if err != nil {
		return saleListingSourceRow{}, mapNotFound(err)
	}
	return source, nil
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

func (s *Service) buildingByHousingCompanyID(ctx context.Context, housingCompanyID uuid.UUID) (Building, error) {
	var building Building
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
LIMIT 1`, housingCompanyID).Scan(&building.ID, &building.Details.Identity.Key, &building.Details.Location.StreetAddress, &building.Details.Location.City, &building.Details.Location.Postal, &building.Details.HousingCompany, &building.Details.BusinessID, &building.Details.BuildYear, &building.Details.FloorCount, &building.Details.ApartmentCount, &building.Details.Elevator, &building.Details.EnergyEfficiencyLabel, &latitude, &longitude, &mergeDecisionCount, &mergedFrom)
	if err != nil {
		return Building{}, mapNotFound(err)
	}
	building.Details.Identity.Strategy = "canonical_housing_company"
	building.Details.Identity.Confidence = 1
	building.Details.Location.Latitude = latitude
	building.Details.Location.Longitude = longitude
	building.Details.EnergyClass = building.Details.EnergyEfficiencyLabel
	if mergeDecisionCount > 0 {
		building.Metadata = map[string]any{
			"merge_decision_count": mergeDecisionCount,
			"merged_from":          mergedFrom,
		}
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

func (s *Service) enrichBuildingFromOfferingSources(ctx context.Context, building *Building, buildingID uuid.UUID) error {
	rows, err := s.db.Query(ctx, `
SELECT
    sl.sale_listing_id
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
WHERE pu.housing_company_id = $1
    AND pos.property_offering_source_link_status <> 'rejected'
    AND sl.sale_listing_source_kind IN ('ad', 'announcement')
ORDER BY
    CASE WHEN sl.sale_listing_source_kind = 'ad' THEN 0 ELSE 1 END,
    sl.sale_listing_last_seen_at DESC NULLS LAST,
    pos.property_offering_source_link_score DESC,
    sl.sale_listing_created_at DESC
LIMIT 200`, buildingID)
	if err != nil {
		return err
	}
	defer rows.Close()
	seen := map[string]struct{}{}
	for rows.Next() {
		var saleListingID uuid.UUID
		if err := rows.Scan(&saleListingID); err != nil {
			return err
		}
		sourceKey := saleListingID.String()
		if _, ok := seen[sourceKey]; ok {
			continue
		}
		seen[sourceKey] = struct{}{}
		listing, err := s.saleListingBySourceID(ctx, saleListingID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return err
		}
		mergeBuildingDetails(&building.Details, listing.Building)
		mergeSiteDetails(&building.Site, listing.Site)
		mergeTextSections(&building.Texts, listing.Texts)
		building.SourceRecords = appendUniqueListingSources(building.SourceRecords, []ListingSource{listing.Source})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
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
    pb.housing_company_build_year,
    COALESCE(po.property_offering_last_seen_at, source_summary.last_seen_at),
    COALESCE(source_summary.providers, ARRAY[]::text[]),
    COALESCE(source_summary.kinds, ARRAY[]::text[])
FROM public.property_units pu
JOIN public.housing_companies pb ON pb.housing_company_id = pu.housing_company_id
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
LEFT JOIN LATERAL (
    SELECT
        max(NULLIF(sl.sale_listing_headline, '')) AS headline,
        min(sl.sale_listing_asking_price) FILTER (WHERE sl.sale_listing_asking_price IS NOT NULL) AS asking_price,
        min(sl.sale_listing_price_per_m2) FILTER (WHERE sl.sale_listing_price_per_m2 IS NOT NULL) AS price_per_m2,
        max(sl.sale_listing_last_seen_at) AS last_seen_at,
        array_agg(DISTINCT sl.sale_listing_source_provider ORDER BY sl.sale_listing_source_provider) AS providers,
        array_agg(DISTINCT sl.sale_listing_source_kind ORDER BY sl.sale_listing_source_kind) AS kinds
    FROM public.property_offering_sources pos
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pos.property_offering_id = po.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
) source_summary ON true
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
		if err := rows.Scan(&item.ID, &item.Kind, &item.FriendlyID, &item.Address, &item.RoomLayout, &item.AreaM2, &item.Price, &item.PricePerM2, &item.BuildYear, &item.LastSeenAt, &item.Providers, &item.Kinds); err != nil {
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
	var canonicalID string
	if err := s.db.QueryRow(ctx, resolveRentalPublicIDSQL, strings.TrimSpace(input)).Scan(&canonicalID); err != nil {
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

func (s *Service) saleOfferingSource(ctx context.Context, offeringID uuid.UUID) (CanonicalOffering, uuid.UUID, error) {
	var offering CanonicalOffering
	var sourceListingID uuid.UUID
	err := s.db.QueryRow(ctx, `
SELECT
    po.property_offering_id::text,
    pb.housing_company_id::text,
    pu.property_unit_id::text,
    selected.sale_listing_id,
    count(DISTINCT pos.property_offering_source_id)::int4,
    count(DISTINCT merges.source_property_offering_id)::int4,
    COALESCE(array_agg(DISTINCT merges.source_property_offering_id::text) FILTER (WHERE merges.source_property_offering_id IS NOT NULL), ARRAY[]::text[])
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.housing_companies pb ON pb.housing_company_id = pu.housing_company_id
JOIN LATERAL (
    SELECT
        sl.sale_listing_id
    FROM public.property_offering_sources linked
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = linked.sale_listing_id
    WHERE linked.property_offering_id = po.property_offering_id
        AND linked.property_offering_source_link_status <> 'rejected'
    ORDER BY
        CASE WHEN sl.sale_listing_source_kind = 'ad' THEN 0 ELSE 1 END,
        CASE WHEN sl.sale_listing_asking_price IS NOT NULL THEN 0 ELSE 1 END,
        CASE WHEN NULLIF(sl.sale_listing_description_text, '') IS NOT NULL THEN 0 ELSE 1 END,
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
LEFT JOIN public.property_offering_merge_decisions merges ON merges.target_property_offering_id = po.property_offering_id
    AND merges.property_offering_merge_decision_status = 'accepted'
WHERE po.property_offering_id = $1
GROUP BY po.property_offering_id, pb.housing_company_id, pu.property_unit_id, selected.sale_listing_id
LIMIT 1`, offeringID).Scan(&offering.OfferingID, &offering.HousingCompanyID, &offering.UnitID, &sourceListingID, &offering.SourceCount, &offering.MergeDecisionCount, &offering.MergedFrom)
	if err != nil {
		return CanonicalOffering{}, uuid.UUID{}, mapNotFound(err)
	}
	offering.PrimarySourceListing = sourceListingID.String()
	return offering, sourceListingID, nil
}

func (s *Service) saleOfferingSourceRecords(ctx context.Context, offeringID uuid.UUID) ([]OfferingSourceRecord, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    sl.sale_listing_id::text,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    COALESCE(sl.sale_listing_url, ''),
    COALESCE(sl.sale_listing_headline, ''),
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    pos.property_offering_source_link_status,
    pos.property_offering_source_link_method,
    pos.property_offering_source_link_score::int4
FROM public.property_offering_sources pos
JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
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
	mergeUnitDetails(&dst.Unit, src.Unit)
	mergeBuildingDetails(&dst.Building, src.Building)
	mergeSiteDetails(&dst.Site, src.Site)
	mergeCommercialDetails(&dst.Commercial, src.Commercial)
	mergeTextSections(&dst.Texts, src.Texts)
	mergeMedia(&dst.Media, src.Media)
	dst.Contacts = appendUniqueContacts(dst.Contacts, src.Contacts)
	dst.Showings = append(dst.Showings, src.Showings...)
	dst.Links = appendUniqueLinks(dst.Links, src.Links)
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
	dst.Condition = firstNonEmpty(dst.Condition, src.Condition)
	dst.Sauna = firstBool(dst.Sauna, src.Sauna)
	dst.Balcony = firstBool(dst.Balcony, src.Balcony)
	dst.Parking = firstNonEmpty(dst.Parking, src.Parking)
	dst.Availability = firstNonEmpty(dst.Availability, src.Availability)
	dst.KitchenDescription = firstNonEmpty(dst.KitchenDescription, src.KitchenDescription)
	dst.BathroomDescription = firstNonEmpty(dst.BathroomDescription, src.BathroomDescription)
	dst.StorageDescription = firstNonEmpty(dst.StorageDescription, src.StorageDescription)
	dst.FloorMaterialsDescription = firstNonEmpty(dst.FloorMaterialsDescription, src.FloorMaterialsDescription)
	dst.WallMaterialsDescription = firstNonEmpty(dst.WallMaterialsDescription, src.WallMaterialsDescription)
	dst.BalconyDescription = firstNonEmpty(dst.BalconyDescription, src.BalconyDescription)
	dst.SaunaDescription = firstNonEmpty(dst.SaunaDescription, src.SaunaDescription)
	dst.ViewsDescription = firstNonEmpty(dst.ViewsDescription, src.ViewsDescription)
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
	dst.EnergyClass = firstNonEmpty(dst.EnergyClass, src.EnergyClass)
	dst.EnergyEfficiencyLabel = firstNonEmpty(dst.EnergyEfficiencyLabel, src.EnergyEfficiencyLabel)
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
	dst.Sauna = firstBool(dst.Sauna, src.Sauna)
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
	dst.Services = firstNonEmpty(dst.Services, src.Services)
	dst.Transport = firstNonEmpty(dst.Transport, src.Transport)
	dst.WaterSupplyTypes = compactStrings(append(dst.WaterSupplyTypes, src.WaterSupplyTypes...))
}

func mergeCommercialDetails(dst *CommercialDetails, src CommercialDetails) {
	matchedTransaction := dst.MatchedTransaction
	dst.Status = firstNonEmpty(dst.Status, src.Status)
	dst.BookingStatus = firstNonEmpty(dst.BookingStatus, src.BookingStatus)
	dst.PublishedAt = firstTime(dst.PublishedAt, src.PublishedAt)
	dst.UnpublishedAt = firstTime(dst.UnpublishedAt, src.UnpublishedAt)
	dst.FirstSeenAt = firstTime(dst.FirstSeenAt, src.FirstSeenAt)
	dst.LastSeenAt = firstTime(dst.LastSeenAt, src.LastSeenAt)
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
	dst.NewDevelopment = firstBool(dst.NewDevelopment, src.NewDevelopment)
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
	dst.Notes = firstNonEmpty(dst.Notes, src.Notes)
}

func mergeTextSections(dst *TextSections, src TextSections) {
	dst.Description = firstNonEmpty(dst.Description, src.Description)
	dst.Availability = firstNonEmpty(dst.Availability, src.Availability)
	dst.RenovationsDone = firstNonEmpty(dst.RenovationsDone, src.RenovationsDone)
	dst.RenovationsPlanned = firstNonEmpty(dst.RenovationsPlanned, src.RenovationsPlanned)
	dst.AdditionalInfo = firstNonEmpty(dst.AdditionalInfo, src.AdditionalInfo)
	dst.Area = firstNonEmpty(dst.Area, src.Area)
	dst.Building = firstNonEmpty(dst.Building, src.Building)
	dst.Transport = firstNonEmpty(dst.Transport, src.Transport)
	dst.Amenities = firstNonEmpty(dst.Amenities, src.Amenities)
	dst.Charges = firstNonEmpty(dst.Charges, src.Charges)
	dst.Kitchen = firstNonEmpty(dst.Kitchen, src.Kitchen)
	dst.Bathroom = firstNonEmpty(dst.Bathroom, src.Bathroom)
	dst.Storage = firstNonEmpty(dst.Storage, src.Storage)
	dst.Materials = firstNonEmpty(dst.Materials, src.Materials)
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

func firstTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
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
    sl.sale_listing_id::text,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
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
JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
WHERE pos.property_offering_id = $1
    AND pos.sale_listing_id = $2
    AND pos.property_offering_source_link_status <> 'rejected'
LIMIT 1`, offeringID, sourceID).Scan(&out.ID, &out.Provider, &out.Kind, &out.NativeID, &out.Payload)
	if err != nil {
		return OfferingSourceRawPayload{}, mapNotFound(err)
	}
	return out, nil
}

func (s *Service) enrichSaleListingFromCanonicalBuilding(ctx context.Context, listing *SaleListing, offeringID uuid.UUID, saleListingID uuid.UUID) error {
	var housingCompany, businessID, address, postal, city, energyLabel, heating, heatingFuel, roofMaterial, roofType, carStorage, otherInfo *string
	var buildYear, floorCount, apartmentCount *int32
	var elevator, sauna *bool
	var latitude, longitude *float64
	var elevatorRenovated, facadeRenovated, windowRenovated, roofRenovated, pipeRenovated, balconyRenovated, electricityRenovated *bool
	var elevatorRenovatedYear, facadeRenovatedYear, windowRenovatedYear, roofRenovatedYear, pipeRenovatedYear, balconyRenovatedYear, electricityRenovatedYear *int32
	err := s.db.QueryRow(ctx, `
SELECT
    COALESCE(NULLIF(fb.frontdoor_building_company_name, ''), pb.housing_company_name),
    COALESCE(NULLIF(fb.frontdoor_building_business_id, ''), pb.housing_company_business_id),
    COALESCE(NULLIF(trim(concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number)), ''), pb.housing_company_address_norm),
    COALESCE(NULLIF(fb.frontdoor_building_postcode, ''), pb.housing_company_postal_norm),
    COALESCE(NULLIF(fb.frontdoor_building_municipality, ''), pb.housing_company_city_norm),
    COALESCE(fb.frontdoor_building_build_year, fb.frontdoor_building_construction_end_year, pb.housing_company_build_year)::int4,
    COALESCE(fb.frontdoor_building_floor_count, pb.housing_company_floor_count)::int4,
    COALESCE(fb.frontdoor_building_apartment_count, pb.housing_company_apartment_count)::int4,
    COALESCE(fb.frontdoor_building_has_elevator, pb.housing_company_elevator),
    fb.frontdoor_building_has_sauna,
    COALESCE(NULLIF(fb.frontdoor_building_energy_certificate_code, ''), pb.housing_company_energy_efficiency_label),
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
    fb.frontdoor_building_electricity_renovated_year,
    postgis.ST_Y(pb.housing_company_geom)::double precision,
    postgis.ST_X(pb.housing_company_geom)::double precision
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
JOIN public.housing_companies pb ON pb.housing_company_id = pu.housing_company_id
LEFT JOIN public.property_source_offerings sl ON sl.sale_listing_id = $2
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE po.property_offering_id = $1
LIMIT 1`, offeringID, saleListingID).Scan(&housingCompany, &businessID, &address, &postal, &city, &buildYear, &floorCount, &apartmentCount, &elevator, &sauna, &energyLabel, &heating, &heatingFuel, &roofMaterial, &roofType, &carStorage, &otherInfo, &elevatorRenovated, &elevatorRenovatedYear, &facadeRenovated, &facadeRenovatedYear, &windowRenovated, &windowRenovatedYear, &roofRenovated, &roofRenovatedYear, &pipeRenovated, &pipeRenovatedYear, &balconyRenovated, &balconyRenovatedYear, &electricityRenovated, &electricityRenovatedYear, &latitude, &longitude)
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
FROM public.property_source_offerings sl
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
