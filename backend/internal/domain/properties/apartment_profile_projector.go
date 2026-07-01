package properties

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
)

func (s *Service) ProjectSaleListingCanonicalProfile(ctx context.Context, input string) (CanonicalProfileProjectionResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return CanonicalProfileProjectionResult{}, ErrNotFound
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return CanonicalProfileProjectionResult{}, err
	}
	if err := s.projectSaleListingCanonicalProfile(ctx, saleListingID); err != nil {
		return CanonicalProfileProjectionResult{}, err
	}
	profile, err := s.dimensionApartmentProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		return CanonicalProfileProjectionResult{}, err
	}
	return CanonicalProfileProjectionResult{SaleListingID: saleListingID.String(), ApartmentProfile: profile}, nil
}

func (s *Service) projectSaleListingCanonicalProfile(ctx context.Context, saleListingID uuid.UUID) error {
	if _, err := s.queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "profile_projection_requested"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty: %w", err)
	}
	return nil
}

func (s *Service) enrichSaleListingCanonicalApartmentProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	profile, err := s.dimensionApartmentProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	listing.ApartmentProfile = profile
	return nil
}

func (s *Service) dimensionApartmentProfileForSaleListing(ctx context.Context, saleListingID uuid.UUID) (ApartmentProfile, error) {
	row := s.db.QueryRow(ctx, `
WITH linked AS (
    SELECT pu.property_unit_id, pu.housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = $1
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC
    LIMIT 1
)
SELECT
    linked.housing_company_id,
    linked.property_unit_id,
    (p.dimensions #>> '{unit,area_m2}')::double precision,
    (p.dimensions #>> '{unit,living_area_m2}')::double precision,
    COALESCE(p.dimensions #>> '{layout,room_layout}', ''),
    (p.dimensions #>> '{layout,room_count}')::integer,
    (p.dimensions #>> '{layout,bedroom_count}')::integer,
    (p.dimensions #>> '{unit,floor_level}')::integer,
    (p.dimensions #>> '{unit,total_floors}')::integer,
    COALESCE(p.dimensions #>> '{layout,kitchen_type}', ''),
    COALESCE(p.dimensions #>> '{layout,quality}', ''),
    (p.dimensions #>> '{layout,awkward}')::boolean,
    COALESCE(p.dimensions #>> '{condition,unit_condition}', ''),
    COALESCE(p.dimensions #>> '{condition,kitchen_condition}', ''),
    COALESCE(p.dimensions #>> '{condition,bathroom_condition}', ''),
    (p.dimensions #>> '{condition,surface_renovation_need}')::boolean,
    (p.dimensions #>> '{condition,modernization_need}')::boolean,
    (p.dimensions #>> '{features,sauna}')::boolean,
    (p.dimensions #>> '{features,balcony}')::boolean,
    (p.dimensions #>> '{features,balcony_glazing}')::boolean,
    COALESCE(p.dimensions #>> '{features,parking_type}', ''),
    COALESCE(p.dimensions #>> '{features,storage_quality}', ''),
    COALESCE(p.dimensions #>> '{features,view_quality}', ''),
    (p.dimensions #>> '{features,noise_risk}')::boolean,
    COALESCE(p.dimensions #>> '{features,accessibility}', ''),
    (p.dimensions #>> '{charges,maintenance_monthly_eur}')::double precision,
    (p.dimensions #>> '{charges,capital_monthly_eur}')::double precision,
    (p.dimensions #>> '{charges,total_monthly_eur}')::double precision,
    (p.dimensions #>> '{charges,debt_share_eur}')::bigint,
    COALESCE(p.dimensions #>> '{risk,shareholder_liability}', ''),
    'medium'::text,
    p.resolved_at
FROM linked
JOIN public.dimension_profiles p ON p.target_type = 'unit'
    AND p.target_id = linked.property_unit_id`, saleListingID)
	var profile ApartmentProfile
	var housingCompanyID, propertyUnitID uuid.UUID
	var updatedAt time.Time
	err := row.Scan(&housingCompanyID, &propertyUnitID, &profile.AreaM2, &profile.LivingAreaM2, &profile.RoomLayout, &profile.RoomCount, &profile.BedroomCount, &profile.FloorLevel, &profile.TotalFloors, &profile.KitchenType, &profile.LayoutQuality, &profile.AwkwardLayout, &profile.Condition, &profile.KitchenCondition, &profile.BathroomCondition, &profile.SurfaceRenovationNeed, &profile.ModernizationNeed, &profile.Sauna, &profile.Balcony, &profile.BalconyGlazing, &profile.ParkingType, &profile.StorageQuality, &profile.ViewQuality, &profile.NoiseRisk, &profile.Accessibility, &profile.MaintenanceChargeMonthly, &profile.CapitalChargeMonthly, &profile.TotalChargeMonthly, &profile.DebtShareEUR, &profile.ShareholderLiability, &profile.Confidence, &updatedAt)
	if err != nil {
		return ApartmentProfile{}, err
	}
	profile.HousingCompanyID = housingCompanyID.String()
	profile.PropertyUnitID = propertyUnitID.String()
	profile.UpdatedAt = updatedAt.Format(time.RFC3339)
	return profile, nil
}

