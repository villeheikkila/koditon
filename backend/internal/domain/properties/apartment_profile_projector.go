package properties

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
)

func (s *Service) ProjectSaleListingApartmentProfile(ctx context.Context, input string) (ApartmentProfileProjectionResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return ApartmentProfileProjectionResult{}, ErrNotFound
	}
	_, saleListingID, err := s.saleOfferingSource(ctx, offeringID)
	if err != nil {
		return ApartmentProfileProjectionResult{}, err
	}
	if err := s.projectSaleListingApartmentProfile(ctx, saleListingID); err != nil {
		return ApartmentProfileProjectionResult{}, err
	}
	row, err := s.queries.GetSaleListingApartmentProfile(ctx, saleListingID)
	if err != nil {
		return ApartmentProfileProjectionResult{}, err
	}
	return ApartmentProfileProjectionResult{SaleListingID: saleListingID.String(), ApartmentProfile: apartmentProfileFromRow(row)}, nil
}

func (s *Service) projectSaleListingApartmentProfile(ctx context.Context, saleListingID uuid.UUID) error {
	if err := s.queries.ProjectSaleListingApartmentProfile(ctx, saleListingID); err != nil {
		return fmt.Errorf("project sale listing apartment profile: %w", err)
	}
	if err := s.queries.UpsertSaleListingApartmentProfileProviderFieldSources(ctx, saleListingID); err != nil {
		return fmt.Errorf("upsert apartment profile provider field sources: %w", err)
	}
	if err := s.queries.ProjectHousingCompanyRenovationsForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project housing company renovations: %w", err)
	}
	if err := s.queries.ProjectHousingCompanySystemsFromRenovationsForSaleListing(ctx, saleListingID); err != nil {
		return fmt.Errorf("project housing company systems from renovations: %w", err)
	}
	if err := s.queries.ProjectSaleListingApartmentProfileLLMFacts(ctx, saleListingID); err != nil {
		return fmt.Errorf("project llm valuation facts to apartment profile: %w", err)
	}
	if err := s.queries.UpsertSaleListingApartmentProfileLLMFieldSources(ctx, saleListingID); err != nil {
		return fmt.Errorf("upsert llm apartment profile field sources: %w", err)
	}
	return nil
}

func (s *Service) enrichSaleListingApartmentProfile(ctx context.Context, listing *SaleListing, saleListingID uuid.UUID) error {
	row, err := s.queries.GetSaleListingApartmentProfile(ctx, saleListingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	listing.ApartmentProfile = apartmentProfileFromRow(row)
	return nil
}

func apartmentProfileFromRow(row db.SaleListingApartmentProfile) ApartmentProfile {
	return ApartmentProfile{
		HousingCompanyID:      ptrUUIDString(row.HousingCompanyID),
		PropertyUnitID:        ptrUUIDString(row.PropertyUnitID),
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
