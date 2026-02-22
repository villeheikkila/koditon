package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"koditon-go/internal/ads"
)

type adsReportPageMsg struct {
	page ads.ReportPage
	err  error
}

type adsReportDetailMsg struct {
	key    string
	detail ads.Detail
	err    error
}

type adsReportBrowserScreen struct {
	ctx           *appContext
	action        action
	params        ads.SearchParams
	formValues    []string
	rows          []ads.ReportRow
	total         int64
	table         table.Model
	selected      int
	detail        ads.Detail
	detailErr     string
	detailCache   map[string]ads.Detail
	loading       bool
	errorText     string
	detailLoading string
	width         int
	height        int
	breadcrumb    string
}

func newAdsReportBrowserScreen(ctx *appContext, action action, params ads.SearchParams, formValues []string, breadcrumb string) Screen {
	cols := []table.Column{{Title: "Src", Width: 9}, {Title: "Kind", Width: 13}, {Title: "ID", Width: 12}, {Title: "City", Width: 12}, {Title: "Postal", Width: 8}, {Title: "Price", Width: 10}, {Title: "Area", Width: 8}, {Title: "Seen", Width: 10}, {Title: "Headline", Width: 36}}
	t := table.New(table.WithColumns(cols), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(jobTableStyles())
	return &adsReportBrowserScreen{ctx: ctx, action: action, params: params, formValues: append([]string(nil), formValues...), rows: nil, table: t, detailCache: map[string]ads.Detail{}, loading: true, breadcrumb: breadcrumb}
}

func (s *adsReportBrowserScreen) Key() string {
	return "ads-report-browser"
}

func (s *adsReportBrowserScreen) Init() tea.Cmd {
	return fetchAdsReportPageCmd(s.ctx.runner, s.params)
}

func (s *adsReportBrowserScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	bodyW := max(80, width-8)
	leftW := max(60, int(float64(bodyW)*0.62))
	rightW := max(26, bodyW-leftW-2)
	s.table.SetWidth(leftW - 4)
	s.table.SetHeight(max(10, height-14))
	_ = rightW
}

func (s *adsReportBrowserScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	switch typed := msg.(type) {
	case adsReportPageMsg:
		s.loading = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.errorText = ""
		s.rows = typed.page.Rows
		s.total = typed.page.Total
		s.params.Page = typed.page.Page
		s.params.PageSize = typed.page.PageSize
		s.selected = 0
		s.table.SetRows(buildAdsTableRows(s.rows))
		s.table.SetCursor(0)
		if len(s.rows) == 0 {
			s.detail = ads.Detail{}
			s.detailErr = "no rows"
			return nil
		}
		return s.loadSelectedDetailCmd()
	case adsReportDetailMsg:
		if typed.key == "" {
			return nil
		}
		s.detailLoading = ""
		if typed.err != nil {
			s.detailErr = typed.err.Error()
			return nil
		}
		s.detailErr = ""
		s.detail = typed.detail
		s.detailCache[typed.key] = typed.detail
		return nil
	}
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "left", "h", "backspace":
			nav.Pop()
			return nil
		case "n":
			if s.hasNextPage() && !s.loading {
				s.params.Page++
				s.formValues[11] = strconv.FormatInt(int64(s.params.Page), 10)
				s.loading = true
				return fetchAdsReportPageCmd(s.ctx.runner, s.params)
			}
			return nil
		case "p":
			if s.params.Page > 1 && !s.loading {
				s.params.Page--
				s.formValues[11] = strconv.FormatInt(int64(s.params.Page), 10)
				s.loading = true
				return fetchAdsReportPageCmd(s.ctx.runner, s.params)
			}
			return nil
		case "r":
			if !s.loading {
				s.loading = true
				return fetchAdsReportPageCmd(s.ctx.runner, s.params)
			}
			return nil
		case "f":
			nav.Replace(newAdsReportFormScreen(s.ctx, s.action, s.formValues, s.breadcrumb))
			return nil
		}
	}
	if s.loading {
		return nil
	}
	before := s.table.Cursor()
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	after := s.table.Cursor()
	if before != after {
		s.selected = after
		if loadCmd := s.loadSelectedDetailCmd(); loadCmd != nil {
			return tea.Batch(cmd, loadCmd)
		}
	}
	return cmd
}

func (s *adsReportBrowserScreen) View() string {
	leftContent := s.ctx.styles.progressLabel.Render(fmt.Sprintf("Results (%d rows total)", s.total)) + "\n" + s.ctx.styles.muted.Render(s.resultsMetaLine()) + "\n"
	if s.loading {
		leftContent += "\n" + s.ctx.styles.muted.Render("Loading page...")
	} else if s.errorText != "" {
		leftContent += "\n" + s.ctx.styles.error.Render(s.errorText)
	} else if len(s.rows) == 0 {
		leftContent += "\n" + s.ctx.styles.muted.Render("No rows for current filters.")
	} else {
		leftContent += "\n" + s.table.View()
	}
	rightContent := s.ctx.styles.progressLabel.Render("Detail") + "\n" + s.ctx.styles.muted.Render(s.selectedMetaLine()) + "\n\n" + s.detailView()
	bodyW := max(80, s.width-8)
	leftW := max(60, int(float64(bodyW)*0.62))
	rightW := max(26, bodyW-leftW-2)
	leftPanel := s.ctx.styles.panel.Width(leftW).Render(leftContent)
	rightPanel := s.ctx.styles.panel.Width(rightW).Render(rightContent)
	footer := s.ctx.styles.muted.Render("j/k move • n/p page • r reload • f filters • Esc back")
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel), "", footer)
}

