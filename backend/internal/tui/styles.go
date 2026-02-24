package tui

import "charm.land/lipgloss/v2"

type styles struct {
	app                 lipgloss.Style
	title               lipgloss.Style
	help                lipgloss.Style
	breadcrumb          lipgloss.Style
	logo                lipgloss.Style
	panel               lipgloss.Style
	selected            lipgloss.Style
	selectedDescription lipgloss.Style
	normal              lipgloss.Style
	description         lipgloss.Style
	success             lipgloss.Style
	error               lipgloss.Style
	muted               lipgloss.Style
	inputLabel          lipgloss.Style
	progressLabel       lipgloss.Style
	progressFill        lipgloss.Style
	progressEmpty       lipgloss.Style
	running             lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		app:                 lipgloss.NewStyle().Padding(1, 2),
		title:               lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		help:                lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		breadcrumb:          lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true),
		logo:                lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true),
		panel:               lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1),
		selected:            lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("33")).Bold(true),
		selectedDescription: lipgloss.NewStyle().Foreground(lipgloss.Color("153")).Background(lipgloss.Color("24")),
		normal:              lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		description:         lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		success:             lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		error:               lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		muted:               lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		inputLabel:          lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true),
		progressLabel:       lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true),
		progressFill:        lipgloss.NewStyle().Background(lipgloss.Color("37")).Foreground(lipgloss.Color("37")),
		progressEmpty:       lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("238")),
		running:             lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
	}
}
