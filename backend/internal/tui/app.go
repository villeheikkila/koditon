package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"koditon-go/internal/syncflows"
)

type runFinishedMsg struct {
	actionTitle string
	output      string
	err         error
	duration    time.Duration
}

type model struct {
	runner       *syncflows.Runner
	actions      []action
	cursor       int
	input        textinput.Model
	awaitingText bool
	running      bool
	currentTitle string
	lastOutput   string
	lastError    string
	lastDuration time.Duration
	spinner      spinner.Model
	width        int
	height       int
}

func NewModel(runner *syncflows.Runner) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "input"
	ti.CharLimit = 256
	ti.Width = 40
	ti.Prompt = "> "
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return model{runner: runner, actions: buildActions(), input: ti, spinner: sp}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.running {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		case runFinishedMsg:
			m.running = false
			m.lastDuration = msg.duration
			m.lastOutput = msg.output
			if msg.err != nil {
				m.lastError = msg.err.Error()
			} else {
				m.lastError = ""
			}
			return m, nil
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.awaitingText {
			switch msg.String() {
			case "esc":
				m.awaitingText = false
				m.input.SetValue("")
				return m, nil
			case "enter":
				input := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				m.awaitingText = false
				if input == "" {
					m.lastError = "input cannot be empty"
					return m, nil
				}
				a := m.actions[m.cursor]
				m.running = true
				m.currentTitle = a.Title
				m.lastOutput = ""
				m.lastError = ""
				return m, tea.Batch(m.spinner.Tick, runActionCmd(m.runner, a, input))
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			a := m.actions[m.cursor]
			if a.NeedsInput {
				m.awaitingText = true
				m.input.Placeholder = a.Prompt
				m.input.Focus()
				return m, textinput.Blink
			}
			m.running = true
			m.currentTitle = a.Title
			m.lastOutput = ""
			m.lastError = ""
			return m, tea.Batch(m.spinner.Tick, runActionCmd(m.runner, a, ""))
		}
	}
	return m, nil
}

func runActionCmd(runner *syncflows.Runner, a action, input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		start := time.Now()
		output, err := a.Run(ctx, runner, input)
		return runFinishedMsg{actionTitle: a.Title, output: output, err: err, duration: time.Since(start)}
	}
}

func (m model) View() string {
	if m.running {
		return lipgloss.NewStyle().Padding(1, 2).Render(fmt.Sprintf("%s Running: %s\n\nPlease wait...", m.spinner.View(), m.currentTitle))
	}
	if m.awaitingText {
		head := lipgloss.NewStyle().Bold(true).Render(m.actions[m.cursor].Title)
		return lipgloss.NewStyle().Padding(1, 2).Render(head + "\n" + m.actions[m.cursor].Description + "\n\nEnter " + m.actions[m.cursor].Prompt + " (Esc to cancel):\n" + m.input.View())
	}
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("Koditon Manual Sync TUI")
	b.WriteString(title)
	b.WriteString("\nUse ↑/↓ to move, Enter to run, q to quit.\n\n")
	for i, a := range m.actions {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		line := fmt.Sprintf("%s %s", cursor, a.Title)
		if m.cursor == i {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.lastOutput != "" || m.lastError != "" {
		b.WriteString("\nLast run:\n")
		if m.lastOutput != "" {
			b.WriteString("  Output: ")
			b.WriteString(m.lastOutput)
			b.WriteString("\n")
		}
		if m.lastError != "" {
			b.WriteString("  Error: ")
			b.WriteString(m.lastError)
			b.WriteString("\n")
		}
		if m.lastDuration > 0 {
			b.WriteString("  Duration: ")
			b.WriteString(m.lastDuration.Round(time.Millisecond).String())
			b.WriteString("\n")
		}
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
