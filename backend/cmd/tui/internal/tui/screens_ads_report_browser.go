package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"koditon-go/internal/domain/ads"
)

type adsReportPageMsg struct {
	page ads.ReportPage
	err  error
}

type adsReportBrowserScreen struct {
	ctx        *appContext
	action     action
	params     ads.SearchParams
	formValues []string
	rows       []ads.UnifiedEntityRow
	total      int64
	table      table.Model
	selected   int
	loading    bool
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newAdsReportBrowserScreen(ctx *appContext, action action, params ads.SearchParams, formValues []string, breadcrumb string) Screen {
	cols := []table.Column{{Title: "Src", Width: 9}, {Title: "Kind", Width: 13}, {Title: "ID", Width: 18}, {Title: "City", Width: 12}, {Title: "Postal", Width: 8}, {Title: "Price", Width: 10}, {Title: "Area", Width: 8}, {Title: "Seen", Width: 10}, {Title: "Headline", Width: 40}}
	t := table.New(table.WithColumns(cols), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(jobTableStyles())
	return &adsReportBrowserScreen{ctx: ctx, action: action, params: params, formValues: append([]string(nil), formValues...), rows: nil, table: t, loading: true, breadcrumb: breadcrumb}
}

func (s *adsReportBrowserScreen) Key() string { return "ads-report-browser" }

func (s *adsReportBrowserScreen) Init() tea.Cmd { return fetchAdsReportPageCmd(s.ctx.runner, s.params) }

func (s *adsReportBrowserScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.table.SetWidth(max(70, width-12))
	s.table.SetHeight(max(10, height-14))
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
		return nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "esc", "left", "h", "backspace":
			nav.Pop()
			return nil
		case "enter":
			if len(s.rows) == 0 || s.loading {
				return nil
			}
			idx := safeIndex(s.table.Cursor(), len(s.rows))
			row := s.rows[idx]
			nav.Push(newAdsEntityDetailScreen(s.ctx, row, s.breadcrumb+" > Detail"))
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
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *adsReportBrowserScreen) View() string {
	content := s.ctx.styles.progressLabel.Render(fmt.Sprintf("Results (%d rows total)", s.total)) + "\n" + s.ctx.styles.muted.Render(s.resultsMetaLine()) + "\n"
	if s.loading {
		content += "\n" + s.ctx.styles.muted.Render("Loading page...")
	} else if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	} else if len(s.rows) == 0 {
		content += "\n" + s.ctx.styles.muted.Render("No rows for current filters.")
	} else {
		content += "\n" + s.table.View()
	}
	content += "\n\n" + s.ctx.styles.muted.Render("Enter detail • j/k move • n/p page • r reload • f filters • Esc back")
	return s.ctx.styles.panel.Width(max(70, s.width-8)).Render(content)
}

func (s *adsReportBrowserScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: "Enter detail • j/k move • n/p page • r reload • f filters • Esc back • q quit"}
}

func (s *adsReportBrowserScreen) hasNextPage() bool {
	if s.params.PageSize <= 0 {
		return false
	}
	return int64(s.params.Page)*int64(s.params.PageSize) < s.total
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

func buildAdsTableRows(rows []ads.UnifiedEntityRow) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		out = append(out, table.Row{row.Source, row.Kind, trimForProgress(row.NativeID, 18), trimForProgress(row.City, 12), trimForProgress(row.Postal, 8), trimForProgress(int64PtrToString(row.Price), 10), trimForProgress(float64PtrToString(row.Area), 8), row.LastSeenAt.Format("2006-01-02"), trimForProgress(firstNonEmpty(row.Headline, row.Address), 80)})
	}
	return out
}

func fetchAdsReportPageCmd(runner interface {
	AdsSearchReports(context.Context, ads.SearchParams) (ads.ReportPage, error)
}, params ads.SearchParams,
) tea.Cmd {
	return func() tea.Msg {
		if runner == nil {
			return adsReportPageMsg{err: fmt.Errorf("runner unavailable")}
		}
		page, err := runner.AdsSearchReports(context.Background(), params)
		return adsReportPageMsg{page: page, err: err}
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
