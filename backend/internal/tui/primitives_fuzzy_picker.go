package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type fuzzyPicker struct {
	list    selectableList
	search  textinput.Model
	options []string
	styles  styles
}

func newFuzzyPicker(title string, st styles) fuzzyPicker {
	search := textinput.New()
	search.Placeholder = "type city name"
	search.Prompt = "Search: "
	search.CharLimit = 128
	search.Width = 42
	l := newSelectableList(title, 56, 18, st)
	l.model.SetShowPagination(true)
	return fuzzyPicker{list: l, search: search, options: make([]string, 0), styles: st}
}

func (f *fuzzyPicker) Focus() {
	f.search.Focus()
}

func (f *fuzzyPicker) Blur() {
	f.search.Blur()
}

func (f *fuzzyPicker) Resize(width int, height int) {
	f.list.Resize(max(56, width-10), max(10, height-14))
	f.search.Width = max(24, min(60, width-16))
}

func (f *fuzzyPicker) SetOptions(options []string) {
	f.options = append(f.options[:0], options...)
	f.search.SetValue("")
	f.applyFilter("")
}

func (f *fuzzyPicker) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			return f.list.Update(msg)
		}
	}
	var cmd tea.Cmd
	f.search, cmd = f.search.Update(msg)
	f.applyFilter(f.search.Value())
	return cmd
}

func (f *fuzzyPicker) Selected() string {
	item, ok := f.list.model.SelectedItem().(navItem)
	if !ok || strings.TrimSpace(item.title) == "" {
		return ""
	}
	return item.title
}

func (f *fuzzyPicker) View() string {
	return f.search.View() + "\n\n" + f.list.View()
}

func (f *fuzzyPicker) applyFilter(query string) {
	query = strings.TrimSpace(strings.ToLower(query))
	if len(f.options) == 0 {
		f.list.model.SetItems([]list.Item{})
		return
	}
	items := make([]navItem, 0, len(f.options))
	if query == "" {
		for i, option := range f.options {
			items = append(items, navItem{idx: i, title: option, desc: "prices city"})
		}
		f.list.SetItems(items)
		return
	}
	lowered := make([]string, len(f.options))
	for i, option := range f.options {
		lowered[i] = strings.ToLower(option)
	}
	matches := fuzzy.Find(query, lowered)
	for _, match := range matches {
		items = append(items, navItem{idx: match.Index, title: f.options[match.Index], desc: fmt.Sprintf("score %d", match.Score)})
	}
	f.list.SetItems(items)
}
