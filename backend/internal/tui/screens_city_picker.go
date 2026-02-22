package tui

import tea "github.com/charmbracelet/bubbletea"

type cityOptionsMsg struct {
	cities []string
	err    error
}

type cityPickerScreen struct {
	ctx        *appContext
	action     action
	values     []string
	picker     fuzzyPicker
	loading    bool
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newCityPickerScreen(ctx *appContext, action action, values []string, breadcrumb string) *cityPickerScreen {
	picker := newFuzzyPicker("Select City", ctx.styles)
	picker.Focus()
	return &cityPickerScreen{ctx: ctx, action: action, values: append([]string(nil), values...), picker: picker, loading: true, breadcrumb: breadcrumb}
}

func (s *cityPickerScreen) Key() string {
	return "city-picker"
}

func (s *cityPickerScreen) Init() tea.Cmd {
	return fetchPricesCitiesCmd(s.ctx.runner)
}

func (s *cityPickerScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.picker.Resize(width, height)
}

func (s *cityPickerScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	if optionsMsg, ok := msg.(cityOptionsMsg); ok {
		s.loading = false
		if optionsMsg.err != nil {
			s.errorText = "load cities: " + optionsMsg.err.Error()
			return nil
		}
		s.errorText = ""
		s.picker.SetOptions(optionsMsg.cities)
		return nil
	}
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			nav.Pop()
			return nil
		case "enter":
			if s.loading {
				return nil
			}
			city := s.picker.Selected()
			if city == "" {
				s.errorText = "select a city"
				return nil
			}
			inputs := append(append([]string(nil), s.values...), city)
			if len(s.action.Prompts) > len(inputs) {
				nav.Replace(newPromptScreen(s.ctx, s.action, inputs, len(inputs), s.breadcrumb))
				return nil
			}
			nav.Replace(newJobScreen(s.ctx, s.action, inputs, s.breadcrumb))
			return nil
		}
	}
	if s.loading {
		return nil
	}
	return s.picker.Update(msg)
}

func (s *cityPickerScreen) View() string {
	body := s.ctx.styles.muted.Render("Loading cities...")
	if !s.loading {
		body = s.picker.View()
	}
	content := s.ctx.styles.title.Render(s.action.Title) + "\n" + s.ctx.styles.description.Render(s.action.Description) + "\n\n" + s.ctx.styles.inputLabel.Render("Type to fuzzy-match cities. Enter selects closest match. Esc cancel") + "\n" + body
	if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	}
	return s.ctx.styles.panel.Width(max(56, min(110, s.width-8))).Render(content)
}

func (s *cityPickerScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: helpDefault()}
}
