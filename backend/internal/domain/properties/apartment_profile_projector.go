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
	row, err := s.queries.GetDimensionApartmentProfileForSaleListing(ctx, &saleListingID)
	if err != nil {
		return ApartmentProfile{}, err
	}
	return apartmentProfileFromDB(row), nil
}

func (s *Service) enrichSaleListingCanonicalBuildingProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row, err := s.queries.GetCanonicalBuildingProfileForSaleListing(ctx, &saleListingID)
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
	rows, err := s.queries.ListSaleListingQualityScores(ctx, &saleListingID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		listing.QualityScores = append(listing.QualityScores, qualityScoreFromDB(row))
	}
	return nil
}

func apartmentProfileFromDB(row db.GetDimensionApartmentProfileForSaleListingRow) ApartmentProfile {
	return ApartmentProfile{HousingCompanyID: row.HousingCompanyID.String(), PropertyUnitID: row.PropertyUnitID.String(), AreaM2: row.AreaM2, LivingAreaM2: row.LivingAreaM2, RoomLayout: stringValue(row.RoomLayout), RoomCount: row.RoomCount, BedroomCount: row.BedroomCount, FloorLevel: row.FloorLevel, TotalFloors: row.TotalFloors, KitchenType: stringValue(row.KitchenType), LayoutQuality: stringValue(row.LayoutQuality), AwkwardLayout: row.AwkwardLayout, Condition: stringValue(row.Condition), KitchenCondition: stringValue(row.KitchenCondition), BathroomCondition: stringValue(row.BathroomCondition), SurfaceRenovationNeed: row.SurfaceRenovationNeed, ModernizationNeed: row.ModernizationNeed, Sauna: row.Sauna, Balcony: row.Balcony, BalconyGlazing: row.BalconyGlazing, ParkingType: stringValue(row.ParkingType), StorageQuality: stringValue(row.StorageQuality), ViewQuality: stringValue(row.ViewQuality), NoiseRisk: row.NoiseRisk, Accessibility: stringValue(row.Accessibility), MaintenanceChargeMonthly: row.MaintenanceChargeMonthly, CapitalChargeMonthly: row.CapitalChargeMonthly, TotalChargeMonthly: row.TotalChargeMonthly, DebtShareEUR: row.DebtShareEur, ShareholderLiability: stringValue(row.ShareholderLiability), Confidence: stringValue(row.Confidence), UpdatedAt: row.ResolvedAt.Format(time.RFC3339)}
}

func buildingProfilesFromDB(row db.GetCanonicalBuildingProfileForSaleListingRow) (BuildingProfile, HousingCompanyProfile) {
	profile := BuildingProfile{PhysicalBuildingID: ptrUUIDString(row.PhysicalBuildingID), HousingCompanyID: row.BuildingHousingCompanyID.String(), BuildYear: row.BuildYear, FloorCount: row.FloorCount, ApartmentCount: row.ApartmentCount, EnergyClass: stringValue(row.EnergyClass), HeatingMethod: stringValue(row.HeatingMethod), Material: stringValue(row.Material), RoofType: stringValue(row.RoofType), RoofMaterial: stringValue(row.RoofMaterial), Elevator: row.Elevator, Confidence: stringValue(row.BuildingConfidence), UpdatedAt: row.BuildingResolvedAt.Format(time.RFC3339)}
	housingProfile := HousingCompanyProfile{HousingCompanyID: row.HousingCompanyID.String(), Name: stringValue(row.HousingCompanyName), BusinessID: stringValue(row.BusinessID), BuildYear: row.HousingCompanyBuildYear, ApartmentCount: row.HousingCompanyApartmentCount, PlotOwnershipType: stringValue(row.PlotOwnershipType), EnergyClass: stringValue(row.HousingCompanyEnergyClass), MaintenanceRisk: stringValue(row.MaintenanceRisk), FinancialRisk: stringValue(row.FinancialRisk), RepairBacklogRisk: stringValue(row.RepairBacklogRisk), Confidence: stringValue(row.HousingCompanyConfidence), UpdatedAt: row.HousingCompanyResolvedAt.Format(time.RFC3339)}
	return profile, housingProfile
}

func qualityScoreFromDB(row db.ListSaleListingQualityScoresRow) PropertyQualityScore {
	return PropertyQualityScore{TargetType: stringValue(row.TargetType), Dimension: stringValue(row.Dimension), Value: int32Value(row.Value), Confidence: stringValue(row.Confidence), Reasons: qualityScoreReasons(row.Reasons), UpdatedAt: row.ResolvedAt.Format(time.RFC3339)}
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int32Value(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}
