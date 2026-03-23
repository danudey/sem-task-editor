package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	statusMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Width(18).
			Align(lipgloss.Right)

	focusedLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Bold(true).
				Width(18).
				Align(lipgloss.Right)

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	daySelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("62"))

	dayActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	dayInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
)
