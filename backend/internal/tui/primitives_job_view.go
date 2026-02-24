package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type jobView struct {
	styles           styles
	spinner          spinner.Model
	progressBar      progress.Model
	activity         viewport.Model
	followOutput     bool
	progressLines    []string
	progressCurrent  int
	progressTotal    int
	progressMessage  string
	lastOutput       string
	lastError        string
	lastDuration     time.Duration
	resultData       *actionTable
	resultTable      table.Model
	hasResultTable   bool
	resultTableTitle string
	resultTableCount int
	currentTitle     string
}

func newJobView(st styles) jobView {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	pb := progress.New(progress.WithDefaultBlend())
	pb.SetWidth(34)
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.SetContent(st.muted.Render("Waiting for updates..."))
	return jobView{styles: st, spinner: sp, progressBar: pb, activity: vp, followOutput: true, progressLines: make([]string, 0, 300), resultTable: table.New()}
}

func (j *jobView) Start(title string) tea.Cmd {
	j.currentTitle = title
	j.lastOutput = ""
	j.lastError = ""
	j.progressCurrent = 0
	j.progressTotal = 0
	j.progressMessage = "Starting..."
	j.progressLines = j.progressLines[:0]
	j.followOutput = true
	j.hasResultTable = false
	j.resultData = nil
	j.refreshActivity()
	return tea.Batch(j.spinner.Tick, j.progressBar.SetPercent(0))
}

func (j *jobView) OnProgress(msg runProgressMsg) tea.Cmd {
	j.progressMessage = msg.message
	if msg.total > 0 {
		j.progressCurrent = msg.current
		j.progressTotal = msg.total
	}
	j.pushProgressLine(msg.message)
	j.refreshActivity()
	if j.progressTotal > 0 {
		return j.progressBar.SetPercent(float64(j.progressCurrent) / float64(j.progressTotal))
	}
	return nil
}

func (j *jobView) OnFinished(msg runFinishedMsg) {
	j.lastDuration = msg.duration
	j.lastOutput = msg.result.Output
	j.resultData = msg.result.Table
	j.hasResultTable = msg.result.Table != nil
	if j.hasResultTable {
		j.resultTableTitle = msg.result.Table.Title
		j.resultTableCount = len(msg.result.Table.Rows)
		j.rebuildResultTable()
	}
	if msg.err != nil {
		j.lastError = formatErrMultiline(msg.err)
		j.pushProgressLine("Finished with errors")
		j.refreshActivity()
		return
	}
	j.lastError = ""
	j.pushProgressLine("Finished successfully")
	j.refreshActivity()
}

func (j *jobView) Resize(width int, height int) {
	j.activity.SetWidth(max(56, width-10))
	j.activity.SetHeight(max(10, height-18))
	j.progressBar.SetWidth(max(24, width-14))
	if j.hasResultTable {
		j.resultTable.SetHeight(max(8, height-18))
		j.resultTable.SetWidth(max(56, width-8))
	}
}

func (j *jobView) UpdateRunning(msg tea.Msg) tea.Cmd {
	switch typed := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		j.spinner, cmd = j.spinner.Update(typed)
		return cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		j.progressBar, cmd = j.progressBar.Update(typed)
		return cmd
	}
	return nil
}

func (j *jobView) UpdateFinished(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "end", "G":
			j.followOutput = true
		default:
			j.followOutput = false
		}
	}
	if j.hasResultTable {
		var cmd tea.Cmd
		j.resultTable, cmd = j.resultTable.Update(msg)
		return cmd
	}
	var cmd tea.Cmd
	j.activity, cmd = j.activity.Update(msg)
	return cmd
}

