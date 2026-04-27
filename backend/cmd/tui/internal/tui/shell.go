package tui

import "strings"

type shellState struct {
	Title      string
	Breadcrumb string
	Help       string
	Body       string
	Footer     string
}

func renderShell(st styles, state shellState) string {
	content := st.title.Render(state.Title)
	if state.Breadcrumb != "" {
		content += "\n" + st.breadcrumb.Render(state.Breadcrumb)
	}
	if state.Help != "" {
		content += "\n" + st.help.Render(state.Help)
	}
	content += "\n\n" + state.Body
	if state.Footer != "" {
		content += "\n\n" + st.muted.Render(state.Footer)
	}
	return trimLineEndSpaces(st.app.Render(content))
}

func trimLineEndSpaces(value string) string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}
