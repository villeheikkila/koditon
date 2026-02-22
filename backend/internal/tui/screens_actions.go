package tui

import tea "github.com/charmbracelet/bubbletea"

type actionsScreen struct {
	ctx          *appContext
	subsystemIdx int
	list         selectableList
	width        int
	height       int
}

func newActionsScreen(ctx *appContext, subsystemIdx int) *actionsScreen {
	safe := safeIndex(subsystemIdx, len(ctx.subsystems))
	sub := ctx.subsystems[safe]
	l := newSelectableList(sub.Title+" Actions", 56, 20, ctx.styles)
	items := make([]navItem, 0, len(sub.Actions))
	for i, a := range sub.Actions {
		items = append(items, navItem{idx: i, title: a.Title, desc: a.Description})
	}
	l.SetItems(items)
	return &actionsScreen{ctx: ctx, subsystemIdx: safe, list: l}
}

func (s *actionsScreen) Key() string {
	return "actions"
}

func (s *actionsScreen) Init() tea.Cmd {
	return nil
}

func (s *actionsScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	contentH := max(16, height-8)
	listW := max(56, width-6)
	s.list.Resize(listW, contentH)
}

func (s *actionsScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "left", "h", "backspace":
			nav.Pop()
			return nil
		case "enter", "right", "l":
			a := s.selectedAction()
			if len(a.Prompts) == 0 {
				nav.Push(newJobScreen(s.ctx, a, nil, breadcrumbForActions(s.ctx.subsystems[s.subsystemIdx])))
				return nil
			}
			if a.UseCityPicker {
				nav.Push(newCityPickerScreen(s.ctx, a, nil, breadcrumbForActions(s.ctx.subsystems[s.subsystemIdx])))
				return nil
			}
			nav.Push(newPromptScreen(s.ctx, a, nil, 0, breadcrumbForActions(s.ctx.subsystems[s.subsystemIdx])))
			return nil
		}
	}
	return s.list.Update(msg)
}

func (s *actionsScreen) selectedAction() action {
	sub := s.ctx.subsystems[s.subsystemIdx]
	idx := safeIndex(s.list.SelectedIndex(), len(sub.Actions))
	return sub.Actions[idx]
}

func (s *actionsScreen) View() string {
	return s.ctx.styles.panel.Width(max(56, s.width-6)).Render(s.list.View())
}

func (s *actionsScreen) ShellState() shellState {
	sub := s.ctx.subsystems[s.subsystemIdx]
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: breadcrumbForActions(sub), Help: helpDefault()}
}
