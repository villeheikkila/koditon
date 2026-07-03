package properties

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
)

type TransactionMatchRunOptions struct {
	AutoLinkSafe     bool
	ScoreThreshold   int32
	CompetitorMargin int32
	TargetListingID  *uuid.UUID
}

type TransactionMatchRunSummary struct {
	RunID      string
	Candidates int32
	AutoLinked int32
	Ambiguous  int32
}

type transactionMatchCandidateRaw struct {
	ListingID            uuid.UUID
	TransactionID        uuid.UUID
	ListingLayout        string
	TransactionLayout    string
	ListingArea          *float64
	TransactionArea      float64
	ListingType          string
	TransactionType      string
	ListingBuildYear     *int32
	TransactionYear      int32
	ListingFloor         *int32
	ListingTotalFloors   *int32
	TransactionFloor     string
	ListingElevator      *bool
	TransactionElevator  bool
	ListingCondition     string
	TransactionCondition string
	ListingPlotOwned     *bool
	TransactionPlotOwned *bool
	ListingEnergy        string
	TransactionEnergy    string
	ListingPrice         *int64
	TransactionPrice     int32
	ListingFirstSeenAt   *time.Time
	ListingLastSeenAt    *time.Time
	ListingCreatedAt     time.Time
	ListingUpdatedAt     time.Time
	TransactionCreatedAt time.Time
}

type transactionMatchCandidate struct {
	transactionMatchCandidateRaw
	Score             int32
	Confidence        string
	PriceDeltaPercent *float64
	Reasons           []byte
	Status            string
}

func (s *Service) RunSaleListingTransactionMatch(ctx context.Context, opts TransactionMatchRunOptions) (TransactionMatchRunSummary, error) {
	if opts.ScoreThreshold <= 0 {
		opts.ScoreThreshold = 90
	}
	if opts.CompetitorMargin <= 0 {
		opts.CompetitorMargin = 15
	}
	runID, err := s.createTransactionMatchRun(ctx, opts)
	if err != nil {
		return TransactionMatchRunSummary{}, err
	}
	raw, err := s.loadTransactionMatchCandidateRows(ctx, opts.TargetListingID)
	if err != nil {
		return TransactionMatchRunSummary{}, err
	}
	candidates := make([]transactionMatchCandidate, 0, len(raw))
	for _, row := range raw {
		candidate, ok := scoreTransactionMatchCandidate(row, opts.ScoreThreshold)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	if err := s.insertTransactionMatchCandidates(ctx, runID, candidates); err != nil {
		return TransactionMatchRunSummary{}, err
	}
	var autoLinked int32
	var ambiguous int32
	if opts.AutoLinkSafe {
		autoLinked, ambiguous, err = s.applyTransactionMatchLinks(ctx, runID, candidates, opts.ScoreThreshold, opts.CompetitorMargin)
		if err != nil {
			return TransactionMatchRunSummary{}, err
		}
	}
	if err := s.finishTransactionMatchRun(ctx, runID, int32(len(candidates)), autoLinked, ambiguous); err != nil {
		return TransactionMatchRunSummary{}, err
	}
	return TransactionMatchRunSummary{RunID: runID.String(), Candidates: int32(len(candidates)), AutoLinked: autoLinked, Ambiguous: ambiguous}, nil
}

func (s *Service) createTransactionMatchRun(ctx context.Context, opts TransactionMatchRunOptions) (uuid.UUID, error) {
	mode := "dry_run"
	if opts.AutoLinkSafe {
		mode = "auto_link_safe"
	}
	runID, err := s.queries.CreateTransactionMatchRun(ctx, db.CreateTransactionMatchRunParams{Mode: &mode, ScoreThreshold: &opts.ScoreThreshold, CompetitorMargin: &opts.CompetitorMargin})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create transaction match run: %w", err)
	}
	return runID, nil
}

