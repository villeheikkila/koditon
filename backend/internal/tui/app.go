package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"koditon-go/internal/syncflows"
)

const appLogo = `
 _  __         _ _ _
| |/ /___   __| (_) |_ ___  _ __
| ' // _ \ / _  | | __/ _ \| '_ \
| . \ (_) | (_| | | || (_) | | | |
|_|\_\___/ \__,_|_|\__\___/|_| |_|
`

type navStage string

const (
	stageSubsystem navStage = "subsystem"
	stageAction    navStage = "action"
)

type runFinishedMsg struct {
	actionTitle string
	result      actionResult
	err         error
	duration    time.Duration
}

type runProgressMsg struct {
	message string
	current int
	total   int
}

type cityOptionsMsg struct {
	cities []string
	err    error
}

type navItem struct {
	idx   int
	title string
	desc  string
}

func (i navItem) FilterValue() string { return i.title }
func (i navItem) Title() string       { return i.title }
func (i navItem) Description() string { return i.desc }

type navDelegate struct {
	styles styles
}

func (d navDelegate) Height() int                             { return 3 }
func (d navDelegate) Spacing() int                            { return 0 }
func (d navDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d navDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(navItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	title := "  " + it.title
	if selected {
		_, _ = fmt.Fprint(w, d.styles.selected.Render(title)+"\n"+d.styles.selectedDescription.Render("    "+it.desc)+"\n")
		return
	}
	_, _ = fmt.Fprint(w, d.styles.normal.Render(title)+"\n"+d.styles.description.Render("    "+it.desc)+"\n")
}

type model struct {
	runner           *syncflows.Runner
	subsystems       []subsystem
	subsystemList    list.Model
	actionList       list.Model
	navStage         navStage
	showJobOutput    bool
	input            textinput.Model
	awaitingCity     bool
	cityLoading      bool
	cityList         list.Model
	awaitingInput    bool
	inputAction      action
	inputPromptStep  int
	inputValues      []string
	running          bool
	currentTitle     string
	lastOutput       string
	lastError        string
	lastDuration     time.Duration
	progressCurrent  int
	progressTotal    int
	progressMessage  string
	progressLines    []string
	activity         viewport.Model
	followOutput     bool
	progressBar      progress.Model
	resultData       *actionTable
	resultTable      table.Model
	hasResultTable   bool
	resultTableTitle string
	resultTableCount int
	events           chan tea.Msg
	spinner          spinner.Model
	width            int
	height           int
	styles           styles
}

func NewModel(runner *syncflows.Runner) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "input"
	ti.CharLimit = 256
	ti.Width = 48
	ti.Prompt = "> "
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sts := defaultStyles()
	systems := buildSubsystems()
	items := make([]list.Item, 0, len(systems))
	for i, s := range systems {
		items = append(items, navItem{idx: i, title: s.Title, desc: s.Description})
	}
	delegate := navDelegate{styles: sts}
	subList := list.New(items, delegate, 48, 20)
	subList.Title = "Subsystems"
	subList.SetShowStatusBar(false)
	subList.SetFilteringEnabled(false)
	subList.SetShowPagination(false)
	subList.Styles.Title = sts.title
	subList.Styles.NoItems = sts.muted
	actList := list.New([]list.Item{}, delegate, 56, 20)
	actList.Title = "Actions"
	actList.SetShowStatusBar(false)
	actList.SetFilteringEnabled(false)
	actList.SetShowPagination(false)
	actList.Styles.Title = sts.title
	actList.Styles.NoItems = sts.muted
	cityList := list.New([]list.Item{}, delegate, 56, 18)
	cityList.Title = "Select City"
	cityList.SetShowStatusBar(true)
	cityList.SetFilteringEnabled(true)
	cityList.SetShowPagination(true)
	cityList.Styles.Title = sts.title
	cityList.Styles.NoItems = sts.muted
	vp := viewport.New(80, 20)
	vp.SetContent(sts.muted.Render("Waiting for updates..."))
	pb := progress.New(progress.WithDefaultGradient())
	pb.Width = 34
	m := model{
		runner:        runner,
		subsystems:    systems,
		subsystemList: subList,
		actionList:    actList,
		cityList:      cityList,
		navStage:      stageSubsystem,
		input:         ti,
		spinner:       sp,
		styles:        sts,
		progressLines: make([]string, 0, 300),
		activity:      vp,
		followOutput:  true,
		progressBar:   pb,
		inputValues:   make([]string, 0, 2),
		resultTable:   table.New(),
	}
	m.refreshActionList()
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.running {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			if m.showJobOutput {
				switch msg.String() {
				case "end", "G":
					m.followOutput = true
				default:
					m.followOutput = false
				}
				var cmd tea.Cmd
				m.activity, cmd = m.activity.Update(msg)
				return m, cmd
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		case runProgressMsg:
			m.progressMessage = msg.message
			if msg.total > 0 {
				m.progressCurrent = msg.current
				m.progressTotal = msg.total
			}
			m.pushProgressLine(msg.message)
			m.refreshActivity()
			cmds := []tea.Cmd{waitForEventCmd(m.events)}
			if m.progressTotal > 0 {
				cmds = append(cmds, m.progressBar.SetPercent(float64(m.progressCurrent)/float64(m.progressTotal)))
			}
			return m, tea.Batch(cmds...)
		case runFinishedMsg:
			m.running = false
			m.showJobOutput = true
			m.lastDuration = msg.duration
			m.lastOutput = msg.result.Output
			m.resultData = msg.result.Table
			m.hasResultTable = msg.result.Table != nil
			if m.hasResultTable {
				m.resultTableTitle = msg.result.Table.Title
				m.resultTableCount = len(msg.result.Table.Rows)
				m.rebuildResultTable()
			}
			if msg.err != nil {
				m.lastError = msg.err.Error()
				m.pushProgressLine("Finished with errors")
			} else {
				m.lastError = ""
				m.pushProgressLine("Finished successfully")
			}
			m.refreshActivity()
			return m, nil
		case progress.FrameMsg:
			p, cmd := m.progressBar.Update(msg)
			m.progressBar = p.(progress.Model)
			return m, cmd
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case cityOptionsMsg:
		m.cityLoading = false
		if msg.err != nil {
			m.awaitingCity = false
			m.lastError = fmt.Sprintf("load cities: %v", msg.err)
			return m, nil
		}
		items := make([]list.Item, 0, len(msg.cities))
		for i, city := range msg.cities {
			items = append(items, navItem{idx: i, title: city, desc: "prices city"})
		}
		m.cityList.SetItems(items)
		if len(items) > 0 {
			m.cityList.Select(0)
		}
		return m, nil
	case tea.KeyMsg:
		if m.awaitingCity {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.awaitingCity = false
				m.cityLoading = false
				m.inputPromptStep = 0
				m.inputValues = m.inputValues[:0]
				return m, nil
			case "enter":
				if m.cityLoading {
					return m, nil
				}
				item, ok := m.cityList.SelectedItem().(navItem)
				if !ok || strings.TrimSpace(item.title) == "" {
					m.lastError = "select a city"
					return m, nil
				}
				m.awaitingCity = false
				m.cityLoading = false
				m.inputValues = append(m.inputValues, item.title)
				if len(m.inputAction.Prompts) <= 1 {
					inputs := append([]string(nil), m.inputValues...)
					m.inputPromptStep = 0
					m.inputValues = m.inputValues[:0]
					return m.startAction(m.inputAction, inputs)
				}
				m.awaitingInput = true
				m.inputPromptStep = 1
				m.input.Placeholder = m.inputAction.Prompts[m.inputPromptStep]
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			default:
				if m.cityLoading {
					return m, nil
				}
				var cmd tea.Cmd
				m.cityList, cmd = m.cityList.Update(msg)
				return m, cmd
			}
		}
		if m.awaitingInput {
			switch msg.String() {
			case "esc":
				m.awaitingInput = false
				m.inputPromptStep = 0
				m.inputValues = m.inputValues[:0]
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.input.Value())
				if value == "" {
					m.lastError = "input cannot be empty"
					return m, nil
				}
				m.inputValues = append(m.inputValues, value)
				m.input.SetValue("")
				nextStep := m.inputPromptStep + 1
				if nextStep < len(m.inputAction.Prompts) {
					m.inputPromptStep = nextStep
					m.input.Placeholder = m.inputAction.Prompts[m.inputPromptStep]
					return m, textinput.Blink
				}
				inputs := append([]string(nil), m.inputValues...)
				m.awaitingInput = false
				m.inputPromptStep = 0
				m.inputValues = m.inputValues[:0]
				m.input.Blur()
				return m.startAction(m.inputAction, inputs)
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "left", "h", "backspace":
			if m.navStage == stageAction {
				if m.showJobOutput && !m.running {
					m.showJobOutput = false
					m.hasResultTable = false
					m.resultData = nil
					return m, nil
				}
				m.navStage = stageSubsystem
				m.showJobOutput = false
				m.hasResultTable = false
				m.resultData = nil
				return m, nil
			}
		case "enter", "right", "l":
			if m.navStage == stageSubsystem {
				m.navStage = stageAction
				m.showJobOutput = false
				m.refreshActionList()
				return m, nil
			}
			if m.showJobOutput {
				return m, nil
			}
			a := m.selectedAction()
			if len(a.Prompts) > 0 {
				if a.UseCityPicker {
					return m.startCityPicker(a)
				}
				m.awaitingInput = true
				m.inputAction = a
				m.inputPromptStep = 0
				m.inputValues = m.inputValues[:0]
				m.input.Placeholder = a.Prompts[0]
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			}
			return m.startAction(a, nil)
		}
	}
	if m.showJobOutput && !m.running {
		if m.hasResultTable {
			var cmd tea.Cmd
			m.resultTable, cmd = m.resultTable.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.activity, cmd = m.activity.Update(msg)
		return m, cmd
	}
	if m.navStage == stageSubsystem {
		var cmd tea.Cmd
		m.subsystemList, cmd = m.subsystemList.Update(msg)
		m.refreshActionList()
		return m, cmd
	}
	var cmd tea.Cmd
	m.actionList, cmd = m.actionList.Update(msg)
	return m, cmd
}

func (m model) selectedSubsystemIndex() int {
	item, ok := m.subsystemList.SelectedItem().(navItem)
	if !ok || item.idx < 0 || item.idx >= len(m.subsystems) {
		return 0
	}
	return item.idx
}

func (m model) selectedAction() action {
	sub := m.subsystems[m.selectedSubsystemIndex()]
	item, ok := m.actionList.SelectedItem().(navItem)
	if !ok || item.idx < 0 || item.idx >= len(sub.Actions) {
		return sub.Actions[0]
	}
	return sub.Actions[item.idx]
}

func (m *model) refreshActionList() {
	sub := m.subsystems[m.selectedSubsystemIndex()]
	items := make([]list.Item, 0, len(sub.Actions))
	for i, a := range sub.Actions {
		items = append(items, navItem{idx: i, title: a.Title, desc: a.Description})
	}
	m.actionList.SetItems(items)
	m.actionList.Title = sub.Title + " Actions"
	if len(items) > 0 && m.actionList.Index() >= len(items) {
		m.actionList.Select(0)
	}
}

func (m model) startCityPicker(a action) (tea.Model, tea.Cmd) {
	m.awaitingCity = true
	m.cityLoading = true
	m.awaitingInput = false
	m.inputAction = a
	m.inputPromptStep = 0
	m.inputValues = m.inputValues[:0]
	m.cityList.Title = "Select City"
	m.cityList.SetItems([]list.Item{})
	m.cityList.ResetFilter()
	return m, fetchPricesCitiesCmd(m.runner)
}

func (m model) startAction(a action, inputs []string) (tea.Model, tea.Cmd) {
	m.running = true
	m.showJobOutput = true
	m.currentTitle = a.Title
	m.lastOutput = ""
	m.lastError = ""
	m.progressCurrent = 0
	m.progressTotal = 0
	m.progressMessage = "Starting..."
	m.progressLines = m.progressLines[:0]
	m.events = make(chan tea.Msg, 256)
	m.followOutput = true
	m.hasResultTable = false
	m.resultData = nil
	m.refreshActivity()
	return m, tea.Batch(m.spinner.Tick, runActionCmd(m.runner, a, inputs, m.events), waitForEventCmd(m.events), m.progressBar.SetPercent(0))
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

func runActionCmd(runner *syncflows.Runner, a action, inputs []string, events chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			ctx := context.Background()
			start := time.Now()
			report := func(update progressUpdate) {
				events <- runProgressMsg{message: update.Message, current: update.Current, total: update.Total}
			}
			result, err := a.Run(ctx, runner, inputs, report)
			events <- runFinishedMsg{actionTitle: a.Title, result: result, err: err, duration: time.Since(start)}
		}()
		return nil
	}
}

