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
		out = append(out, TransactionMatchPostalSummary{Postal: postal, NameFi: valueOrEmpty(row.PostalNameFi), MunicipalityName: valueOrEmpty(row.MunicipalityName), CandidateCount: int64Value(row.CandidateCount), ListingCount: int64Value(row.ListingCount), TransactionCount: int64Value(row.TransactionCount), HighCount: int64Value(row.HighCount), MediumCount: int64Value(row.MediumCount), LowCount: int64Value(row.LowCount), AmbiguousCount: int64Value(row.AmbiguousCount), LatestAt: valueOrEmpty(row.LatestAt)})
	}
	return out, nil
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
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
			ID:                valueOrEmpty(row.ID),
			Status:            valueOrEmpty(row.Status),
			LinkType:          valueOrEmpty(row.LinkType),
			LinkMethod:        valueOrEmpty(row.LinkMethod),
			Score:             ptrInt32Value(row.Score),
			Confidence:        valueOrEmpty(row.Confidence),
			PriceDeltaPercent: row.PriceDeltaPercent,
			Reasons:           row.Reasons,
			CreatedAt:         valueOrEmpty(row.CreatedAt),
			Listing: TransactionMatchListingCandidate{
				ID:                   valueOrEmpty(row.ListingID),
				OfferingID:           valueOrEmpty(row.OfferingID),
				CanonicalID:          row.SaleListingCanonicalID,
				SourceProvider:       row.SaleListingSourceProvider,
				NativeID:             row.SaleListingNativeID,
				URL:                  valueOrEmpty(row.ListingUrl),
				ExternalURLAvailable: boolPtrValue(row.ExternalUrlAvailable),
				Headline:             valueOrEmpty(row.ListingHeadline),
				StreetAddress:        valueOrEmpty(row.ListingStreetAddress),
				City:                 valueOrEmpty(row.ListingCity),
				Postal:               valueOrEmpty(row.ListingPostal),
				RoomLayout:           valueOrEmpty(row.ListingRoomLayout),
				Condition:            valueOrEmpty(row.ListingCondition),
				ConditionMatchCode:   valueOrEmpty(row.ListingConditionMatchCode),
				AreaM2:               row.SaleListingAreaValue,
				AskingPrice:          row.SaleListingAskingPrice,
				PricePerM2:           row.SaleListingPricePerM2,
				BuildYear:            row.SaleListingBuildYear,
				FloorLevel:           row.SaleListingFloorLevel,
				TotalFloors:          row.SaleListingTotalFloors,
				Elevator:             row.SaleListingElevator,
				EnergyMatchCode:      valueOrEmpty(row.ListingEnergyMatchCode),
				EnergyLabel:          valueOrEmpty(row.ListingEnergyLabel),
				PlotOwnershipRaw:     valueOrEmpty(row.ListingPlotOwnershipRaw),
				PlotOwned:            row.SaleListingPlotOwned,
				FirstSeenAt:          valueOrEmpty(row.ListingFirstSeenAt),
				LastSeenAt:           valueOrEmpty(row.ListingLastSeenAt),
			},
			Transaction: TransactionMatchTransaction{
				ID:                  valueOrEmpty(row.TransactionIDText),
				Description:         valueOrEmpty(row.TransactionDescription),
				Type:                valueOrEmpty(row.TransactionType),
				Category:            valueOrEmpty(row.TransactionCategory),
				AreaM2:              row.PricesTransactionArea,
				Price:               int64(row.PricesTransactionPrice),
				PricePerSquareMeter: int64(row.PricesTransactionPricePerSquareMeter),
				BuildYear:           row.PricesTransactionBuildYear,
				Floor:               valueOrEmpty(row.TransactionFloor),
				Elevator:            row.PricesTransactionElevator,
				Condition:           valueOrEmpty(row.TransactionCondition),
				ConditionMatchCode:  valueOrEmpty(row.TransactionConditionMatchCode),
				Plot:                valueOrEmpty(row.TransactionPlot),
				PlotOwned:           row.PricesTransactionPlotOwned,
				EnergyClass:         valueOrEmpty(row.TransactionEnergyClass),
				EnergyMatchCode:     valueOrEmpty(row.TransactionEnergyMatchCode),
				PeriodIdentifier:    valueOrEmpty(row.TransactionPeriodIdentifier),
				CreatedAt:           valueOrEmpty(row.TransactionCreatedAt),
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
