package properties

import (
	"testing"
	"time"
)

func TestScoreTransactionMatchCandidateRequiresExactFloor(t *testing.T) {
	row := validTransactionMatchCandidateRaw()
	row.TransactionFloor = "4/6"
	if _, ok := scoreTransactionMatchCandidate(row, 90); ok {
		t.Fatal("expected floor mismatch to reject candidate")
	}
}

func TestScoreTransactionMatchCandidateRequiresKnownTransactionFloor(t *testing.T) {
	row := validTransactionMatchCandidateRaw()
	row.TransactionFloor = ""
	if _, ok := scoreTransactionMatchCandidate(row, 90); ok {
		t.Fatal("expected missing transaction floor to reject candidate")
	}
}

func TestScoreTransactionMatchCandidateAcceptsExactFloor(t *testing.T) {
	row := validTransactionMatchCandidateRaw()
	candidate, ok := scoreTransactionMatchCandidate(row, 90)
	if !ok {
		t.Fatal("expected exact floor candidate")
	}
	if candidate.Score < 90 {
		t.Fatalf("expected high enough score, got %d", candidate.Score)
	}
}

func validTransactionMatchCandidateRaw() transactionMatchCandidateRaw {
	area := 54.5
	buildYear := int32(1985)
	floor := int32(3)
	totalFloors := int32(6)
	elevator := true
	plotOwned := true
	price := int64(185000)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return transactionMatchCandidateRaw{
		ListingLayout:        "2h+k",
		TransactionLayout:    "2h+k",
		ListingArea:          &area,
		TransactionArea:      area,
		ListingType:          "apartment_block",
		TransactionType:      "Kerrostalo",
		ListingBuildYear:     &buildYear,
		TransactionYear:      buildYear,
		ListingFloor:         &floor,
		ListingTotalFloors:   &totalFloors,
		TransactionFloor:     "3/6",
		ListingElevator:      &elevator,
		TransactionElevator:  elevator,
		ListingCondition:     "Hyvä",
		TransactionCondition: "Hyvä",
		ListingPlotOwned:     &plotOwned,
		TransactionPlotOwned: &plotOwned,
		ListingEnergy:        "C",
		TransactionEnergy:    "C",
		ListingPrice:         &price,
		TransactionPrice:     180000,
		ListingCreatedAt:     now.AddDate(0, -1, 0),
		ListingUpdatedAt:     now,
		TransactionCreatedAt: now.AddDate(0, 1, 0),
	}
}