func (s *adsReportBrowserScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: "j/k move • n/p page • r reload • f filters • Esc back • q quit"}
}

func (s *adsReportBrowserScreen) hasNextPage() bool {
	if s.params.PageSize <= 0 {
		return false
	}
	return int64(s.params.Page)*int64(s.params.PageSize) < s.total
}

func (s *adsReportBrowserScreen) loadSelectedDetailCmd() tea.Cmd {
	if s.selected < 0 || s.selected >= len(s.rows) {
		return nil
	}
	row := s.rows[s.selected]
	key := row.Source + ":" + row.Kind + ":" + row.EntityID
	if cached, ok := s.detailCache[key]; ok {
		s.detail = cached
		s.detailErr = ""
		return nil
	}
	s.detailLoading = key
	s.detailErr = ""
	return fetchAdsReportDetailCmd(s.ctx.runner, row.Source, row.Kind, row.EntityID, key)
}

func (s *adsReportBrowserScreen) resultsMetaLine() string {
	if s.total == 0 {
		return "page 1/1"
	}
	pages := int64(1)
	if s.params.PageSize > 0 {
		pages = (s.total + int64(s.params.PageSize) - 1) / int64(s.params.PageSize)
	}
	return fmt.Sprintf("page %d/%d • page_size=%d • source=%s • kind=%s • sort=%s", s.params.Page, pages, s.params.PageSize, s.params.Source, s.params.Kind, s.params.Sort)
}

func (s *adsReportBrowserScreen) selectedMetaLine() string {
	if len(s.rows) == 0 {
		return "no selection"
	}
	row := s.rows[s.selected]
	return fmt.Sprintf("%s/%s • id=%s", row.Source, row.Kind, row.EntityID)
}

func (s *adsReportBrowserScreen) detailView() string {
	if s.loading {
		return s.ctx.styles.muted.Render("Waiting for results...")
	}
	if s.detailLoading != "" {
		return s.ctx.styles.muted.Render("Loading detail...")
	}
	if s.detailErr != "" {
		return s.ctx.styles.error.Render(s.detailErr)
	}
	if len(s.detail.Summary) == 0 && len(s.detail.Related) == 0 {
		return s.ctx.styles.muted.Render("No detail available")
	}
	lines := make([]string, 0, len(s.detail.Summary)+len(s.detail.Related)+4)
	for _, field := range s.detail.Summary {
		lines = append(lines, s.ctx.styles.inputLabel.Render(field.Label)+": "+s.ctx.styles.normal.Render(field.Value))
	}
	if len(s.detail.Related) > 0 {
		lines = append(lines, "", s.ctx.styles.progressLabel.Render("Related"))
		for _, field := range s.detail.Related {
			lines = append(lines, s.ctx.styles.inputLabel.Render(field.Label)+": "+s.ctx.styles.normal.Render(field.Value))
		}
	}
	return strings.Join(lines, "\n")
}

func buildAdsTableRows(rows []ads.ReportRow) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		out = append(out, table.Row{
			row.Source,
			row.Kind,
			trimForProgress(row.EntityID, 12),
			trimForProgress(row.City, 12),
			trimForProgress(row.Postal, 8),
			trimForProgress(int64PtrToString(row.Price), 10),
			trimForProgress(float64PtrToString(row.Area), 8),
			row.LastSeenAt.Format("2006-01-02"),
			trimForProgress(firstNonEmpty(row.Headline, row.Address), 60),
		})
	}
	return out
}

func fetchAdsReportPageCmd(runner interface {
	AdsSearchReports(context.Context, ads.SearchParams) (ads.ReportPage, error)
}, params ads.SearchParams) tea.Cmd {
	return func() tea.Msg {
		if runner == nil {
			return adsReportPageMsg{err: fmt.Errorf("runner unavailable")}
		}
		page, err := runner.AdsSearchReports(context.Background(), params)
		return adsReportPageMsg{page: page, err: err}
	}
}

func fetchAdsReportDetailCmd(runner interface {
	AdsReportDetail(context.Context, string, string, string) (ads.Detail, error)
}, source, kind, entityID, key string) tea.Cmd {
	return func() tea.Msg {
		if runner == nil {
			return adsReportDetailMsg{key: key, err: fmt.Errorf("runner unavailable")}
		}
		detail, err := runner.AdsReportDetail(context.Background(), source, kind, entityID)
		return adsReportDetailMsg{key: key, detail: detail, err: err}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func int64PtrToString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func float64PtrToString(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *value)
}
