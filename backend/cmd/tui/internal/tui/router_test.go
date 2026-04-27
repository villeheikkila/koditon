package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeScreen struct {
	key       string
	initCount int
	resizedW  int
	resizedH  int
}

func (s *fakeScreen) Key() string {
	return s.key
}

func (s *fakeScreen) Init() tea.Cmd {
	s.initCount++
	return nil
}

func (s *fakeScreen) Resize(width int, height int) {
	s.resizedW = width
	s.resizedH = height
}

func (s *fakeScreen) Update(_ tea.Msg, _ Navigator) tea.Cmd {
	return nil
}

func (s *fakeScreen) View() string {
	return s.key
}

func (s *fakeScreen) ShellState() shellState {
	return shellState{Title: s.key}
}

func TestRouterNavigation(t *testing.T) {
	home := &fakeScreen{key: "home"}
	r := newRouter(home)
	r.resize(120, 40)
	if got := r.top().Key(); got != "home" {
		t.Fatalf("expected top home, got %s", got)
	}
	nav := routerNavigator{r: r}
	actions := &fakeScreen{key: "actions"}
	nav.Push(actions)
	if got := r.top().Key(); got != "actions" {
		t.Fatalf("expected top actions, got %s", got)
	}
	if actions.resizedW != 120 || actions.resizedH != 40 {
		t.Fatalf("expected pushed screen resized to 120x40, got %dx%d", actions.resizedW, actions.resizedH)
	}
	prompt := &fakeScreen{key: "prompt"}
	nav.Replace(prompt)
	if got := r.top().Key(); got != "prompt" {
		t.Fatalf("expected top prompt, got %s", got)
	}
	nav.Pop()
	if got := r.top().Key(); got != "home" {
		t.Fatalf("expected top home after pop, got %s", got)
	}
	job := &fakeScreen{key: "job"}
	nav.Reset(job)
	if len(r.stack) != 1 {
		t.Fatalf("expected stack size 1, got %d", len(r.stack))
	}
	if got := r.top().Key(); got != "job" {
		t.Fatalf("expected top job after reset, got %s", got)
	}
	nav.Quit()
	if !r.quit {
		t.Fatalf("expected router quit flag true")
	}
}
