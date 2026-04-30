package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pricesMatchReviewStartMsg struct {
	run   pricesMatchRunSummary
	group pricesMatchCandidateGroup
	err   error
}

type pricesMatchReviewGroupMsg struct {
	group pricesMatchCandidateGroup
	err   error
}

type pricesMatchReviewLinkMsg struct {
	group pricesMatchCandidateGroup
	err   error
}

type pricesMatchReviewRejectMsg struct {
	group pricesMatchCandidateGroup
	err   error
}

type pricesMatchRunSummary struct {
	ID         string
	Candidates int32
	AutoLinked int32
	Ambiguous  int32
}

type pricesMatchCandidateGroup struct {
	RunID      string
	Offset     int
	Total      int
	Listing    pricesMatchListing
	Candidates []pricesMatchCandidate
}

type pricesMatchListing struct {
	ID       string
	PublicID string
	Source   string
	Headline string
	Address  string
	City     string
	Postal   string
	Price    string
	Area     string
	Layout   string
	Seen     string
}

type pricesMatchCandidate struct {
	CandidateID   string
	TransactionID string
	Score         int32
	PriceDelta    string
	Description   string
	Type          string
	Area          string
	Price         string
	Floor         string
	Energy        string
	Year          string
	Period        string
	Neighborhood  string
}

type pricesMatchReviewScreen struct {
	ctx        *appContext
	breadcrumb string
	table      table.Model
	run        pricesMatchRunSummary
	group      pricesMatchCandidateGroup
	loading    bool
	linking    bool
	errorText  string
	infoText   string
	width      int
	height     int
}

