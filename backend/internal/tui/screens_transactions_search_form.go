package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type transactionsSearchFormScreen struct {
	ctx        *appContext
	action     action
	values     []string
	form       formPrimitive
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newTransactionsSearchFormScreen(ctx *appContext, action action, values []string, breadcrumb string) Screen {
	fields := []formField{
		newTextFormField("query", "Postal code, neighborhood or address", "00170"),
		newTextFormField("min_area", "Min area (m2)", "35"),
		newTextFormField("max_area", "Max area (m2)", "45"),
		newChoiceFormField("sort", "Sort", []formChoice{
			{Label: "Price ascending (cheapest first)", Value: "price_asc"},
			{Label: "Price descending", Value: "price_desc"},
			{Label: "Date newest first", Value: "date_desc"},
			{Label: "Date oldest first", Value: "date_asc"},
		}, 0),
		newTextFormField("limit", "Max rows", "500"),
	}
	form := newFormPrimitive(ctx.styles, fields)
	return &transactionsSearchFormScreen{
		ctx:        ctx,
		action:     action,
		values:     append([]string(nil), values...),
		form:       form,
		breadcrumb: breadcrumb,
	}
}

func (s *transactionsSearchFormScreen) Key() string {
	return "transactions-search-form"
}

func (s *transactionsSearchFormScreen) Init() tea.Cmd {
	return nil
}

func (s *transactionsSearchFormScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.form.Resize(width - 10)
}

func (s *transactionsSearchFormScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			nav.Pop()
			return nil
		case "enter":
			inputs, err := s.buildInputs()
			if err != nil {
				s.errorText = err.Error()
				return nil
			}
			s.errorText = ""
			nav.Replace(newJobScreen(s.ctx, s.action, inputs, s.breadcrumb))
			return nil
		}
	}
	return s.form.Update(msg)
}

func (s *transactionsSearchFormScreen) View() string {
	city := safeInput(s.values, 0)
	content := s.ctx.styles.title.Render(s.action.Title) + "\n" + s.ctx.styles.description.Render("Targeted transaction search form") + "\n\n" + s.ctx.styles.inputLabel.Render("City") + "\n" + s.ctx.styles.normal.Render("  "+city) + "\n\n" + s.form.View() + "\n\n" + s.ctx.styles.muted.Render("Enter runs search. Tab/j/k move fields. h/l changes sort mode.")
	if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	}
	return s.ctx.styles.panel.Width(max(56, min(120, s.width-8))).Render(content)
}

func (s *transactionsSearchFormScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: helpDefault()}
}

func (s *transactionsSearchFormScreen) buildInputs() ([]string, error) {
	city := safeInput(s.values, 0)
	if city == "" {
		return nil, fmt.Errorf("city is required")
	}
	values := s.form.Values()
	minArea := strings.TrimSpace(values["min_area"])
	maxArea := strings.TrimSpace(values["max_area"])
	limit := strings.TrimSpace(values["limit"])
	if minArea != "" {
		if _, err := strconv.ParseFloat(minArea, 64); err != nil {
			return nil, fmt.Errorf("min area must be numeric")
		}
	}
	if maxArea != "" {
		if _, err := strconv.ParseFloat(maxArea, 64); err != nil {
			return nil, fmt.Errorf("max area must be numeric")
		}
	}
	if minArea != "" && maxArea != "" {
		minValue, _ := strconv.ParseFloat(minArea, 64)
		maxValue, _ := strconv.ParseFloat(maxArea, 64)
		if minValue > maxValue {
			return nil, fmt.Errorf("min area cannot exceed max area")
		}
	}
	if limit != "" {
		parsedLimit, err := strconv.Atoi(limit)
		if err != nil {
			return nil, fmt.Errorf("max rows must be numeric")
		}
		if parsedLimit < 1 || parsedLimit > 5000 {
			return nil, fmt.Errorf("max rows must be between 1 and 5000")
		}
	}
	inputs := []string{
		city,
		strings.TrimSpace(values["query"]),
		minArea,
		maxArea,
		parseSortMode(values["sort"]),
		limit,
	}
	return inputs, nil
}
