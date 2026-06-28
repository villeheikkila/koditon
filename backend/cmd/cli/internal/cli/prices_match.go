package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/domain/properties"
)

type PricesMatchSaleListingsFlags struct {
	AutoLinkSafe     bool
	ScoreThreshold   int
	CompetitorMargin int
	JSON             bool
	Out              io.Writer
}

type pricesMatchSaleListingsSummary struct {
	RunID             string `json:"run_id"`
	Mode              string `json:"mode"`
	ScoreThreshold    int32  `json:"score_threshold"`
	CompetitorMargin  int32  `json:"competitor_margin"`
	Candidates        int32  `json:"candidates"`
	HighConfidence    int32  `json:"high_confidence"`
	MediumConfidence  int32  `json:"medium_confidence"`
	LowConfidence     int32  `json:"low_confidence"`
	AutoLinked        int32  `json:"auto_linked"`
	Ambiguous         int32  `json:"ambiguous"`
	StartedAt         string `json:"started_at"`
	FinishedAt        string `json:"finished_at"`
	TopCandidateScore *int32 `json:"top_candidate_score,omitempty"`
}

func RunPricesMatchSaleListings(ctx context.Context, pool *pgxpool.Pool, f PricesMatchSaleListingsFlags) error {
	out := resolveOutput(f.Out)
	threshold := f.ScoreThreshold
	if threshold == 0 {
		threshold = 90
	}
	margin := f.CompetitorMargin
	if margin == 0 {
		margin = 15
	}
	service := properties.NewService(pool)
	run, err := service.RunSaleListingTransactionMatch(ctx, properties.TransactionMatchRunOptions{AutoLinkSafe: f.AutoLinkSafe, ScoreThreshold: int32(threshold), CompetitorMargin: int32(margin)})
	if err != nil {
		return fmt.Errorf("match sale listings: %w", err)
	}
	summary, err := loadPricesMatchSaleListingsSummary(ctx, pool, run.RunID)
	if err != nil {
		return err
	}
	if f.JSON {
		return writeJSON(out, summary)
	}
	fmt.Fprintln(out, headerStyle.Render("Prices transaction matching"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, renderKeyValue("Run", summary.RunID))
	fmt.Fprintln(out, renderKeyValue("Mode", summary.Mode))
	fmt.Fprintln(out, renderKeyValue("Threshold", fmt.Sprintf("%d", summary.ScoreThreshold)))
	fmt.Fprintln(out, renderKeyValue("Margin", fmt.Sprintf("%d", summary.CompetitorMargin)))
	fmt.Fprintln(out)
	headers := []string{"Candidates", "High", "Medium", "Low", "Auto-linked", "Ambiguous", "Top score"}
	topScore := ""
	if summary.TopCandidateScore != nil {
		topScore = fmt.Sprintf("%d", *summary.TopCandidateScore)
	}
	fmt.Fprint(out, renderTable(headers, [][]string{{
		fmt.Sprintf("%d", summary.Candidates),
		fmt.Sprintf("%d", summary.HighConfidence),
		fmt.Sprintf("%d", summary.MediumConfidence),
		fmt.Sprintf("%d", summary.LowConfidence),
		fmt.Sprintf("%d", summary.AutoLinked),
		fmt.Sprintf("%d", summary.Ambiguous),
		topScore,
	}}))
	return nil
}

func loadPricesMatchSaleListingsSummary(ctx context.Context, pool *pgxpool.Pool, runID string) (pricesMatchSaleListingsSummary, error) {
	const query = `
SELECT
    r.sale_listing_prices_transaction_match_run_id::text,
    r.sale_listing_prices_transaction_match_run_mode,
    r.sale_listing_prices_transaction_match_score_threshold,
    r.sale_listing_prices_transaction_match_competitor_margin,
    r.sale_listing_prices_transaction_match_candidates_count,
    count(*) FILTER (WHERE c.sale_listing_prices_transaction_match_confidence = 'high')::integer,
    count(*) FILTER (WHERE c.sale_listing_prices_transaction_match_confidence = 'medium')::integer,
    count(*) FILTER (WHERE c.sale_listing_prices_transaction_match_confidence = 'low')::integer,
    r.sale_listing_prices_transaction_match_auto_linked_count,
    r.sale_listing_prices_transaction_match_ambiguous_count,
    r.sale_listing_prices_transaction_match_started_at::text,
    COALESCE(r.sale_listing_prices_transaction_match_finished_at::text, ''),
    max(c.sale_listing_prices_transaction_match_score)::integer
FROM public.sale_listing_prices_transaction_match_runs r
LEFT JOIN public.sale_listing_prices_transaction_match_candidates c ON c.sale_listing_prices_transaction_match_run_id = r.sale_listing_prices_transaction_match_run_id
WHERE r.sale_listing_prices_transaction_match_run_id = $1::uuid
GROUP BY r.sale_listing_prices_transaction_match_run_id`
	var summary pricesMatchSaleListingsSummary
	if err := pool.QueryRow(ctx, query, runID).Scan(
		&summary.RunID,
		&summary.Mode,
		&summary.ScoreThreshold,
		&summary.CompetitorMargin,
		&summary.Candidates,
		&summary.HighConfidence,
		&summary.MediumConfidence,
		&summary.LowConfidence,
		&summary.AutoLinked,
		&summary.Ambiguous,
		&summary.StartedAt,
		&summary.FinishedAt,
		&summary.TopCandidateScore,
	); err != nil {
		return pricesMatchSaleListingsSummary{}, fmt.Errorf("load match summary: %w", err)
	}
	return summary, nil
}