func (s *Service) loadTransactionMatchCandidateRows(ctx context.Context, targetListingID *uuid.UUID) ([]transactionMatchCandidateRaw, error) {
	rows, err := s.queries.LoadTransactionMatchCandidateRows(ctx, targetListingID)
	if err != nil {
		return nil, fmt.Errorf("query transaction match candidates: %w", err)
	}
	out := make([]transactionMatchCandidateRaw, 0, len(rows))
	for _, row := range rows {
		if row.SaleListingID == nil {
			continue
		}
		createdAt := time.Now().UTC()
		if row.SaleListingCreatedAt != nil {
			createdAt = *row.SaleListingCreatedAt
		}
		out = append(out, transactionMatchCandidateRaw{ListingID: *row.SaleListingID, TransactionID: row.PricesTransactionID, ListingLayout: valueOrEmpty(row.ListingLayout), TransactionLayout: valueOrEmpty(row.TransactionLayout), ListingArea: row.SaleListingAreaValue, TransactionArea: row.PricesTransactionArea, ListingType: valueOrEmpty(row.ListingType), TransactionType: valueOrEmpty(row.TransactionType), ListingBuildYear: row.SaleListingBuildYear, TransactionYear: row.PricesTransactionBuildYear, ListingFloor: row.SaleListingFloorLevel, ListingTotalFloors: row.SaleListingTotalFloors, TransactionFloor: valueOrEmpty(row.TransactionFloor), ListingElevator: row.SaleListingElevator, TransactionElevator: row.PricesTransactionElevator, ListingCondition: valueOrEmpty(row.ListingCondition), TransactionCondition: valueOrEmpty(row.TransactionCondition), ListingPlotOwned: row.SaleListingPlotOwned, TransactionPlotOwned: row.PricesTransactionPlotOwned, ListingEnergy: valueOrEmpty(row.ListingEnergy), TransactionEnergy: valueOrEmpty(row.TransactionEnergy), ListingPrice: row.SaleListingAskingPrice, TransactionPrice: row.PricesTransactionPrice, ListingFirstSeenAt: row.SaleListingFirstSeenAt, ListingLastSeenAt: row.SaleListingLastSeenAt, ListingCreatedAt: createdAt, ListingUpdatedAt: row.SaleListingUpdatedAt, TransactionCreatedAt: row.PricesTransactionCreatedAt})
	}
	return out, nil
}

func scoreTransactionMatchCandidate(row transactionMatchCandidateRaw, threshold int32) (transactionMatchCandidate, bool) {
	layoutCode, layoutScore := transactionLayoutMatch(row.ListingLayout, row.TransactionLayout)
	transactionFloor, transactionTotalFloors, floorOK := parseTransactionFloor(row.TransactionFloor)
	if !floorOK || row.ListingFloor == nil || *row.ListingFloor != transactionFloor {
		return transactionMatchCandidate{}, false
	}
	if transactionTotalFloors != nil && (row.ListingTotalFloors == nil || *row.ListingTotalFloors != *transactionTotalFloors) {
		return transactionMatchCandidate{}, false
	}
	if row.ListingArea == nil || math.Abs(*row.ListingArea-row.TransactionArea) > 0.001 || layoutScore == 0 {
		return transactionMatchCandidate{}, false
	}
	if strings.TrimSpace(row.ListingType) == "" || strings.TrimSpace(row.ListingType) != transactionPropertyType(row.TransactionType) {
		return transactionMatchCandidate{}, false
	}
	if row.ListingBuildYear == nil || row.TransactionYear <= 0 || *row.ListingBuildYear != row.TransactionYear {
		return transactionMatchCandidate{}, false
	}
	if row.ListingElevator == nil || *row.ListingElevator != row.TransactionElevator {
		return transactionMatchCandidate{}, false
	}
	listingCondition := conditionMatchCode(row.ListingCondition)
	transactionCondition := conditionMatchCode(row.TransactionCondition)
	if listingCondition == "" || listingCondition != transactionCondition {
		return transactionMatchCandidate{}, false
	}
	if row.ListingPlotOwned == nil || row.TransactionPlotOwned == nil || *row.ListingPlotOwned != *row.TransactionPlotOwned {
		return transactionMatchCandidate{}, false
	}
	energyScore := int32(0)
	if row.ListingEnergy != "" && row.TransactionEnergy != "" && strings.EqualFold(row.ListingEnergy, row.TransactionEnergy) {
		energyScore = 8
	}
	temporalScore := transactionTemporalScore(row)
	score := int32(75) + layoutScore + energyScore + temporalScore
	priceDelta := transactionPriceDelta(row.ListingPrice, row.TransactionPrice)
	reasons, err := json.Marshal(map[string]any{"postal": "exact", "area": map[string]any{"listing": row.ListingArea, "transaction": row.TransactionArea}, "layout": map[string]any{"code": layoutCode, "listing": row.ListingLayout, "transaction": row.TransactionLayout}, "property_type": row.ListingType, "floor_level": map[string]any{"listing": row.ListingFloor, "transaction": transactionFloor}, "total_floors": map[string]any{"listing": row.ListingTotalFloors, "transaction": transactionTotalFloors}, "energy": map[string]any{"listing": row.ListingEnergy, "transaction": row.TransactionEnergy}, "build_year": map[string]any{"listing": row.ListingBuildYear, "transaction": row.TransactionYear}, "elevator": map[string]any{"listing": row.ListingElevator, "transaction": row.TransactionElevator}, "condition": map[string]any{"listing": listingCondition, "transaction": transactionCondition}, "plot_owned": map[string]any{"listing": row.ListingPlotOwned, "transaction": row.TransactionPlotOwned}, "transaction_created_at": row.TransactionCreatedAt, "listing_first_seen_at": row.ListingFirstSeenAt, "listing_last_seen_at": row.ListingLastSeenAt, "score": map[string]any{"postal": 15, "area": 15, "layout": layoutScore, "property_type": 10, "floor": "exact_gate", "total_floors": "exact_gate", "build_year": "exact_gate", "elevator": "exact_gate", "condition": "exact_gate", "plot_owned": "exact_gate", "energy": energyScore, "temporal": temporalScore}, "price_delta_percent": priceDelta})
	if err != nil {
		return transactionMatchCandidate{}, false
	}
	confidence := "low"
	if score >= threshold {
		confidence = "high"
	} else if score >= 90 {
		confidence = "medium"
	}
	return transactionMatchCandidate{transactionMatchCandidateRaw: row, Score: score, Confidence: confidence, PriceDeltaPercent: priceDelta, Reasons: reasons, Status: "candidate"}, true
}

