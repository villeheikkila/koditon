package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type homeScreen struct {
	ctx    *appContext
	list   selectableList
	width  int
	height int
}

func newHomeScreen(ctx *appContext) *homeScreen {
	l := newSelectableList("Subsystems", 48, 20, ctx.styles)
	items := make([]navItem, 0, len(ctx.subsystems))
	for i, sub := range ctx.subsystems {
		items = append(items, navItem{idx: i, title: sub.Title, desc: sub.Description})
	}
	l.SetItems(items)
	return &homeScreen{ctx: ctx, list: l}
}

func (s *homeScreen) Key() string {
	return "home"
}

func (s *homeScreen) Init() tea.Cmd {
	return nil
}

func (s *homeScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	contentH := max(16, height-8)
	listW := max(56, width-6)
	s.list.Resize(listW, contentH)
}

func (s *homeScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "enter", "right", "l":
			subIdx := safeIndex(s.list.SelectedIndex(), len(s.ctx.subsystems))
			nav.Push(newActionsScreen(s.ctx, subIdx))
			return nil
		}
	}
	return s.list.Update(msg)
}

func (s *homeScreen) View() string {
	sub := s.ctx.subsystems[safeIndex(s.list.SelectedIndex(), len(s.ctx.subsystems))]
	var b strings.Builder
	b.WriteString(s.ctx.styles.logo.Render(appLogo))
	b.WriteString("\n")
	b.WriteString(s.list.View())
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.progressLabel.Render("Selected Subsystem"))
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.selected.Render(sub.Title))
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.description.Render(sub.Description))
	b.WriteString("\n")
	b.WriteString(s.ctx.styles.muted.Render("Press Enter to open actions."))
	return b.String()
}

func (s *homeScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: "Subsystems", Help: helpDefault()}
}