func newPricesMatchReviewScreen(ctx *appContext, breadcrumb string) Screen {
	cols := []table.Column{{Title: "Score", Width: 6}, {Title: "Delta", Width: 7}, {Title: "Description", Width: 24}, {Title: "Type", Width: 7}, {Title: "Area", Width: 7}, {Title: "Price", Width: 10}, {Title: "Floor", Width: 7}, {Title: "Energy", Width: 8}, {Title: "Year", Width: 6}, {Title: "Period", Width: 8}, {Title: "Neighborhood", Width: 18}}
	t := table.New(table.WithColumns(cols), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(jobTableStyles())
	return &pricesMatchReviewScreen{ctx: ctx, breadcrumb: breadcrumb, table: t, loading: true}
}

func (s *pricesMatchReviewScreen) Key() string { return "prices-match-review" }

func (s *pricesMatchReviewScreen) Init() tea.Cmd {
	if s.ctx.dbPool == nil {
		return func() tea.Msg { return pricesMatchReviewStartMsg{err: fmt.Errorf("database pool unavailable")} }
	}
	return startPricesMatchReviewCmd(s.ctx.dbPool)
}

func (s *pricesMatchReviewScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.table.SetWidth(max(88, width-12))
	s.table.SetHeight(max(8, height-22))
}

func (s *pricesMatchReviewScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	switch typed := msg.(type) {
	case pricesMatchReviewStartMsg:
		s.loading = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.run = typed.run
		s.setGroup(typed.group)
		s.infoText = fmt.Sprintf("auto-linked=%d ambiguous=%d candidates=%d", s.run.AutoLinked, s.run.Ambiguous, s.run.Candidates)
		return nil
	case pricesMatchReviewGroupMsg:
		s.loading = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.setGroup(typed.group)
		return nil
	case pricesMatchReviewLinkMsg:
		s.linking = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.errorText = ""
		s.infoText = "linked selected transaction"
		s.setGroup(typed.group)
		return nil
	case pricesMatchReviewRejectMsg:
		s.linking = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.errorText = ""
		s.infoText = "rejected candidates for listing"
		s.setGroup(typed.group)
		return nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "esc", "left", "h", "backspace":
			nav.Pop()
			return nil
		case "r":
			if s.loading || s.linking {
				return nil
			}
			s.loading = true
			s.errorText = ""
			return startPricesMatchReviewCmd(s.ctx.dbPool)
		case "n", "s":
			if s.loading || s.linking || !s.hasGroup() {
				return nil
			}
			s.loading = true
			return loadPricesMatchCandidateGroupCmd(s.ctx.dbPool, s.group.RunID, min(s.group.Offset+1, max(0, s.group.Total-1)))
		case "p":
			if s.loading || s.linking || !s.hasGroup() {
				return nil
			}
			s.loading = true
			return loadPricesMatchCandidateGroupCmd(s.ctx.dbPool, s.group.RunID, max(0, s.group.Offset-1))
		case "enter":
			if s.loading || s.linking || !s.hasGroup() || len(s.group.Candidates) == 0 {
				return nil
			}
			idx := safeIndex(s.table.Cursor(), len(s.group.Candidates))
			s.linking = true
			s.errorText = ""
			return linkPricesMatchCandidateCmd(s.ctx.dbPool, s.group, s.group.Candidates[idx])
		case "x":
			if s.loading || s.linking || !s.hasGroup() {
				return nil
			}
			s.linking = true
			s.errorText = ""
			return rejectPricesMatchListingCmd(s.ctx.dbPool, s.group)
		}
	}
	if s.loading || s.linking {
		return nil
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *pricesMatchReviewScreen) View() string {
	var b strings.Builder
	b.WriteString(s.ctx.styles.progressLabel.Render("Prices transaction match review"))
	b.WriteString("\n")
	if s.run.ID != "" {
		b.WriteString(s.ctx.styles.muted.Render(fmt.Sprintf("run=%s • %s", s.run.ID, s.infoText)))
		b.WriteString("\n")
	}
	if s.loading {
		b.WriteString("\n")
		b.WriteString(s.ctx.styles.muted.Render("Running safe auto-link and loading review candidates..."))
		return s.ctx.styles.panel.Width(max(88, s.width-8)).Render(b.String())
	}
	if s.errorText != "" {
		b.WriteString("\n")
		b.WriteString(s.ctx.styles.error.Render(s.errorText))
		return s.ctx.styles.panel.Width(max(88, s.width-8)).Render(b.String())
	}
	if !s.hasGroup() {
		b.WriteString("\n")
		b.WriteString(s.ctx.styles.success.Render("No ambiguous matches left for this run."))
		b.WriteString("\n\n")
		b.WriteString(s.ctx.styles.muted.Render("Press r to run matching again or Esc to go back."))
		return s.ctx.styles.panel.Width(max(88, s.width-8)).Render(b.String())
	}
	listing := s.group.Listing
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.selected.Render(fmt.Sprintf("Listing %d/%d", s.group.Offset+1, s.group.Total)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s • %s • %s %s • %s m2 • %s • %s\n", listing.Source, listing.PublicID, listing.Postal, listing.City, listing.Area, listing.Price, listing.Seen))
	b.WriteString(s.ctx.styles.description.Render(firstNonEmpty(listing.Headline, listing.Address)))
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.muted.Render(fmt.Sprintf("layout=%s • address=%s", listing.Layout, listing.Address)))
	b.WriteString("\n\n")
	if s.linking {
		b.WriteString(s.ctx.styles.running.Render("Applying selection..."))
	} else {
		b.WriteString(s.table.View())
	}
	b.WriteString("\n\n")
	b.WriteString(s.ctx.styles.muted.Render("Enter link selected • x none match • j/k move • n or s skip • p previous • r rerun auto-link • Esc back"))
	return s.ctx.styles.panel.Width(max(88, s.width-8)).Render(b.String())
}

func (s *pricesMatchReviewScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: "Enter link • x reject • n/s skip • p previous • r rerun • Esc back • q quit"}
}

func (s *pricesMatchReviewScreen) setGroup(group pricesMatchCandidateGroup) {
	s.group = group
	s.table.SetRows(buildPricesMatchCandidateRows(group.Candidates))
	s.table.SetCursor(0)
}

func (s *pricesMatchReviewScreen) hasGroup() bool {
	return s.group.RunID != "" && s.group.Listing.ID != ""
}

func startPricesMatchReviewCmd(pool *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var runID string
		if err := pool.QueryRow(ctx, `SELECT public.fnc__refresh_sale_listing_prices_transaction_matches(true, 90, 15)::text`).Scan(&runID); err != nil {
			return pricesMatchReviewStartMsg{err: fmt.Errorf("run matching: %w", err)}
		}
		run, err := loadPricesMatchRunSummary(ctx, pool, runID)
		if err != nil {
			return pricesMatchReviewStartMsg{err: err}
		}
		group, err := loadPricesMatchCandidateGroup(ctx, pool, runID, 0)
		return pricesMatchReviewStartMsg{run: run, group: group, err: err}
	}
}

func loadPricesMatchCandidateGroupCmd(pool *pgxpool.Pool, runID string, offset int) tea.Cmd {
	return func() tea.Msg {
		group, err := loadPricesMatchCandidateGroup(context.Background(), pool, runID, offset)
		return pricesMatchReviewGroupMsg{group: group, err: err}
	}
}

