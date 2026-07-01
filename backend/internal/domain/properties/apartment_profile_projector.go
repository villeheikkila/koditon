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
	row, err := s.queries.GetDimensionApartmentProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		return ApartmentProfile{}, err
	}
	return apartmentProfileFromDB(row), nil
}

func (s *Service) enrichSaleListingCanonicalBuildingProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row, err := s.queries.GetCanonicalBuildingProfileForSaleListing(ctx, saleListingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	profile, housingProfile := buildingProfilesFromDB(row)
	listing.BuildingProfile = profile
	listing.HousingProfile = housingProfile
	return nil
}

func (s *Service) enrichSaleListingQualityScores(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	rows, err := s.queries.ListSaleListingQualityScores(ctx, saleListingID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.QualityScores = append(listing.QualityScores, qualityScoreFromDB(row))
	}
	return nil
}

func apartmentProfileFromDB(row db.GetDimensionApartmentProfileForSaleListingRow) ApartmentProfile {
	return ApartmentProfile{HousingCompanyID: row.HousingCompanyID.String(), PropertyUnitID: row.PropertyUnitID.String(), AreaM2: &row.AreaM2, LivingAreaM2: &row.LivingAreaM2, RoomLayout: row.RoomLayout, RoomCount: &row.RoomCount, BedroomCount: &row.BedroomCount, FloorLevel: &row.FloorLevel, TotalFloors: &row.TotalFloors, KitchenType: row.KitchenType, LayoutQuality: row.LayoutQuality, AwkwardLayout: &row.AwkwardLayout, Condition: row.Condition, KitchenCondition: row.KitchenCondition, BathroomCondition: row.BathroomCondition, SurfaceRenovationNeed: &row.SurfaceRenovationNeed, ModernizationNeed: &row.ModernizationNeed, Sauna: &row.Sauna, Balcony: &row.Balcony, BalconyGlazing: &row.BalconyGlazing, ParkingType: row.ParkingType, StorageQuality: row.StorageQuality, ViewQuality: row.ViewQuality, NoiseRisk: &row.NoiseRisk, Accessibility: row.Accessibility, MaintenanceChargeMonthly: &row.MaintenanceChargeMonthly, CapitalChargeMonthly: &row.CapitalChargeMonthly, TotalChargeMonthly: &row.TotalChargeMonthly, DebtShareEUR: &row.DebtShareEur, ShareholderLiability: row.ShareholderLiability, Confidence: row.Confidence, UpdatedAt: row.ResolvedAt.Format(time.RFC3339)}
}

func buildingProfilesFromDB(row db.GetCanonicalBuildingProfileForSaleListingRow) (BuildingProfile, HousingCompanyProfile) {
	profile := BuildingProfile{PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: row.BuildingHousingCompanyID.String(), BuildYear: &row.BuildYear, FloorCount: &row.FloorCount, ApartmentCount: &row.ApartmentCount, EnergyClass: row.EnergyClass, HeatingMethod: row.HeatingMethod, Material: row.Material, RoofType: row.RoofType, RoofMaterial: row.RoofMaterial, Elevator: &row.Elevator, Confidence: row.BuildingConfidence}
	if row.BuildingResolvedAt != nil {
		profile.UpdatedAt = row.BuildingResolvedAt.Format(time.RFC3339)
	}
	housingProfile := HousingCompanyProfile{HousingCompanyID: row.HousingCompanyID.String(), Name: row.HousingCompanyName, BusinessID: row.BusinessID, BuildYear: row.HousingCompanyBuildYear, ApartmentCount: &row.HousingCompanyApartmentCount, PlotOwnershipType: row.PlotOwnershipType, EnergyClass: row.HousingCompanyEnergyClass, MaintenanceRisk: row.MaintenanceRisk, FinancialRisk: row.FinancialRisk, RepairBacklogRisk: row.RepairBacklogRisk, Confidence: row.HousingCompanyConfidence}
	if row.HousingCompanyResolvedAt != nil {
		housingProfile.UpdatedAt = row.HousingCompanyResolvedAt.Format(time.RFC3339)
	}
	return profile, housingProfile
}

func qualityScoreFromDB(row db.ListSaleListingQualityScoresRow) PropertyQualityScore {
	return PropertyQualityScore{TargetType: row.TargetType, Dimension: row.Dimension, Value: row.Value, Confidence: row.Confidence, Reasons: qualityScoreReasons(row.Reasons), UpdatedAt: row.ResolvedAt.Format(time.RFC3339)}
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
