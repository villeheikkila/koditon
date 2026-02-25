package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
)

type promptScreen struct {
	ctx        *appContext
	action     action
	step       int
	values     []string
	input      promptInput
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newPromptScreen(ctx *appContext, action action, values []string, step int, breadcrumb string) *promptScreen {
	in := newPromptInput()
	in.Focus()
	in.SetPlaceholder(action.Prompts[step])
	return &promptScreen{ctx: ctx, action: action, step: step, values: append([]string(nil), values...), input: in, breadcrumb: breadcrumb}
}

func (s *promptScreen) Key() string {
	return "prompt"
}

func (s *promptScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *promptScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.input.Resize(width - 16)
}

func (s *promptScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "esc":
			nav.Pop()
			return nil
		case "enter":
			value := s.input.Value()
			if value == "" {
				s.errorText = "input cannot be empty"
				return nil
			}
			s.errorText = ""
			s.values = append(s.values, value)
			next := s.step + 1
			if next >= len(s.action.Prompts) {
				nav.Replace(nextScreenForActionInput(s.ctx, s.action, s.values, s.breadcrumb))
				return nil
			}
			s.step = next
			s.input.SetPlaceholder(s.action.Prompts[s.step])
			s.input.SetValue("")
			return textinput.Blink
		}
	}
	return s.input.Update(msg)
}

func (s *promptScreen) View() string {
	content := s.ctx.styles.title.Render(s.action.Title) + "\n" + s.ctx.styles.description.Render(s.action.Description) + "\n\n" + s.ctx.styles.inputLabel.Render(formatActionPrompt(s.action, s.step)) + "\n" + s.input.View()
	if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	}
	return s.ctx.styles.panel.Width(max(56, min(90, s.width-8))).Render(content)
}

func (s *promptScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: helpDefault()}
}