func (s *Service) insertTransactionMatchCandidates(ctx context.Context, runID uuid.UUID, candidates []transactionMatchCandidate) error {
	for _, candidate := range candidates {
		if err := s.queries.InsertTransactionMatchCandidate(ctx, db.InsertTransactionMatchCandidateParams{RunID: &runID, ListingID: &candidate.ListingID, TransactionID: &candidate.TransactionID, Score: &candidate.Score, Confidence: &candidate.Confidence, Reasons: candidate.Reasons, PriceDeltaPercent: candidate.PriceDeltaPercent}); err != nil {
			return fmt.Errorf("insert transaction match candidate: %w", err)
		}
	}
	return nil
}

func (s *Service) applyTransactionMatchLinks(ctx context.Context, runID uuid.UUID, candidates []transactionMatchCandidate, threshold int32, margin int32) (int32, int32, error) {
	selected := selectTransactionMatches(candidates, threshold, margin)
	var autoLinked int32
	for _, candidate := range selected {
		rowsAffected, err := s.queries.ApplyTransactionMatchLink(ctx, db.ApplyTransactionMatchLinkParams{TransactionID: candidate.TransactionID, Score: candidate.Score, Reasons: candidate.Reasons, RunID: runID, ListingID: candidate.ListingID})
		if err != nil {
			return 0, 0, fmt.Errorf("link transaction match: %w", err)
		}
		if rowsAffected == nil || *rowsAffected == 0 {
			continue
		}
		autoLinked++
		if err := s.queries.MarkTransactionMatchLinked(ctx, db.MarkTransactionMatchLinkedParams{RunID: &runID, ListingID: &candidate.ListingID, TransactionID: &candidate.TransactionID}); err != nil {
			return 0, 0, fmt.Errorf("mark transaction match linked: %w", err)
		}
	}
	ambiguous, err := s.queries.MarkAmbiguousTransactionMatches(ctx, db.MarkAmbiguousTransactionMatchesParams{RunID: &runID, Threshold: &threshold})
	if err != nil {
		return 0, 0, fmt.Errorf("mark ambiguous transaction matches: %w", err)
	}
	return autoLinked, int32(ambiguous), nil
}

func (s *Service) finishTransactionMatchRun(ctx context.Context, runID uuid.UUID, candidates int32, autoLinked int32, ambiguous int32) error {
	if err := s.queries.FinishTransactionMatchRun(ctx, db.FinishTransactionMatchRunParams{Candidates: &candidates, AutoLinked: &autoLinked, Ambiguous: &ambiguous, RunID: &runID}); err != nil {
		return fmt.Errorf("finish transaction match run: %w", err)
	}
	return nil
}

func selectTransactionMatches(candidates []transactionMatchCandidate, threshold int32, margin int32) []transactionMatchCandidate {
	byListing := map[uuid.UUID][]transactionMatchCandidate{}
	byTransaction := map[uuid.UUID][]transactionMatchCandidate{}
	for _, candidate := range candidates {
		byListing[candidate.ListingID] = append(byListing[candidate.ListingID], candidate)
		byTransaction[candidate.TransactionID] = append(byTransaction[candidate.TransactionID], candidate)
	}
	sortCandidateGroups(byListing)
	sortCandidateGroups(byTransaction)
	selected := []transactionMatchCandidate{}
	for _, candidate := range candidates {
		if candidate.Score < threshold {
			continue
		}
		listingGroup := byListing[candidate.ListingID]
		transactionGroup := byTransaction[candidate.TransactionID]
		if len(listingGroup) == 0 || len(transactionGroup) == 0 || listingGroup[0].TransactionID != candidate.TransactionID || transactionGroup[0].ListingID != candidate.ListingID {
			continue
		}
		if len(listingGroup) > 1 && listingGroup[1].Score > candidate.Score-margin {
			continue
		}
		if len(transactionGroup) > 1 && transactionGroup[1].Score > candidate.Score-margin {
			continue
		}
		selected = append(selected, candidate)
	}
	return selected
}