func linkPricesMatchCandidateCmd(pool *pgxpool.Pool, group pricesMatchCandidateGroup, candidate pricesMatchCandidate) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			return pricesMatchReviewLinkMsg{err: fmt.Errorf("begin link transaction: %w", err)}
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `SELECT public.fnc__link_sale_listing_prices_transaction($1, $2::uuid)`, group.Listing.PublicID, candidate.TransactionID); err != nil {
			return pricesMatchReviewLinkMsg{err: fmt.Errorf("link transaction: %w", err)}
		}
		if _, err := tx.Exec(ctx, `UPDATE public.sale_listing_prices_transaction_match_candidates SET sale_listing_prices_transaction_match_status = CASE WHEN sale_listing_prices_transaction_match_candidate_id = $3::uuid THEN 'auto_linked' ELSE 'rejected' END WHERE sale_listing_prices_transaction_match_run_id = $1::uuid AND sale_listing_id = $2::uuid`, group.RunID, group.Listing.ID, candidate.CandidateID); err != nil {
			return pricesMatchReviewLinkMsg{err: fmt.Errorf("mark candidates: %w", err)}
		}
		if _, err := tx.Exec(ctx, `UPDATE public.sale_listings SET sale_listing_prices_match_status = 'manual_linked', sale_listing_prices_match_run_id = $2::uuid, sale_listing_updated_at = now() WHERE sale_listing_id = $1::uuid`, group.Listing.ID, group.RunID); err != nil {
			return pricesMatchReviewLinkMsg{err: fmt.Errorf("mark listing: %w", err)}
		}
		if err := tx.Commit(ctx); err != nil {
			return pricesMatchReviewLinkMsg{err: fmt.Errorf("commit link transaction: %w", err)}
		}
		nextOffset := group.Offset
		if group.Total > 0 && nextOffset >= group.Total-1 {
			nextOffset = max(0, group.Total-2)
		}
		next, err := loadPricesMatchCandidateGroup(ctx, pool, group.RunID, nextOffset)
		return pricesMatchReviewLinkMsg{group: next, err: err}
	}
}

func rejectPricesMatchListingCmd(pool *pgxpool.Pool, group pricesMatchCandidateGroup) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			return pricesMatchReviewRejectMsg{err: fmt.Errorf("begin reject transaction: %w", err)}
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `UPDATE public.sale_listing_prices_transaction_match_candidates SET sale_listing_prices_transaction_match_status = 'rejected' WHERE sale_listing_prices_transaction_match_run_id = $1::uuid AND sale_listing_id = $2::uuid AND sale_listing_prices_transaction_match_status = 'ambiguous'`, group.RunID, group.Listing.ID); err != nil {
			return pricesMatchReviewRejectMsg{err: fmt.Errorf("reject candidates: %w", err)}
		}
		if _, err := tx.Exec(ctx, `UPDATE public.sale_listings SET sale_listing_prices_match_status = 'rejected', sale_listing_prices_match_run_id = $2::uuid, sale_listing_updated_at = now() WHERE sale_listing_id = $1::uuid`, group.Listing.ID, group.RunID); err != nil {
			return pricesMatchReviewRejectMsg{err: fmt.Errorf("mark listing rejected: %w", err)}
		}
		if err := tx.Commit(ctx); err != nil {
			return pricesMatchReviewRejectMsg{err: fmt.Errorf("commit reject transaction: %w", err)}
		}
		nextOffset := group.Offset
		if group.Total > 0 && nextOffset >= group.Total-1 {
			nextOffset = max(0, group.Total-2)
		}
		next, err := loadPricesMatchCandidateGroup(ctx, pool, group.RunID, nextOffset)
		return pricesMatchReviewRejectMsg{group: next, err: err}
	}
}

func loadPricesMatchRunSummary(ctx context.Context, pool *pgxpool.Pool, runID string) (pricesMatchRunSummary, error) {
	var run pricesMatchRunSummary
	if err := pool.QueryRow(ctx, `SELECT sale_listing_prices_transaction_match_run_id::text, sale_listing_prices_transaction_match_candidates_count, sale_listing_prices_transaction_match_auto_linked_count, sale_listing_prices_transaction_match_ambiguous_count FROM public.sale_listing_prices_transaction_match_runs WHERE sale_listing_prices_transaction_match_run_id = $1::uuid`, runID).Scan(&run.ID, &run.Candidates, &run.AutoLinked, &run.Ambiguous); err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("load match run: %w", err)
	}
	return run, nil
}

