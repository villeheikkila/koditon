package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

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

type selectableList struct {
	model list.Model
}

func newSelectableList(title string, width int, height int, st styles) selectableList {
	delegate := navDelegate{styles: st}
	m := list.New([]list.Item{}, delegate, width, height)
	m.Title = title
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	m.SetShowPagination(false)
	m.Styles.Title = st.title
	m.Styles.NoItems = st.muted
	return selectableList{model: m}
}

func (s *selectableList) SetItems(items []navItem) {
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}
	s.model.SetItems(listItems)
	if len(items) > 0 && s.model.Index() >= len(items) {
		s.model.Select(0)
	}
}

func (s *selectableList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.model, cmd = s.model.Update(msg)
	return cmd
}

func (s *selectableList) Resize(width int, height int) {
	s.model.SetSize(width, height)
}

func (s *selectableList) SetTitle(title string) {
	s.model.Title = title
}

func (s *selectableList) SelectedIndex() int {
	item, ok := s.model.SelectedItem().(navItem)
	if !ok {
		return 0
	}
	return item.idx
}

func (s *selectableList) View() string {
	return s.model.View()
}
