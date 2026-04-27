package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type promptInput struct {
	model textinput.Model
}

func newPromptInput() promptInput {
	ti := textinput.New()
	ti.Placeholder = "input"
	ti.CharLimit = 256
	ti.SetWidth(48)
	ti.Prompt = "> "
	return promptInput{model: ti}
}

func (p *promptInput) Focus() {
	p.model.Focus()
}

func (p *promptInput) Blur() {
	p.model.Blur()
}

func (p *promptInput) SetPlaceholder(value string) {
	p.model.Placeholder = value
}

func (p *promptInput) SetValue(value string) {
	p.model.SetValue(value)
}

func (p *promptInput) Value() string {
	return strings.TrimSpace(p.model.Value())
}

func (p *promptInput) Resize(width int) {
	p.model.SetWidth(max(24, min(60, width)))
}

func (p *promptInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.model, cmd = p.model.Update(msg)
	return cmd
}

func (p *promptInput) View() string {
	return p.model.View()
}
