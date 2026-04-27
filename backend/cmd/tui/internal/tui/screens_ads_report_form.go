package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"koditon/internal/domain/ads"
)

type adsReportFormScreen struct {
	ctx        *appContext
	action     action
	form       formPrimitive
	values     []string
	errorText  string
	width      int
	height     int
	breadcrumb string
}

func newAdsReportFormScreen(ctx *appContext, action action, values []string, breadcrumb string) Screen {
	formValues := normalizeAdsFormValues(values)
	fields := []formField{
		newTextFormField("query", "Query (id/url/address/postal/city)", "helsinki 00100"),
		newChoiceFormField("source", "Source", []formChoice{{Label: "All", Value: "all"}, {Label: "Shortcut", Value: "shortcut"}, {Label: "Frontdoor", Value: "frontdoor"}}, choiceIndexByValue("all", "shortcut", "frontdoor", formValues[1])),
		newChoiceFormField("kind", "Kind", []formChoice{{Label: "All", Value: "all"}, {Label: "Ad", Value: "ad"}, {Label: "Announcement", Value: "announcement"}, {Label: "Building", Value: "building"}}, choiceIndexByValue("all", "ad", "announcement", "building", formValues[2])),
		newTextFormField("min_price", "Min price", "100000"),
		newTextFormField("max_price", "Max price", "400000"),
		newTextFormField("min_area", "Min area (m2)", "30"),
		newTextFormField("max_area", "Max area (m2)", "120"),
		newTextFormField("city", "City contains", "Helsinki"),
		newTextFormField("postal", "Postal contains", "00100"),
		newChoiceFormField("sort", "Sort", []formChoice{{Label: "Newest first", Value: "seen_desc"}, {Label: "Price ascending", Value: "price_asc"}, {Label: "Price descending", Value: "price_desc"}, {Label: "Area ascending", Value: "area_asc"}, {Label: "Area descending", Value: "area_desc"}}, choiceIndexByValue("seen_desc", "price_asc", "price_desc", "area_asc", "area_desc", formValues[9])),
		newChoiceFormField("page_size", "Page size", []formChoice{{Label: "25", Value: "25"}, {Label: "50", Value: "50"}, {Label: "100", Value: "100"}}, choiceIndexByValue("25", "50", "100", formValues[10])),
	}
	form := newFormPrimitive(ctx.styles, fields)
	form.fields[0].Input.SetValue(formValues[0])
	form.fields[3].Input.SetValue(formValues[3])
	form.fields[4].Input.SetValue(formValues[4])
	form.fields[5].Input.SetValue(formValues[5])
	form.fields[6].Input.SetValue(formValues[6])
	form.fields[7].Input.SetValue(formValues[7])
	form.fields[8].Input.SetValue(formValues[8])
	return &adsReportFormScreen{ctx: ctx, action: action, form: form, values: formValues, breadcrumb: breadcrumb}
}

func (s *adsReportFormScreen) Key() string {
	return "ads-report-form"
}

func (s *adsReportFormScreen) Init() tea.Cmd {
	return nil
}

func (s *adsReportFormScreen) Resize(width int, height int) {
	s.width = width
	s.height = height
	s.form.Resize(width - 10)
}

func (s *adsReportFormScreen) Update(msg tea.Msg, nav Navigator) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "esc":
			nav.Pop()
			return nil
		case "enter":
			params, formValues, err := s.buildParams()
			if err != nil {
				s.errorText = err.Error()
				return nil
			}
			s.errorText = ""
			s.values = formValues
			nav.Replace(newAdsReportBrowserScreen(s.ctx, s.action, params, formValues, s.breadcrumb))
			return nil
		}
	}
	return s.form.Update(msg)
}

