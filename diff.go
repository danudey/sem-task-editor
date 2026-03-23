package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pmezard/go-difflib/difflib"
)

func (m model) initDiffView() (tea.Model, tea.Cmd) {
	output, err := m.project.GenerateOutput(m.tasks)
	if err != nil {
		m.errText = fmt.Sprintf("Error generating output: %v", err)
		return m, nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(m.project.OriginalContent),
		B:        difflib.SplitLines(output),
		FromFile: m.filePath + " (original)",
		ToFile:   m.filePath + " (modified)",
		Context:  3,
	}

	diffText, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		m.errText = fmt.Sprintf("Error generating diff: %v", err)
		return m, nil
	}

	if diffText == "" {
		m.statusText = "No changes to save"
		return m, nil
	}

	// Colorize the diff
	m.diffContent = colorizeDiff(diffText)

	m.state = stateDiff
	m.diffReady = false

	// The viewport will be initialized on the next WindowSizeMsg or right now
	vp := viewport.New(m.width, m.height-4)
	vp.SetContent(m.diffContent)
	m.diffViewport = vp
	m.diffReady = true

	return m, nil
}

func (m model) diffUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.diffViewport.Width = m.width
		m.diffViewport.Height = m.height - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// Accept changes - save the file
			err := m.project.Save(m.filePath, m.tasks)
			if err != nil {
				m.errText = fmt.Sprintf("Error saving: %v", err)
				m.state = stateList
				return m, nil
			}
			// Print success message and quit
			fmt.Printf("Saved changes to %s\n", m.filePath)
			return m, tea.Quit

		case "n", "N", "esc":
			// Reject changes
			m.state = stateList
			m.statusText = "Changes not saved"
			return m, nil

		case "q":
			m.state = stateList
			return m, nil
		}
	}

	// Update viewport for scrolling
	var cmd tea.Cmd
	m.diffViewport, cmd = m.diffViewport.Update(msg)
	return m, cmd
}

func (m model) diffViewFn() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Review Changes"))
	b.WriteString("\n")

	if m.diffReady {
		b.WriteString(m.diffViewport.View())
		b.WriteString("\n")
	}

	scrollInfo := ""
	if m.diffReady {
		pct := m.diffViewport.ScrollPercent() * 100
		scrollInfo = fmt.Sprintf("  %.0f%%", pct)
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("  y: accept and save  n/Esc: go back  ↑/↓/PgUp/PgDn: scroll%s", scrollInfo)))

	return b.String()
}

func colorizeDiff(diff string) string {
	var b strings.Builder

	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			b.WriteString(diffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(diffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(diffAddStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(diffRemoveStyle.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}
