package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

type jobScreen struct {
	ctx        *appContext
	action     action
	inputs     []string
	view       jobView
	running    bool
	events     chan tea.Msg
	breadcrumb string
}

func newJobScreen(ctx *appContext, action action, inputs []string, breadcrumb string) *jobScreen {
	return &jobScreen{ctx: ctx, action: action, inputs: append([]string(nil), inputs...), view: newJobView(ctx.styles), events: make(chan tea.Msg, 256), breadcrumb: breadcrumb}
}

func (s *jobScreen) Key() string {
	return "job"
}

func (s *jobScreen) Init() tea.Cmd {
	s.running = true
	startCmd := s.view.Start(s.action.Title)
	_, err := s.ctx.runtime.Start(s.ctx.runner, s.action, s.inputs, s.events)
	if err != nil {
		s.running = false
		if errors.Is(err, ErrJobAlreadyRunning) {
			s.view.lastError = "another job is already running"
		} else {
			s.view.lastError = err.Error()
		}
		return nil
	}
	return tea.Batch(startCmd, waitForEventCmd(s.events))
}

func (s *jobScreen) Resize(width int, height int) {
	s.view.Resize(width, height)
}

func (s *jobScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "left", "h", "backspace":
			if !s.running {
				nav.Pop()
				return nil
			}
		}
	}
	if s.running {
		switch typed := msg.(type) {
		case runProgressMsg:
			cmd := s.view.OnProgress(typed)
			return tea.Batch(cmd, waitForEventCmd(s.events))
		case runFinishedMsg:
			s.running = false
			s.view.OnFinished(typed)
			return nil
		default:
			return s.view.UpdateRunning(msg)
		}
	}
	return s.view.UpdateFinished(msg)
}

func (s *jobScreen) View() string {
	return s.view.View(s.running)
}

func (s *jobScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: helpDefault()}
}
