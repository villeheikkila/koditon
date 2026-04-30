package tui

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	syncflows "koditon/internal/sync/flows"
	syncjobs "koditon/internal/sync/jobs"

	tea "charm.land/bubbletea/v2"
)

type AppOption func(*appConfig)

type appConfig struct {
	webBaseURL string
	syncJobs   *syncjobs.Store
	dbPool     *pgxpool.Pool
}

type appContext struct {
	runner     *syncflows.Runner
	syncJobs   *syncjobs.Store
	dbPool     *pgxpool.Pool
	styles     styles
	runtime    *jobRuntime
	subsystems []subsystem
	webBaseURL string
}

type App struct {
	root *rootModel
}

type rootModel struct {
	ctx    *appContext
	router *router
}

const appLogo = `
 _  __         _ _ _
| |/ /___   __| (_) |_ ___  _ __
| ' // _ \ / _  | | __/ _ \| '_ \
| . \ (_) | (_| | | || (_) | | | |
|_|\_\___/ \__,_|_|\__\___/|_| |_|
`

func WithWebBaseURL(url string) AppOption {
	return func(cfg *appConfig) { cfg.webBaseURL = url }
}

func WithSyncJobs(store *syncjobs.Store) AppOption {
	return func(cfg *appConfig) { cfg.syncJobs = store }
}

func WithDBPool(pool *pgxpool.Pool) AppOption {
	return func(cfg *appConfig) { cfg.dbPool = pool }
}

func NewApp(runner *syncflows.Runner, opts ...AppOption) *App {
	cfg := appConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	ctx := &appContext{runner: runner, syncJobs: cfg.syncJobs, dbPool: cfg.dbPool, styles: defaultStyles(), runtime: newJobRuntime(), subsystems: buildSubsystems(), webBaseURL: cfg.webBaseURL}
	home := newHomeScreen(ctx)
	r := newRouter(home)
	return &App{root: &rootModel{ctx: ctx, router: r}}
}

func (a *App) Model() tea.Model {
	return a.root
}

func (m *rootModel) Init() tea.Cmd {
	top := m.router.top()
	if top == nil {
		return nil
	}
	return top.Init()
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.router.resize(typed.Width, typed.Height)
		return m, nil
	case tea.KeyPressMsg:
		if typed.String() == "ctrl+c" {
			if m.ctx.runtime.CancelActive() {
				return m, nil
			}
			return m, tea.Quit
		}
		if typed.String() == "q" {
			return m, tea.Quit
		}
	}
	top := m.router.top()
	if top == nil {
		return m, tea.Quit
	}
	nav := routerNavigator{r: m.router}
	cmd := top.Update(msg, nav)
	if m.router.quit {
		return m, tea.Quit
	}
	nextTop := m.router.top()
	if nextTop == nil {
		return m, tea.Quit
	}
	if nextTop != top {
		return m, tea.Batch(cmd, nextTop.Init())
	}
	return m, cmd
}

func (m *rootModel) View() tea.View {
	top := m.router.top()
	if top == nil {
		v := tea.NewView(m.ctx.styles.app.Render(m.ctx.styles.error.Render("no screen")))
		v.AltScreen = true
		return v
	}
	state := top.ShellState()
	state.Body = top.View()
	v := tea.NewView(renderShell(m.ctx.styles, state))
	v.AltScreen = true
	return v
}

func fetchPricesCitiesCmd(runner *syncflows.Runner) tea.Cmd {
	return func() tea.Msg {
		cities, err := runner.PricesFetchCities(context.Background())
		return cityOptionsMsg{cities: cities, err: err}
	}
}

func waitForEventCmd(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-events }
}

func resultColumnWidth(name string) int {
	switch name {
	case "Date":
		return 10
	case "City":
		return 12
	case "Municipality":
		return 14
	case "Postal":
		return 8
	case "PostalArea":
		return 14
	case "Neighborhood":
		return 16
	case "Address":
		return 34
	case "Price":
		return 10
	case "EUR/m2":
		return 8
	case "Area":
		return 7
	case "Type":
		return 8
	case "Category":
		return 10
	case "Year":
		return 6
	case "Floor":
		return 6
	case "Elev":
		return 5
	case "Condition":
		return 10
	case "Energy":
		return 8
	case "Plot":
		return 10
	case "Period":
		return 8
	default:
		return 12
	}
}

func safeIndex(idx int, maxLen int) int {
	if idx < 0 || idx >= maxLen {
		return 0
	}
	return idx
}

func breadcrumbForActions(sub subsystem) string {
	return "Subsystems > " + sub.Title + " > Actions"
}

func helpDefault() string {
	return "Enter select/run • Esc back • q quit • Ctrl+C cancel active job"
}

func formatActionPrompt(action action, step int) string {
	return fmt.Sprintf("Enter %s (%d/%d) (Esc to cancel)", action.Prompts[step], step+1, len(action.Prompts))
}
