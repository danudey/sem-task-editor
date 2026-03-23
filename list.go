package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) listUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.statusText = ""
		m.errText = ""

		switch msg.String() {
		case "q", "ctrl+c":
			if m.modified {
				m.state = stateConfirmQuit
				return m, nil
			}
			return m, tea.Quit

		case "up", "k":
			if m.listCursor > 0 {
				m.listCursor--
			}

		case "down", "j":
			if m.listCursor < len(m.tasks)-1 {
				m.listCursor++
			}

		case "home", "g":
			m.listCursor = 0

		case "end", "G":
			m.listCursor = len(m.tasks) - 1

		case "e", "enter":
			if len(m.tasks) > 0 {
				m = m.initEditor(m.tasks[m.listCursor], m.listCursor)
			}

		case "a":
			newTask := Task{
				Status: "ACTIVE",
			}
			m = m.initEditor(newTask, -1)

		case "d":
			if len(m.tasks) > 0 {
				m.state = stateConfirmDelete
			}

		case "c":
			if len(m.tasks) > 0 {
				orig := m.tasks[m.listCursor]
				dup := orig
				dup.ID = "" // New tasks don't get an ID
				dup.Name = orig.Name + " (copy)"
				m = m.initEditor(dup, -1)
			}

		case "s":
			if !m.modified {
				m.statusText = "No changes to save"
				return m, nil
			}
			return m.initDiffView()
		}
	}

	// Adjust scroll to keep cursor visible
	visibleLines := max(
		// header + footer
		m.height-6, 1)
	if m.listCursor < m.listScroll {
		m.listScroll = m.listCursor
	}
	if m.listCursor >= m.listScroll+visibleLines {
		m.listScroll = m.listCursor - visibleLines + 1
	}

	return m, nil
}

func (m model) listView() string {
	var b strings.Builder

	title := fmt.Sprintf("Semaphore Task Editor — %s", m.project.ProjectName)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render("  No tasks defined."))
		b.WriteString("\n\n")
	} else {
		visibleLines := max(m.height-6, 1)

		end := min(m.listScroll+visibleLines, len(m.tasks))

		// Column widths
		nameWidth := 40
		schedWidth := 16
		branchWidth := 30

		// Adjust for terminal width
		available := m.width - 8 // margins + status column
		if available > 0 {
			nameWidth = available * 40 / 100
			schedWidth = available * 18 / 100
			branchWidth = available * 30 / 100
			if nameWidth < 15 {
				nameWidth = 15
			}
			if schedWidth < 12 {
				schedWidth = 12
			}
			if branchWidth < 10 {
				branchWidth = 10
			}
		}

		for i := m.listScroll; i < end; i++ {
			t := m.tasks[i]

			// Status indicator
			var status string
			if t.Status == "ACTIVE" {
				status = activeStyle.Render("✓")
			} else {
				status = inactiveStyle.Render("✗")
			}

			// Schedule display
			sched := t.At
			if sched == "" {
				sched = "(manual)"
			}

			// Truncate fields
			name := truncate(t.Name, nameWidth)
			sched = truncate(sched, schedWidth)
			branch := truncate(t.Branch, branchWidth)

			line := fmt.Sprintf(" %s  %-*s  %-*s  %-*s",
				status,
				nameWidth, name,
				schedWidth, sched,
				branchWidth, branch,
			)

			if i == m.listCursor {
				line = selectedStyle.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(m.tasks) > visibleLines {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  [%d/%d]", m.listCursor+1, len(m.tasks))))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Status / error messages
	if m.errText != "" {
		b.WriteString(errorStyle.Render(m.errText))
		b.WriteString("\n")
	} else if m.statusText != "" {
		b.WriteString(statusMsgStyle.Render(m.statusText))
		b.WriteString("\n")
	}

	// Help
	help := []string{
		"↑/↓ navigate",
		"e edit",
		"a add",
		"d delete",
		"c copy",
		"s save",
		"q quit",
	}
	if m.modified {
		help = append([]string{lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("● modified")}, help...)
	}
	b.WriteString(helpStyle.Render(strings.Join(help, "  ")))

	return b.String()
}

func (m model) confirmDeleteUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			name := m.tasks[m.listCursor].Name
			m.tasks = append(m.tasks[:m.listCursor], m.tasks[m.listCursor+1:]...)
			if m.listCursor >= len(m.tasks) && m.listCursor > 0 {
				m.listCursor--
			}
			m.modified = true
			m.state = stateList
			m.statusText = fmt.Sprintf("Deleted task: %s", name)

		case "n", "N", "esc":
			m.state = stateList
		}
	}
	return m, nil
}

func (m model) confirmDeleteView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")

	if m.listCursor < len(m.tasks) {
		name := m.tasks[m.listCursor].Name
		b.WriteString(fmt.Sprintf("  Delete task %q?\n\n", name))
	}

	b.WriteString(helpStyle.Render("  y: yes  n: no"))

	return b.String()
}

func (m model) confirmQuitUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			return m, tea.Quit
		case "n", "N", "esc":
			m.state = stateList
		}
	}
	return m, nil
}

func (m model) confirmQuitView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Unsaved Changes"))
	b.WriteString("\n\n")
	b.WriteString("  You have unsaved changes. Quit without saving?\n\n")
	b.WriteString(helpStyle.Render("  y: discard and quit  n: go back"))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