func (m *model) pushProgressLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	colored := m.colorizeProgressLine(line)
	m.progressLines = append(m.progressLines, m.styles.muted.Render("["+timestamp+"]")+" "+colored)
}

func (m *model) refreshActivity() {
	if len(m.progressLines) == 0 {
		m.activity.SetContent(m.styles.muted.Render("Waiting for updates..."))
		return
	}
	m.activity.SetContent(strings.Join(m.progressLines, "\n"))
	if m.followOutput {
		m.activity.GotoBottom()
	}
}

func (m *model) rebuildResultTable() {
	if m.resultData == nil {
		m.hasResultTable = false
		return
	}
	columns := make([]table.Column, 0, len(m.resultData.Columns))
	for _, col := range m.resultData.Columns {
		columns = append(columns, table.Column{Title: col, Width: resultColumnWidth(col)})
	}
	rows := make([]table.Row, 0, len(m.resultData.Rows))
	for _, row := range m.resultData.Rows {
		rows = append(rows, table.Row(row))
	}
	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(max(8, m.height-18)))
	t.SetStyles(m.tableStyles())
	m.resultTable = t
	m.hasResultTable = true
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

func (m model) tableStyles() table.Styles {
	ts := table.DefaultStyles()
	ts.Header = ts.Header.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("24")).Bold(true)
	ts.Cell = ts.Cell.Foreground(lipgloss.Color("252"))
	ts.Selected = ts.Selected.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("33")).Bold(true)
	return ts
}

