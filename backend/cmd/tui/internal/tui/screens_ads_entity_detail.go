package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"koditon/internal/domain/ads"
)

const rawJSONDefaultLimit = 8 * 1024

type adsEntityDetailMsg struct {
	detail ads.UnifiedEntityDetail
	err    error
}

type adsEntityDetailScreen struct {
	ctx         *appContext
	row         ads.UnifiedEntityRow
	detail      ads.UnifiedEntityDetail
	loading     bool
	errorText   string
	scroll      int
	rawExpanded bool
	width       int
	height      int
	breadcrumb  string
}

func newAdsEntityDetailScreen(ctx *appContext, row ads.UnifiedEntityRow, breadcrumb string) Screen {
	return &adsEntityDetailScreen{ctx: ctx, row: row, loading: true, rawExpanded: false, breadcrumb: breadcrumb}
}

func (s *adsEntityDetailScreen) Key() string { return "ads-entity-detail" }

func (s *adsEntityDetailScreen) Init() tea.Cmd {
	return fetchAdsEntityDetailCmd(s.ctx.runner, s.row.CanonicalID)
}

func (s *adsEntityDetailScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
}

func (s *adsEntityDetailScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	switch typed := msg.(type) {
	case adsEntityDetailMsg:
		s.loading = false
		if typed.err != nil {
			s.errorText = typed.err.Error()
			return nil
		}
		s.detail = typed.detail
		s.errorText = ""
		s.scroll = 0
		return nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "esc", "backspace", "left", "h":
		nav.Pop()
		return nil
	case "down", "j":
		s.scroll++
		return nil
	case "up", "k":
		s.scroll = max(0, s.scroll-1)
		return nil
	case "pgdown":
		s.scroll += max(1, s.height/3)
		return nil
	case "pgup":
		s.scroll = max(0, s.scroll-max(1, s.height/3))
		return nil
	case "x":
		s.rawExpanded = !s.rawExpanded
		return nil
	}
	return nil
}

func (s *adsEntityDetailScreen) View() string {
	panelW := max(70, s.width-8)
	content := s.renderContent(panelW - 4)
	lines := strings.Split(content, "\n")
	maxLines := max(10, s.height-10)
	if s.scroll > max(0, len(lines)-1) {
		s.scroll = max(0, len(lines)-1)
	}
	end := min(len(lines), s.scroll+maxLines)
	visible := strings.Join(lines[s.scroll:end], "\n")
	footer := s.ctx.styles.muted.Render("Esc back • j/k scroll • PgUp/PgDn • x toggle raw JSON")
	return s.ctx.styles.panel.Width(panelW).Render(visible) + "\n\n" + footer
}

func (s *adsEntityDetailScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: "Esc back • j/k scroll • x raw JSON • q quit"}
}

func (s *adsEntityDetailScreen) renderContent(width int) string {
	if s.loading {
		return s.ctx.styles.muted.Render("Loading detail...")
	}
	if s.errorText != "" {
		return s.ctx.styles.error.Render(s.errorText)
	}
	lines := make([]string, 0, 128)
	lines = append(lines, s.ctx.styles.progressLabel.Render("Canonical"))
	canonical := s.detail.Canonical
	lines = append(lines,
		renderDetailLine(s.ctx.styles, "Canonical ID", canonical.CanonicalID),
		renderDetailLine(s.ctx.styles, "Source", canonical.Source),
		renderDetailLine(s.ctx.styles, "Kind", canonical.Kind),
		renderDetailLine(s.ctx.styles, "Native ID", canonical.NativeID),
		renderDetailLine(s.ctx.styles, "Headline", canonical.Headline),
		renderDetailLine(s.ctx.styles, "Address", canonical.Address),
		renderDetailLine(s.ctx.styles, "City", canonical.City),
		renderDetailLine(s.ctx.styles, "Postal", canonical.Postal),
		renderDetailLine(s.ctx.styles, "Price", int64PtrToString(canonical.Price)),
		renderDetailLine(s.ctx.styles, "Area", float64PtrToString(canonical.Area)),
		renderDetailLine(s.ctx.styles, "Room Layout", canonical.RoomLayout),
		renderDetailLine(s.ctx.styles, "URL", canonical.URL),
		renderDetailLine(s.ctx.styles, "Web", s.webLink()),
		renderDetailLine(s.ctx.styles, "Last Seen", canonical.LastSeenAt.Format("2006-01-02 15:04:05Z07:00")),
	)
	for _, field := range s.detail.CanonicalExtra {
		lines = append(lines, renderDetailLine(s.ctx.styles, field.Label, field.Value))
	}
	if len(s.detail.SourceSpecific) > 0 {
		lines = append(lines, "", s.ctx.styles.progressLabel.Render("Source Specific"))
		for _, field := range s.detail.SourceSpecific {
			lines = append(lines, renderDetailLine(s.ctx.styles, field.Label, field.Value))
		}
	}
	if len(s.detail.Related) > 0 {
		lines = append(lines, "", s.ctx.styles.progressLabel.Render("Related"))
		for _, field := range s.detail.Related {
			lines = append(lines, renderDetailLine(s.ctx.styles, field.Label, field.Value))
		}
	}
	if strings.TrimSpace(s.detail.Raw.Pretty) != "" {
		lines = append(lines, "", s.ctx.styles.progressLabel.Render("Raw JSON"))
		raw := s.detail.Raw.Pretty
		title := fmt.Sprintf("payload bytes=%d", s.detail.Raw.OriginalBytes)
		if !s.rawExpanded && len(raw) > rawJSONDefaultLimit {
			raw = raw[:rawJSONDefaultLimit] + "\n... [truncated, press x to expand]"
			title = fmt.Sprintf("%s (showing first %d)", title, rawJSONDefaultLimit)
		} else if s.rawExpanded {
			title = title + " (expanded)"
		}
		lines = append(lines, s.ctx.styles.muted.Render(title))
		for line := range strings.SplitSeq(raw, "\n") {
			lines = append(lines, trimForProgress(line, max(40, width-2)))
		}
	}
	return strings.Join(lines, "\n")
}

func (s *adsEntityDetailScreen) webLink() string {
	base := strings.TrimSpace(s.ctx.webBaseURL)
	id := strings.TrimSpace(s.detail.Canonical.CanonicalID)
	if base == "" || id == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/detail/" + id
}

func renderDetailLine(st styles, label string, value string) string {
	return st.inputLabel.Render(label) + ": " + st.normal.Render(strings.TrimSpace(value))
}

func fetchAdsEntityDetailCmd(runner interface {
	AdsReportDetail(context.Context, string) (ads.UnifiedEntityDetail, error)
}, canonicalID string,
) tea.Cmd {
	return func() tea.Msg {
		if runner == nil {
			return adsEntityDetailMsg{err: fmt.Errorf("runner unavailable")}
		}
		detail, err := runner.AdsReportDetail(context.Background(), canonicalID)
		return adsEntityDetailMsg{detail: detail, err: err}
	}
}
