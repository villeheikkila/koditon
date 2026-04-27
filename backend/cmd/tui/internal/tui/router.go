package tui

import tea "charm.land/bubbletea/v2"

type Screen interface {
	Key() string
	Init() tea.Cmd
	Resize(width int, height int)
	Update(msg tea.Msg, nav Navigator) tea.Cmd
	View() string
	ShellState() shellState
}

type Navigator interface {
	Push(screen Screen)
	Pop()
	Replace(screen Screen)
	Reset(screen Screen)
	Quit()
}

type router struct {
	stack  []Screen
	width  int
	height int
	quit   bool
}

func newRouter(initial Screen) *router {
	return &router{stack: []Screen{initial}}
}

func (r *router) top() Screen {
	if len(r.stack) == 0 {
		return nil
	}
	return r.stack[len(r.stack)-1]
}

func (r *router) resize(width int, height int) {
	r.width = width
	r.height = height
	top := r.top()
	if top == nil {
		return
	}
	top.Resize(width, height)
}

type routerNavigator struct {
	r *router
}

func (n routerNavigator) Push(screen Screen) {
	n.r.stack = append(n.r.stack, screen)
	if n.r.width > 0 && n.r.height > 0 {
		screen.Resize(n.r.width, n.r.height)
	}
}

func (n routerNavigator) Pop() {
	if len(n.r.stack) <= 1 {
		return
	}
	n.r.stack = n.r.stack[:len(n.r.stack)-1]
}

func (n routerNavigator) Replace(screen Screen) {
	if len(n.r.stack) == 0 {
		n.r.stack = []Screen{screen}
	} else {
		n.r.stack[len(n.r.stack)-1] = screen
	}
	if n.r.width > 0 && n.r.height > 0 {
		screen.Resize(n.r.width, n.r.height)
	}
}

func (n routerNavigator) Reset(screen Screen) {
	n.r.stack = []Screen{screen}
	if n.r.width > 0 && n.r.height > 0 {
		screen.Resize(n.r.width, n.r.height)
	}
}

func (n routerNavigator) Quit() {
	n.r.quit = true
}