func (s *Service) enrichSaleListingCanonicalBuildingProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row := s.db.QueryRow(ctx, `
WITH linked AS (
    SELECT pu.physical_building_id, pu.housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = $1
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC NULLS LAST, source_link.created_at DESC
    LIMIT 1
)
SELECT
    linked.physical_building_id,
    linked.housing_company_id,
    (bp.dimensions #>> '{building,build_year}')::integer,
    (bp.dimensions #>> '{building,floor_count}')::integer,
    (bp.dimensions #>> '{building,apartment_count}')::integer,
    COALESCE(bp.dimensions #>> '{building,energy_class}', ''),
    COALESCE(bp.dimensions #>> '{building,heating_method}', ''),
    COALESCE(bp.dimensions #>> '{building,material}', ''),
    COALESCE(bp.dimensions #>> '{building,roof_type}', ''),
    COALESCE(bp.dimensions #>> '{building,roof_material}', ''),
    (bp.dimensions #>> '{building,elevator}')::boolean,
    CASE WHEN bp.target_id IS NULL THEN '' ELSE 'medium' END,
    bp.resolved_at,
    linked.housing_company_id,
    COALESCE(hcp.dimensions #>> '{housing_company,name}', ''),
    COALESCE(hcp.dimensions #>> '{housing_company,business_id}', ''),
    NULL::integer,
    (hcp.dimensions #>> '{housing_company,apartment_count}')::integer,
    COALESCE(hcp.dimensions #>> '{site,plot_ownership_type}', ''),
    ''::text,
    COALESCE(hcp.dimensions #>> '{risk,maintenance_risk}', ''),
    COALESCE(hcp.dimensions #>> '{risk,financial_risk}', ''),
    COALESCE(hcp.dimensions #>> '{risk,repair_backlog_risk}', ''),
    CASE WHEN hcp.target_id IS NULL THEN '' ELSE 'medium' END,
    hcp.resolved_at
FROM linked
LEFT JOIN public.dimension_profiles bp ON bp.target_type = 'building'
    AND bp.target_id = linked.physical_building_id
LEFT JOIN public.dimension_profiles hcp ON hcp.target_type = 'housing_company'
    AND hcp.target_id = linked.housing_company_id`, saleListingID)
	var buildingID, buildingHousingCompanyID, housingCompanyID *uuid.UUID
	var buildingUpdatedAt, housingCompanyUpdatedAt *time.Time
	var profile BuildingProfile
	var housingProfile HousingCompanyProfile
	err := row.Scan(&buildingID, &buildingHousingCompanyID, &profile.BuildYear, &profile.FloorCount, &profile.ApartmentCount, &profile.EnergyClass, &profile.HeatingMethod, &profile.Material, &profile.RoofType, &profile.RoofMaterial, &profile.Elevator, &profile.Confidence, &buildingUpdatedAt, &housingCompanyID, &housingProfile.Name, &housingProfile.BusinessID, &housingProfile.BuildYear, &housingProfile.ApartmentCount, &housingProfile.PlotOwnershipType, &housingProfile.EnergyClass, &housingProfile.MaintenanceRisk, &housingProfile.FinancialRisk, &housingProfile.RepairBacklogRisk, &housingProfile.Confidence, &housingCompanyUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	profile.PhysicalBuildingID = ptrUUIDString(buildingID)
	profile.HousingCompanyID = ptrUUIDString(buildingHousingCompanyID)
	if buildingUpdatedAt != nil {
		profile.UpdatedAt = buildingUpdatedAt.Format(time.RFC3339)
	}
	housingProfile.HousingCompanyID = ptrUUIDString(housingCompanyID)
	if housingCompanyUpdatedAt != nil {
		housingProfile.UpdatedAt = housingCompanyUpdatedAt.Format(time.RFC3339)
	}
	listing.BuildingProfile = profile
	listing.HousingProfile = housingProfile
	return nil
}

func (s *Service) enrichSaleListingQualityScores(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	rows, err := s.db.Query(ctx, `
WITH linked AS (
    SELECT po.property_offering_id, pu.property_unit_id, pu.physical_building_id, pu.housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = $1
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC NULLS LAST, source_link.created_at DESC
    LIMIT 1
),
targets AS (
    SELECT 'offering'::text AS target_type, property_offering_id AS target_id FROM linked WHERE property_offering_id IS NOT NULL
    UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
    UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
    UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
)
SELECT
    CASE dv.target_type
        WHEN 'offering' THEN 'property_offering'
        WHEN 'unit' THEN 'property_unit'
        WHEN 'building' THEN 'physical_building'
        ELSE dv.target_type
    END,
    substring(dv.dimension_key from position('.' in dv.dimension_key) + 1),
    round((dv.confidence * 100)::numeric)::integer,
    CASE WHEN dv.confidence >= 0.8 THEN 'high' WHEN dv.confidence >= 0.6 THEN 'medium' ELSE 'low' END,
    jsonb_build_array(dv.selected_reason),
    dv.resolved_at
FROM targets
JOIN public.dimension_values dv
    ON dv.target_type = targets.target_type
    AND dv.target_id = targets.target_id
    AND dv.dimension_key LIKE 'score.%'
ORDER BY
    CASE dv.target_type WHEN 'offering' THEN 1 WHEN 'unit' THEN 2 WHEN 'building' THEN 3 WHEN 'housing_company' THEN 4 ELSE 5 END,
    dv.dimension_key`, saleListingID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var score PropertyQualityScore
		var reasons []byte
		var updatedAt time.Time
		if err := rows.Scan(&score.TargetType, &score.Dimension, &score.Value, &score.Confidence, &reasons, &updatedAt); err != nil {
			return err
		}
		score.Reasons = qualityScoreReasons(reasons)
		score.UpdatedAt = updatedAt.Format(time.RFC3339)
		listing.QualityScores = append(listing.QualityScores, score)
	}
	return rows.Err()
}

func qualityScoreReasons(data []byte) []string {
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if ok && strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
