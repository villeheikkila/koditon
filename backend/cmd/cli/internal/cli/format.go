package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	labelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func formatPrice(p *int64) string {
	if p == nil {
		return "-"
	}
	v := *p
	if v < 0 {
		return fmt.Sprintf("-%s €", formatThousands(-v))
	}
	return fmt.Sprintf("%s €", formatThousands(v))
}

func formatThousands(v int64) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatArea(a *float64) string {
	if a == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f m²", *a)
}

func formatPriceInt(p int32) string {
	return fmt.Sprintf("%s €", formatThousands(int64(p)))
}

func formatAreaFloat(a float64) string {
	return fmt.Sprintf("%.1f m²", a)
}

func renderKeyValue(label, value string) string {
	return labelStyle.Render(label+":") + " " + value
}

func renderTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var b strings.Builder

	// header
	for i, h := range headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(headerStyle.Render(padRight(h, widths[i])))
	}
	b.WriteByte('\n')

	// separator
	for i, w := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(mutedStyle.Render(strings.Repeat("─", w)))
	}
	b.WriteByte('\n')

	// rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			if i < len(widths) {
				b.WriteString(padRight(cell, widths[i]))
			} else {
				b.WriteString(cell)
			}
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