func (j *jobView) View(running bool) string {
	var b strings.Builder
	if running {
		b.WriteString(j.styles.running.Render(fmt.Sprintf("%s %s", j.spinner.View(), j.currentTitle)))
		b.WriteString("\n")
		if j.progressTotal > 0 {
			b.WriteString(j.styles.progressLabel.Render(fmt.Sprintf("Progress %d/%d", j.progressCurrent, j.progressTotal)))
			b.WriteString("\n")
			b.WriteString(j.progressBar.ViewAs(float64(j.progressCurrent) / float64(j.progressTotal)))
			b.WriteString("\n")
		}
		if j.progressMessage != "" {
			b.WriteString(j.styles.muted.Render("Current: " + j.progressMessage))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if j.hasResultTable && !running {
		b.WriteString(j.styles.progressLabel.Render(fmt.Sprintf("%s (%d rows)", j.resultTableTitle, j.resultTableCount)))
		b.WriteString("\n")
		b.WriteString(j.styles.muted.Render("Use Up/Down/PgUp/PgDn to scroll rows"))
		b.WriteString("\n")
		b.WriteString(j.resultTable.View())
	} else {
		b.WriteString(j.styles.progressLabel.Render("Activity"))
		b.WriteString("\n")
		b.WriteString(j.activity.View())
	}
	if !running && (j.lastOutput != "" || j.lastError != "") {
		b.WriteString("\n\n")
		b.WriteString(j.styles.progressLabel.Render("Last Result"))
		b.WriteString("\n")
		if j.lastOutput != "" {
			b.WriteString(j.styles.success.Render("Output: "))
			b.WriteString(j.styles.normal.Render(j.lastOutput))
			b.WriteString("\n")
		}
		if j.lastError != "" {
			b.WriteString(j.styles.error.Render("Error: "))
			b.WriteString(j.styles.error.Render(j.lastError))
			b.WriteString("\n")
		}
		if j.lastDuration > 0 {
			b.WriteString(j.styles.muted.Render("Duration: " + j.lastDuration.Round(time.Millisecond).String()))
		}
	}
	return b.String()
}

func formatErrMultiline(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	for i, line := range strings.Split(err.Error(), "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		parts := strings.Split(line, ": ")
		for j, part := range parts {
			if j > 0 {
				b.WriteString("\n")
				for range j {
					b.WriteString("  ")
				}
			}
			b.WriteString(part)
		}
	}
	return b.String()
}

func (j *jobView) pushProgressLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	lines := strings.Split(line, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	wrapped := make([]string, 0, len(lines))
	contentWidth := max(24, j.activity.Width()-14)
	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			wrapped = append(wrapped, "")
			continue
		}
		if strings.Contains(raw, "http://") || strings.Contains(raw, "https://") {
			wrapped = append(wrapped, raw)
			continue
		}
		wrapped = append(wrapped, wrapActivityLine(raw, contentWidth)...)
	}
	for i, l := range wrapped {
		prefix := "          "
		if i == 0 {
			prefix = j.styles.muted.Render("[" + timestamp + "]")
		}
		colored := j.colorizeProgressLine(l)
		j.progressLines = append(j.progressLines, prefix+" "+colored)
	}
}

func (j *jobView) refreshActivity() {
	if len(j.progressLines) == 0 {
		j.activity.SetContent(j.styles.muted.Render("Waiting for updates..."))
		return
	}
	j.activity.SetContent(strings.Join(j.progressLines, "\n"))
	if j.followOutput {
		j.activity.GotoBottom()
	}
}

func (j *jobView) rebuildResultTable() {
	if j.resultData == nil {
		j.hasResultTable = false
		return
	}
	columns := make([]table.Column, 0, len(j.resultData.Columns))
	for _, col := range j.resultData.Columns {
		columns = append(columns, table.Column{Title: col, Width: resultColumnWidth(col)})
	}
	rows := make([]table.Row, 0, len(j.resultData.Rows))
	for _, row := range j.resultData.Rows {
		rows = append(rows, table.Row(row))
	}
	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(jobTableStyles())
	j.resultTable = t
	j.hasResultTable = true
}

func jobTableStyles() table.Styles {
	ts := table.DefaultStyles()
	ts.Header = ts.Header.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("24")).Bold(true)
	ts.Cell = ts.Cell.Foreground(lipgloss.Color("252"))
	ts.Selected = ts.Selected.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("33")).Bold(true)
	return ts
}

func (j *jobView) colorizeProgressLine(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"), strings.Contains(lower, "failed"), strings.Contains(lower, "warn"):
		return j.styles.error.Render(line)
	case strings.Contains(lower, "done "), strings.Contains(lower, "completed"), strings.Contains(lower, "success"), strings.Contains(lower, "synced"):
		return j.styles.success.Render(line)
	case strings.Contains(lower, "syncing"), strings.Contains(lower, "running"), strings.Contains(lower, "fetching"), strings.Contains(lower, "progress"), strings.Contains(lower, "searching"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(line)
	default:
		return j.styles.normal.Render(line)
	}
}

func wrapActivityLine(line string, width int) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || width <= 0 || utf8.RuneCountInString(trimmed) <= width {
		return []string{trimmed}
	}
	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return []string{trimmed}
	}
	lines := make([]string, 0, 4)
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if utf8.RuneCountInString(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		if utf8.RuneCountInString(word) > width {
			lines = append(lines, splitLongWord(word, width)...)
			current = ""
			continue
		}
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitLongWord(word string, width int) []string {
	if width <= 0 {
		return []string{word}
	}
	runes := []rune(word)
	if len(runes) <= width {
		return []string{word}
	}
	lines := make([]string, 0, (len(runes)/width)+1)
	for len(runes) > width {
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	return lines
}
