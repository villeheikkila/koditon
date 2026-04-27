package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type formChoice struct {
	Label string
	Value string
}

type formFieldKind int

const (
	formFieldText formFieldKind = iota
	formFieldChoice
)

type formField struct {
	Key         string
	Label       string
	Placeholder string
	Kind        formFieldKind
	Input       promptInput
	Choices     []formChoice
	ChoiceIndex int
}

func newTextFormField(key string, label string, placeholder string) formField {
	input := newPromptInput()
	input.SetPlaceholder(placeholder)
	return formField{Key: key, Label: label, Placeholder: placeholder, Kind: formFieldText, Input: input}
}

func newChoiceFormField(key string, label string, choices []formChoice, defaultIndex int) formField {
	if len(choices) == 0 {
		choices = []formChoice{{Label: "-", Value: ""}}
	}
	safeIndex := defaultIndex
	if safeIndex < 0 || safeIndex >= len(choices) {
		safeIndex = 0
	}
	return formField{Key: key, Label: label, Kind: formFieldChoice, Choices: choices, ChoiceIndex: safeIndex}
}

type formPrimitive struct {
	styles styles
	fields []formField
	focus  int
	width  int
}

func newFormPrimitive(st styles, fields []formField) formPrimitive {
	form := formPrimitive{styles: st, fields: append([]formField(nil), fields...), focus: 0}
	form.syncFocus()
	return form
}

func (f *formPrimitive) Resize(width int) {
	f.width = width
	for i := range f.fields {
		if f.fields[i].Kind == formFieldText {
			f.fields[i].Input.Resize(width - 20)
		}
	}
}

func (f *formPrimitive) Update(msg tea.Msg) tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	field := &f.fields[f.focus]
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch key.String() {
		case "tab", "down":
			f.moveFocus(1)
			return nil
		case "shift+tab", "up":
			f.moveFocus(-1)
			return nil
		case "left":
			if field.Kind == formFieldChoice {
				return f.shiftChoice(-1)
			}
		case "right":
			if field.Kind == formFieldChoice {
				return f.shiftChoice(1)
			}
		case "j":
			if field.Kind == formFieldChoice {
				f.moveFocus(1)
				return nil
			}
		case "k":
			if field.Kind == formFieldChoice {
				f.moveFocus(-1)
				return nil
			}
		case "h":
			if field.Kind == formFieldChoice {
				return f.shiftChoice(-1)
			}
		case "l":
			if field.Kind == formFieldChoice {
				return f.shiftChoice(1)
			}
		}
	}
	if field.Kind != formFieldText {
		return nil
	}
	return field.Input.Update(msg)
}

func (f *formPrimitive) Values() map[string]string {
	values := make(map[string]string, len(f.fields))
	for _, field := range f.fields {
		switch field.Kind {
		case formFieldChoice:
			if len(field.Choices) == 0 {
				values[field.Key] = ""
				continue
			}
			values[field.Key] = strings.TrimSpace(field.Choices[field.ChoiceIndex].Value)
		default:
			values[field.Key] = field.Input.Value()
		}
	}
	return values
}

func (f *formPrimitive) View() string {
	if len(f.fields) == 0 {
		return f.styles.muted.Render("no fields")
	}
	lines := make([]string, 0, len(f.fields)*3)
	for i, field := range f.fields {
		isFocused := i == f.focus
		pointer := "  "
		if isFocused {
			pointer = "> "
		}
		lines = append(lines, f.styles.inputLabel.Render(pointer+field.Label))
		switch field.Kind {
		case formFieldChoice:
			lines = append(lines, "  "+f.renderChoice(field, isFocused))
		default:
			lines = append(lines, "  "+field.Input.View())
		}
		if i < len(f.fields)-1 {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

func (f *formPrimitive) moveFocus(delta int) {
	if len(f.fields) == 0 {
		return
	}
	f.focus = (f.focus + delta + len(f.fields)) % len(f.fields)
	f.syncFocus()
}

func (f *formPrimitive) syncFocus() {
	for i := range f.fields {
		if f.fields[i].Kind != formFieldText {
			continue
		}
		if i == f.focus {
			f.fields[i].Input.Focus()
			continue
		}
		f.fields[i].Input.Blur()
	}
}

func (f *formPrimitive) shiftChoice(delta int) tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	field := &f.fields[f.focus]
	if field.Kind != formFieldChoice || len(field.Choices) == 0 {
		return nil
	}
	field.ChoiceIndex = (field.ChoiceIndex + delta + len(field.Choices)) % len(field.Choices)
	return nil
}

func (f *formPrimitive) renderChoice(field formField, focused bool) string {
	if len(field.Choices) == 0 {
		return "-"
	}
	current := field.Choices[field.ChoiceIndex].Label
	value := "  " + current + "  "
	if focused {
		value = f.styles.selected.Render(value)
	} else {
		value = f.styles.normal.Render(value)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, f.styles.muted.Render("[h/l] "), value)
}