func (m model) colorizeProgressLine(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"), strings.Contains(lower, "failed"), strings.Contains(lower, "warn"):
		return m.styles.error.Render(line)
	case strings.Contains(lower, "done "), strings.Contains(lower, "completed"), strings.Contains(lower, "success"), strings.Contains(lower, "synced"):
		return m.styles.success.Render(line)
	case strings.Contains(lower, "syncing"), strings.Contains(lower, "running"), strings.Contains(lower, "fetching"), strings.Contains(lower, "progress"), strings.Contains(lower, "searching"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(line)
	default:
		return m.styles.normal.Render(line)
	}
}

func (m *model) resize() {
	contentH := max(16, m.height-8)
	listW := max(56, m.width-6)
	m.subsystemList.SetSize(listW, contentH)
	m.actionList.SetSize(listW, contentH)
	m.cityList.SetSize(max(56, m.width-10), max(10, m.height-14))
	m.activity.Width = max(56, m.width-10)
	m.activity.Height = max(10, m.height-18)
	m.progressBar.Width = max(24, m.width-14)
	if m.hasResultTable {
		m.resultTable.SetHeight(max(8, m.height-18))
		m.resultTable.SetWidth(max(56, m.width-8))
	}
}

func (m model) View() string {
	if m.awaitingCity {
		var body string
		if m.cityLoading {
			body = m.styles.muted.Render("Loading cities...")
		} else {
			body = m.cityList.View()
		}
		panel := m.styles.panel.Width(max(56, min(110, m.width-8))).Render(
			m.styles.title.Render(m.inputAction.Title) + "\n" +
				m.styles.description.Render(m.inputAction.Description) + "\n\n" +
				m.styles.inputLabel.Render("Select city (Enter) • Fuzzy filter (/) • Esc cancel") + "\n" +
				body,
		)
		return m.styles.app.Render(panel)
	}
	if m.awaitingInput {
		step := fmt.Sprintf("(%d/%d)", m.inputPromptStep+1, len(m.inputAction.Prompts))
		prompt := m.inputAction.Prompts[m.inputPromptStep]
		inputPanel := m.styles.panel.Width(max(56, min(90, m.width-8))).Render(m.styles.title.Render(m.inputAction.Title) + "\n" + m.styles.description.Render(m.inputAction.Description) + "\n\n" + m.styles.inputLabel.Render("Enter "+prompt+" "+step+" (Esc to cancel)") + "\n" + m.input.View())
		return m.styles.app.Render(inputPanel)
	}
	header := m.styles.title.Render("Koditon Manual Sync")
	help := m.styles.help.Render("Enter select/run • Esc back • q quit")
	breadcrumb := m.styles.breadcrumb.Render(m.currentBreadcrumb())
	if m.navStage == stageSubsystem {
		return m.styles.app.Render(header + "\n" + breadcrumb + "\n" + help + "\n\n" + m.renderMainScreen())
	}
	if m.showJobOutput || m.running {
		return m.styles.app.Render(header + "\n" + breadcrumb + "\n" + help + "\n\n" + m.renderActionPanel())
	}
	return m.styles.app.Render(header + "\n" + breadcrumb + "\n" + help + "\n\n" + m.styles.panel.Width(max(56, m.width-6)).Render(m.actionList.View()))
}

