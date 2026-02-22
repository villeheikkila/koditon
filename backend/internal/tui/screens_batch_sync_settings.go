package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type batchSyncSettingsScreen struct {
	ctx        *appContext
	action     action
	form       formPrimitive
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newBatchSyncSettingsScreen(ctx *appContext, action action, _ []string, breadcrumb string) Screen {
	fields := []formField{
		newTextFormField("max_entries", "Max entries to sync (blank = all)", "10"),
		newTextFormField("delay", "Delay between runs (blank = none)", "1s"),
	}
	form := newFormPrimitive(ctx.styles, fields)
	return &batchSyncSettingsScreen{ctx: ctx, action: action, form: form, breadcrumb: breadcrumb}
}

func (s *batchSyncSettingsScreen) Key() string {
	return "batch-sync-settings"
}

func (s *batchSyncSettingsScreen) Init() tea.Cmd {
	return nil
}

func (s *batchSyncSettingsScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.form.Resize(width - 12)
}

func (s *batchSyncSettingsScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			nav.Pop()
			return nil
		case "enter":
			inputs, err := s.buildInputs()
			if err != nil {
				s.errorText = err.Error()
				return nil
			}
			s.errorText = ""
			nav.Replace(newJobScreen(s.ctx, s.action, inputs, s.breadcrumb))
			return nil
		}
	}
	return s.form.Update(msg)
}

func (s *batchSyncSettingsScreen) View() string {
	content := s.ctx.styles.title.Render(s.action.Title) + "\n" + s.ctx.styles.description.Render("Batch sync settings") + "\n\n" + s.form.View() + "\n\n" + s.ctx.styles.muted.Render("Enter runs with settings. Example: max=10 and delay=1s.")
	if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	}
	return s.ctx.styles.panel.Width(max(56, min(100, s.width-8))).Render(content)
}

func (s *batchSyncSettingsScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: helpDefault()}
}

func (s *batchSyncSettingsScreen) buildInputs() ([]string, error) {
	values := s.form.Values()
	maxEntries := strings.TrimSpace(values["max_entries"])
	delay := strings.TrimSpace(values["delay"])
	if maxEntries != "" {
		n, err := strconv.Atoi(maxEntries)
		if err != nil {
			return nil, fmt.Errorf("max entries must be numeric")
		}
		if n < 1 {
			return nil, fmt.Errorf("max entries must be at least 1")
		}
	}
	if delay != "" {
		if _, err := parseBatchRunOptions([]string{maxEntries, delay}); err != nil {
			return nil, err
		}
	}
	return []string{maxEntries, delay}, nil
}