func sortCandidateGroups(groups map[uuid.UUID][]transactionMatchCandidate) {
	for key := range groups {
		sort.Slice(groups[key], func(i, j int) bool {
			if groups[key][i].Score != groups[key][j].Score {
				return groups[key][i].Score > groups[key][j].Score
			}
			left := math.MaxFloat64
			right := math.MaxFloat64
			if groups[key][i].PriceDeltaPercent != nil {
				left = *groups[key][i].PriceDeltaPercent
			}
			if groups[key][j].PriceDeltaPercent != nil {
				right = *groups[key][j].PriceDeltaPercent
			}
			return left < right
		})
	}
}

func transactionLayoutMatch(listing string, transaction string) (string, int32) {
	listingExact := exactLayoutKey(listing)
	transactionExact := exactLayoutKey(regexp.MustCompile(`(\.\.\.|…).*$`).ReplaceAllString(transaction, ""))
	listingCompact := compactLayoutKey(listing)
	transactionCompact := compactLayoutKey(transaction)
	switch {
	case listingExact == "" || transactionExact == "":
		return "none", 0
	case listingExact == transactionExact:
		return "exact", 35
	case len(transactionExact) >= 2 && strings.HasPrefix(listingExact, transactionExact):
		return "exact_prefix", 32
	case len(listingExact) >= 2 && strings.HasPrefix(transactionExact, listingExact):
		return "exact_prefix", 32
	case listingCompact != "" && listingCompact == transactionCompact:
		return "space_insensitive", 29
	case len(transactionCompact) >= 2 && strings.HasPrefix(listingCompact, transactionCompact):
		return "compact_prefix", 24
	case len(listingCompact) >= 2 && strings.HasPrefix(transactionCompact, listingCompact):
		return "compact_prefix", 24
	default:
		return "none", 0
	}
}

func exactLayoutKey(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func compactLayoutKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "+", "", ",", "", ".", "", "-", "", "/", "")
	return replacer.Replace(exactLayoutKey(value))
}

func parseTransactionFloor(value string) (int32, *int32, bool) {
	parts := regexp.MustCompile(`^\s*(-?[0-9]+)(?:\s*/\s*([0-9]+))?\s*$`).FindStringSubmatch(value)
	if parts == nil {
		return 0, nil, false
	}
	level, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0, nil, false
	}
	if parts[2] == "" {
		return int32(level), nil, true
	}
	total, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		return 0, nil, false
	}
	total32 := int32(total)
	return int32(level), &total32, true
}

func transactionPropertyType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(normalized, "kerrostalo"):
		return "apartment_block"
	case strings.Contains(normalized, "rivitalo"):
		return "row_house"
	case strings.Contains(normalized, "paritalo"):
		return "semi_detached"
	case strings.Contains(normalized, "omakotitalo") || strings.Contains(normalized, "erillistalo"):
		return "detached_house"
	default:
		return normalized
	}
}

func conditionMatchCode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "hyvä") || strings.Contains(normalized, "good"):
		return "good"
	case strings.Contains(normalized, "tyyd") || strings.Contains(normalized, "satisfactory"):
		return "satisfactory"
	case strings.Contains(normalized, "huono") || strings.Contains(normalized, "poor"):
		return "poor"
	default:
		return normalized
	}
}

func transactionTemporalScore(row transactionMatchCandidateRaw) int32 {
	lastSeen := row.ListingUpdatedAt
	if row.ListingLastSeenAt != nil {
		lastSeen = *row.ListingLastSeenAt
	}
	firstSeen := row.ListingCreatedAt
	if row.ListingFirstSeenAt != nil {
		firstSeen = *row.ListingFirstSeenAt
	}
	if !row.TransactionCreatedAt.Before(lastSeen.AddDate(0, 0, -14)) && !row.TransactionCreatedAt.After(lastSeen.AddDate(0, 9, 0)) {
		return 18
	}
	if !row.TransactionCreatedAt.Before(firstSeen.AddDate(0, 0, -90)) && !row.TransactionCreatedAt.After(lastSeen.AddDate(2, 0, 0)) {
		return 8
	}
	return 0
}

func transactionPriceDelta(listingPrice *int64, transactionPrice int32) *float64 {
	if listingPrice == nil || transactionPrice <= 0 {
		return nil
	}
	delta := math.Abs(float64(*listingPrice)-float64(transactionPrice)) / float64(transactionPrice)
	return &delta
}