func (m model) currentBreadcrumb() string {
	if m.navStage == stageSubsystem {
		return "Subsystems"
	}
	sub := m.subsystems[m.selectedSubsystemIndex()]
	return "Subsystems > " + sub.Title + " > Actions"
}

func (m model) renderMainScreen() string {
	sub := m.subsystems[m.selectedSubsystemIndex()]
	var b strings.Builder
	b.WriteString(m.styles.logo.Render(appLogo))
	b.WriteString("\n")
	b.WriteString(m.subsystemList.View())
	b.WriteString("\n")
	b.WriteString(m.styles.progressLabel.Render("Selected Subsystem"))
	b.WriteString("\n")
	b.WriteString(m.styles.selected.Render(sub.Title))
	b.WriteString("\n")
	b.WriteString(m.styles.description.Render(sub.Description))
	b.WriteString("\n")
	b.WriteString(m.styles.muted.Render("Press Enter to open actions."))
	return b.String()
}

func (m model) renderActionPanel() string {
	var b strings.Builder
	if m.running {
		b.WriteString(m.styles.running.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.currentTitle)))
		b.WriteString("\n")
		if m.progressTotal > 0 {
			b.WriteString(m.styles.progressLabel.Render(fmt.Sprintf("Progress %d/%d", m.progressCurrent, m.progressTotal)))
			b.WriteString("\n")
			b.WriteString(m.progressBar.ViewAs(float64(m.progressCurrent) / float64(m.progressTotal)))
			b.WriteString("\n")
		}
		if m.progressMessage != "" {
			b.WriteString(m.styles.muted.Render("Current: " + m.progressMessage))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if m.hasResultTable && !m.running {
		b.WriteString(m.styles.progressLabel.Render(fmt.Sprintf("%s (%d rows)", m.resultTableTitle, m.resultTableCount)))
		b.WriteString("\n")
		b.WriteString(m.styles.muted.Render("Use Up/Down/PgUp/PgDn to scroll rows"))
		b.WriteString("\n")
		b.WriteString(m.resultTable.View())
	} else {
		b.WriteString(m.styles.progressLabel.Render("Activity"))
		b.WriteString("\n")
		b.WriteString(m.activity.View())
	}
	if !m.running && (m.lastOutput != "" || m.lastError != "") {
		b.WriteString("\n\n")
		b.WriteString(m.styles.progressLabel.Render("Last Result"))
		b.WriteString("\n")
		if m.lastOutput != "" {
			b.WriteString(m.styles.success.Render("Output: "))
			b.WriteString(m.styles.normal.Render(m.lastOutput))
			b.WriteString("\n")
		}
		if m.lastError != "" {
			b.WriteString(m.styles.error.Render("Error: "))
			b.WriteString(m.styles.error.Render(m.lastError))
			b.WriteString("\n")
		}
		if m.lastDuration > 0 {
			b.WriteString(m.styles.muted.Render("Duration: " + m.lastDuration.Round(time.Millisecond).String()))
		}
	}
	return b.String()
}
