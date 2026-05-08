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
	row, err := s.queries.GetApartmentProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		return CanonicalProfileProjectionResult{}, err
	}
	return CanonicalProfileProjectionResult{SaleListingID: saleListingID.String(), ApartmentProfile: apartmentProfileFromRow(row)}, nil
}

func (s *Service) projectSaleListingCanonicalProfile(ctx context.Context, saleListingID uuid.UUID) error {
	if err := s.queries.EnsurePhysicalBuildingForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("ensure physical building: %w", err)
	}
	if err := s.queries.UpsertSaleListingProviderClaims(ctx, saleListingID); err != nil {
		return fmt.Errorf("upsert provider property claims: %w", err)
	}
	if err := s.queries.ProjectApartmentProfileForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project apartment profile: %w", err)
	}
	if err := s.queries.ProjectHousingCompanyRenovationsForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project housing company renovations: %w", err)
	}
	if err := s.queries.ProjectHousingCompanySystemsFromRenovationsForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project housing company systems from renovations: %w", err)
	}
	if err := s.queries.ProjectBuildingProfileForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project building profile: %w", err)
	}
	if err := s.queries.ProjectHousingCompanyProfileForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project housing company profile: %w", err)
	}
	if err := s.queries.ProjectQualityScoresForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project quality scores: %w", err)
	}
	return nil
}

func (s *Service) enrichSaleListingCanonicalApartmentProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row, err := s.queries.GetApartmentProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	listing.ApartmentProfile = apartmentProfileFromRow(row)
	return nil
}

func apartmentProfileFromRow(row db.ApartmentProfile) ApartmentProfile {
	return ApartmentProfile{
		HousingCompanyID:      ptrUUIDString(row.HousingCompanyID),
		PropertyUnitID:        row.PropertyUnitID.String(),
		AreaM2:                row.ApartmentProfileAreaM2,
		LivingAreaM2:          row.ApartmentProfileLivingAreaM2,
		RoomLayout:            valueOrEmpty(row.ApartmentProfileRoomLayout),
		RoomCount:             row.ApartmentProfileRoomCount,
		BedroomCount:          row.ApartmentProfileBedroomCount,
		FloorLevel:            row.ApartmentProfileFloorLevel,
		TotalFloors:           row.ApartmentProfileTotalFloors,
		KitchenType:           valueOrEmpty(row.ApartmentProfileKitchenType),
		LayoutQuality:         valueOrEmpty(row.ApartmentProfileLayoutQuality),
		AwkwardLayout:         row.ApartmentProfileAwkwardLayout,
		Condition:             valueOrEmpty(row.ApartmentProfileCondition),
		KitchenCondition:      valueOrEmpty(row.ApartmentProfileKitchenCondition),
		BathroomCondition:     valueOrEmpty(row.ApartmentProfileBathroomCondition),
		SurfaceRenovationNeed: row.ApartmentProfileSurfaceRenovationNeed,
		ModernizationNeed:     row.ApartmentProfileModernizationNeed,
		Sauna:                 row.ApartmentProfileSauna,
		Balcony:               row.ApartmentProfileBalcony,
		BalconyGlazing:        row.ApartmentProfileBalconyGlazing,
		ParkingType:           valueOrEmpty(row.ApartmentProfileParkingType),
		StorageQuality:        valueOrEmpty(row.ApartmentProfileStorageQuality),
		ViewQuality:           valueOrEmpty(row.ApartmentProfileViewQuality),
		NoiseRisk:             row.ApartmentProfileNoiseRisk,
		Accessibility:         valueOrEmpty(row.ApartmentProfileAccessibility),
		Confidence:            row.ApartmentProfileConfidence,
		UpdatedAt:             row.ApartmentProfileUpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) enrichSaleListingCanonicalBuildingProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row := s.db.QueryRow(ctx, `
WITH linked AS (
    SELECT pu.physical_building_id, pu.housing_company_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = $1
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC NULLS LAST, pos.property_offering_source_created_at DESC
    LIMIT 1
)
SELECT
    bp.physical_building_id,
    COALESCE(bp.housing_company_id, linked.housing_company_id),
    bp.building_profile_build_year,
    bp.building_profile_floor_count,
    bp.building_profile_apartment_count,
    COALESCE(bp.building_profile_energy_class, ''),
    COALESCE(bp.building_profile_heating_method, ''),
    COALESCE(bp.building_profile_material, ''),
    COALESCE(bp.building_profile_roof_type, ''),
    COALESCE(bp.building_profile_roof_material, ''),
    bp.building_profile_elevator,
    COALESCE(bp.building_profile_confidence, ''),
    bp.building_profile_updated_at,
    hcp.housing_company_id,
    COALESCE(hcp.housing_company_profile_name, ''),
    COALESCE(hcp.housing_company_profile_business_id, ''),
    hcp.housing_company_profile_build_year,
    hcp.housing_company_profile_apartment_count,
    COALESCE(hcp.housing_company_profile_plot_ownership_type, ''),
    COALESCE(hcp.housing_company_profile_energy_class, ''),
    COALESCE(hcp.housing_company_profile_maintenance_risk, ''),
    COALESCE(hcp.housing_company_profile_financial_risk, ''),
    COALESCE(hcp.housing_company_profile_repair_backlog_risk, ''),
    COALESCE(hcp.housing_company_profile_confidence, ''),
    hcp.housing_company_profile_updated_at
FROM linked
LEFT JOIN public.building_profiles bp ON bp.physical_building_id = linked.physical_building_id
LEFT JOIN public.housing_company_profiles hcp ON hcp.housing_company_id = linked.housing_company_id`, saleListingID)
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
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = $1
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC NULLS LAST, pos.property_offering_source_created_at DESC
    LIMIT 1
),
targets AS (
    SELECT 'property_offering'::text AS target_type, property_offering_id AS target_id FROM linked WHERE property_offering_id IS NOT NULL
    UNION ALL SELECT 'property_unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
    UNION ALL SELECT 'physical_building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
    UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
)
SELECT
    pqs.property_quality_score_target_type,
    pqs.property_quality_score_dimension,
    pqs.property_quality_score_value,
    pqs.property_quality_score_confidence,
    pqs.property_quality_score_reasons,
    pqs.property_quality_score_updated_at
FROM targets
JOIN public.property_quality_scores pqs
    ON pqs.property_quality_score_target_type = targets.target_type
    AND pqs.property_quality_score_target_id = targets.target_id
ORDER BY
    CASE pqs.property_quality_score_target_type WHEN 'property_offering' THEN 1 WHEN 'property_unit' THEN 2 WHEN 'physical_building' THEN 3 WHEN 'housing_company' THEN 4 ELSE 5 END,
    pqs.property_quality_score_dimension`, saleListingID)
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