func loadPricesMatchCandidateGroup(ctx context.Context, pool *pgxpool.Pool, runID string, offset int) (pricesMatchCandidateGroup, error) {
	const groupQuery = `
WITH groups AS (
    SELECT c.sale_listing_id, max(c.sale_listing_prices_transaction_match_score) AS best_score, count(*) OVER ()::integer AS total
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.sale_listing_prices_transaction_match_run_id = $1::uuid
        AND c.sale_listing_prices_transaction_match_status = 'ambiguous'
    GROUP BY c.sale_listing_id
    ORDER BY best_score DESC, c.sale_listing_id
)
SELECT
    g.total,
    sl.sale_listing_id::text,
    sl.sale_listing_public_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_headline,
    COALESCE(sl.sale_listing_street_address, ''),
    COALESCE(sl.sale_listing_city, ''),
    COALESCE(sl.sale_listing_postal, ''),
    COALESCE(sl.sale_listing_asking_price::text, ''),
    COALESCE(sl.sale_listing_area_value::text, ''),
    COALESCE(sl.sale_listing_room_layout, ''),
    COALESCE(sl.sale_listing_last_seen_at::date::text, '')
FROM groups g
JOIN public.sale_listings sl ON sl.sale_listing_id = g.sale_listing_id
LIMIT 1 OFFSET $2`
	if offset < 0 {
		offset = 0
	}
	group := pricesMatchCandidateGroup{RunID: runID, Offset: offset}
	if err := pool.QueryRow(ctx, groupQuery, runID, offset).Scan(&group.Total, &group.Listing.ID, &group.Listing.PublicID, &group.Listing.Source, &group.Listing.Headline, &group.Listing.Address, &group.Listing.City, &group.Listing.Postal, &group.Listing.Price, &group.Listing.Area, &group.Listing.Layout, &group.Listing.Seen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pricesMatchCandidateGroup{RunID: runID}, nil
		}
		return pricesMatchCandidateGroup{}, fmt.Errorf("load candidate group: %w", err)
	}
	const candidatesQuery = `
SELECT
    c.sale_listing_prices_transaction_match_candidate_id::text,
    pt.prices_transaction_id::text,
    c.sale_listing_prices_transaction_match_score,
    COALESCE(round((c.sale_listing_prices_transaction_match_price_delta_percent * 100)::numeric, 1)::text || '%', ''),
    pt.prices_transaction_description,
    pt.prices_transaction_type,
    pt.prices_transaction_area::text,
    pt.prices_transaction_price::text,
    COALESCE(pt.prices_transaction_floor, ''),
    COALESCE(pt.prices_transaction_energy_class, ''),
    pt.prices_transaction_build_year::text,
    pt.prices_transaction_period_identifier,
    COALESCE(pn.prices_neighborhood_name, '')
FROM public.sale_listing_prices_transaction_match_candidates c
JOIN public.prices_transactions pt ON pt.prices_transaction_id = c.prices_transaction_id
LEFT JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
WHERE c.sale_listing_prices_transaction_match_run_id = $1::uuid
    AND c.sale_listing_id = $2::uuid
    AND c.sale_listing_prices_transaction_match_status = 'ambiguous'
ORDER BY c.sale_listing_prices_transaction_match_score DESC, c.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST, c.sale_listing_prices_transaction_match_candidate_id`
	rows, err := pool.Query(ctx, candidatesQuery, runID, group.Listing.ID)
	if err != nil {
		return pricesMatchCandidateGroup{}, fmt.Errorf("load candidates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var candidate pricesMatchCandidate
		if err := rows.Scan(&candidate.CandidateID, &candidate.TransactionID, &candidate.Score, &candidate.PriceDelta, &candidate.Description, &candidate.Type, &candidate.Area, &candidate.Price, &candidate.Floor, &candidate.Energy, &candidate.Year, &candidate.Period, &candidate.Neighborhood); err != nil {
			return pricesMatchCandidateGroup{}, fmt.Errorf("scan candidate: %w", err)
		}
		group.Candidates = append(group.Candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return pricesMatchCandidateGroup{}, fmt.Errorf("iterate candidates: %w", err)
	}
	return group, nil
}

func buildPricesMatchCandidateRows(candidates []pricesMatchCandidate) []table.Row {
	rows := make([]table.Row, 0, len(candidates))
	for _, candidate := range candidates {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", candidate.Score),
			trimForProgress(candidate.PriceDelta, 7),
			trimForProgress(candidate.Description, 24),
			trimForProgress(candidate.Type, 7),
			trimForProgress(candidate.Area, 7),
			trimForProgress(candidate.Price, 10),
			trimForProgress(candidate.Floor, 7),
			trimForProgress(candidate.Energy, 8),
			trimForProgress(candidate.Year, 6),
			trimForProgress(candidate.Period, 8),
			trimForProgress(candidate.Neighborhood, 18),
		})
	}
	return rows
}