func (s *adsReportFormScreen) View() string {
	content := s.ctx.styles.title.Render(s.action.Title) + "\n" + s.ctx.styles.description.Render("Unified Shortcut + Frontdoor ads report search") + "\n\n" + s.form.View() + "\n\n" + s.ctx.styles.muted.Render("Enter search • Tab/j/k navigate • h/l change choices • Esc back")
	if s.errorText != "" {
		content += "\n" + s.ctx.styles.error.Render(s.errorText)
	}
	return s.ctx.styles.panel.Width(max(60, min(120, s.width-8))).Render(content)
}

func (s *adsReportFormScreen) ShellState() shellState {
	return shellState{Title: "Koditon Manual Sync", Breadcrumb: s.breadcrumb, Help: "Enter search • Tab/j/k fields • h/l choices • Esc back • q quit"}
}

func (s *adsReportFormScreen) buildParams() (ads.SearchParams, []string, error) {
	values := s.form.Values()
	query := strings.TrimSpace(values["query"])
	source := strings.TrimSpace(values["source"])
	kind := strings.TrimSpace(values["kind"])
	minPrice, err := parseOptionalInt64(strings.TrimSpace(values["min_price"]))
	if err != nil {
		return ads.SearchParams{}, nil, fmt.Errorf("min price must be numeric")
	}
	maxPrice, err := parseOptionalInt64(strings.TrimSpace(values["max_price"]))
	if err != nil {
		return ads.SearchParams{}, nil, fmt.Errorf("max price must be numeric")
	}
	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return ads.SearchParams{}, nil, fmt.Errorf("min price cannot exceed max price")
	}
	minArea, err := parseOptionalFloat64(strings.TrimSpace(values["min_area"]))
	if err != nil {
		return ads.SearchParams{}, nil, fmt.Errorf("min area must be numeric")
	}
	maxArea, err := parseOptionalFloat64(strings.TrimSpace(values["max_area"]))
	if err != nil {
		return ads.SearchParams{}, nil, fmt.Errorf("max area must be numeric")
	}
	if minArea != nil && maxArea != nil && *minArea > *maxArea {
		return ads.SearchParams{}, nil, fmt.Errorf("min area cannot exceed max area")
	}
	city := strings.TrimSpace(values["city"])
	postal := strings.TrimSpace(values["postal"])
	sort := strings.TrimSpace(values["sort"])
	pageSize, err := strconv.Atoi(strings.TrimSpace(values["page_size"]))
	if err != nil {
		return ads.SearchParams{}, nil, fmt.Errorf("page size must be 25, 50 or 100")
	}
	if pageSize != 25 && pageSize != 50 && pageSize != 100 {
		return ads.SearchParams{}, nil, fmt.Errorf("page size must be 25, 50 or 100")
	}
	params := ads.SearchParams{Query: query, Source: source, Kind: kind, MinPrice: minPrice, MaxPrice: maxPrice, MinArea: minArea, MaxArea: maxArea, City: city, Postal: postal, Sort: sort, Page: 1, PageSize: int32(pageSize)}
	formValues := []string{query, source, kind, int64ToString(minPrice), int64ToString(maxPrice), float64ToString(minArea), float64ToString(maxArea), city, postal, sort, strconv.Itoa(pageSize), "1"}
	return params, formValues, nil
}

func normalizeAdsFormValues(values []string) []string {
	defaults := []string{"", "all", "all", "", "", "", "", "", "", "seen_desc", "50", "1"}
	if len(values) == 0 {
		return defaults
	}
	out := append([]string(nil), defaults...)
	for i := 0; i < len(out) && i < len(values); i++ {
		if strings.TrimSpace(values[i]) != "" {
			out[i] = strings.TrimSpace(values[i])
		}
	}
	return out
}

func choiceIndexByValue(options ...string) int {
	if len(options) == 0 {
		return 0
	}
	value := options[len(options)-1]
	for idx := 0; idx < len(options)-1; idx++ {
		if options[idx] == value {
			return idx
		}
	}
	return 0
}

func parseOptionalInt64(value string) (*int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func int64ToString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func float64ToString(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *value)
}
