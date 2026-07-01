package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"koditon/internal/db"
)

func (s *Service) TransactionMatchPostals(ctx context.Context, limit int32) ([]TransactionMatchPostalSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.queries.ListTransactionMatchPostals(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list transaction match postals: %w", err)
	}
	out := []TransactionMatchPostalSummary{}
	for _, row := range rows {
		postal := ""
		if row.Postal != nil {
			postal = *row.Postal
		}
		out = append(out, TransactionMatchPostalSummary{Postal: postal, NameFi: row.PostalNameFi, MunicipalityName: row.MunicipalityName, CandidateCount: row.CandidateCount, ListingCount: row.ListingCount, TransactionCount: row.TransactionCount, HighCount: row.HighCount, MediumCount: row.MediumCount, LowCount: row.LowCount, AmbiguousCount: row.AmbiguousCount, LatestAt: row.LatestAt})
	}
	return out, nil
}

func (s *Service) TransactionMatchCandidates(ctx context.Context, postal string, status string, transactionID string, limit int32) ([]TransactionMatchCandidate, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if status != "candidate" && status != "ambiguous" {
		status = ""
	}
	var parsedTransactionID *uuid.UUID
	if transactionID != "" {
		id, err := uuid.Parse(transactionID)
		if err != nil {
			return nil, fmt.Errorf("parse transaction id: %w", err)
		}
		parsedTransactionID = &id
	}
	rows, err := s.queries.ListTransactionMatchCandidates(ctx, db.ListTransactionMatchCandidatesParams{TransactionID: parsedTransactionID, Postal: emptyToNil(postal), Status: emptyToNil(status), LimitCount: limit})
	if err != nil {
		return nil, fmt.Errorf("list transaction match candidates: %w", err)
	}
	out := []TransactionMatchCandidate{}
	for _, row := range rows {
		item := TransactionMatchCandidate{
			ID:                row.ID,
			Status:            row.Status,
			LinkType:          row.LinkType,
			LinkMethod:        row.LinkMethod,
			Score:             row.LatestScore,
			Confidence:        row.Confidence,
			PriceDeltaPercent: row.PriceDeltaPercent,
			Reasons:           row.Reasons,
			CreatedAt:         row.CreatedAt,
			Listing: TransactionMatchListingCandidate{
				ID:                   row.ListingID,
				OfferingID:           row.OfferingID,
				CanonicalID:          row.SaleListingCanonicalID,
				SourceProvider:       row.SaleListingSourceProvider,
				NativeID:             row.SaleListingNativeID,
				URL:                  row.ListingUrl,
				ExternalURLAvailable: row.ExternalUrlAvailable,
				Headline:             row.ListingHeadline,
				StreetAddress:        row.ListingStreetAddress,
				City:                 row.ListingCity,
				Postal:               row.ListingPostal,
				RoomLayout:           row.ListingRoomLayout,
				Condition:            row.ListingCondition,
				ConditionMatchCode:   row.ListingConditionMatchCode,
				AreaM2:               row.SaleListingAreaValue,
				AskingPrice:          row.SaleListingAskingPrice,
				PricePerM2:           row.SaleListingPricePerM2,
				BuildYear:            row.SaleListingBuildYear,
				FloorLevel:           row.SaleListingFloorLevel,
				TotalFloors:          row.SaleListingTotalFloors,
				Elevator:             row.SaleListingElevator,
				EnergyMatchCode:      row.ListingEnergyMatchCode,
				EnergyLabel:          row.ListingEnergyLabel,
				PlotOwnershipRaw:     row.ListingPlotOwnershipRaw,
				PlotOwned:            row.SaleListingPlotOwned,
				FirstSeenAt:          row.ListingFirstSeenAt,
				LastSeenAt:           row.ListingLastSeenAt,
			},
			Transaction: TransactionMatchTransaction{
				ID:                  row.TransactionIDText,
				Description:         row.TransactionDescription,
				Type:                row.TransactionType,
				Category:            row.TransactionCategory,
				AreaM2:              row.PricesTransactionArea,
				Price:               int64(row.PricesTransactionPrice),
				PricePerSquareMeter: int64(row.PricesTransactionPricePerSquareMeter),
				BuildYear:           row.PricesTransactionBuildYear,
				Floor:               row.TransactionFloor,
				Elevator:            row.PricesTransactionElevator,
				Condition:           row.TransactionCondition,
				ConditionMatchCode:  row.TransactionConditionMatchCode,
				Plot:                row.TransactionPlot,
				PlotOwned:           row.PricesTransactionPlotOwned,
				EnergyClass:         row.TransactionEnergyClass,
				EnergyMatchCode:     row.TransactionEnergyMatchCode,
				PeriodIdentifier:    row.TransactionPeriodIdentifier,
				CreatedAt:           row.TransactionCreatedAt,
			},
		}
		item.Listing.Condition = displayCondition(item.Listing.Condition)
		item.Listing.EnergyLabel = displayEnergyClass(item.Listing.EnergyLabel, item.Listing.EnergyMatchCode)
		item.Transaction.Condition = displayCondition(item.Transaction.Condition)
		item.Transaction.EnergyClass = displayEnergyClass(item.Transaction.EnergyClass, item.Transaction.EnergyMatchCode)
		out = append(out, item)
	}
	return out, nil
}
