package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	edFieldName = iota
	edFieldDesc
	edFieldBranch
	edFieldPipeline
	edFieldEnabled
	edFieldSchedule
	edFieldParams
	edFieldCount
)

func (m model) initEditor(task Task, idx int) model {
	m.state = stateEditor
	m.editIdx = idx
	m.editTask = task
	m.editCursor = 0
	m.editEnabled = task.Status == "ACTIVE"

	// Initialize text inputs: name, description, pipeline file
	// Branch is handled specially (opens picker)
	labels := [4]string{"Name", "Description", "Branch", "Pipeline File"}
	values := [4]string{task.Name, task.Description, task.Branch, task.PipelineFile}
	widths := [4]int{60, 60, 60, 60}

	for i := range 4 {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 200
		ti.Width = widths[i]
		ti.SetValue(values[i])
		m.editInputs[i] = ti
	}

	m.editInputs[0].Focus()
	for i := 1; i < 4; i++ {
		m.editInputs[i].Blur()
	}

	return m
}

func (m model) editorUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			return m, nil

		case "ctrl+s":
			return m.saveEditorTask()

		case "tab", "down":
			return m.editorNextField()

		case "shift+tab", "up":
			return m.editorPrevField()

		case "enter":
			switch m.editCursor {
			case edFieldBranch:
				// Open branch picker
				return m.initBranchPicker()
			case edFieldSchedule:
				// Open cron editor
				m = m.initCronEditor()
				return m, nil
			case edFieldParams:
				// Open parameter list
				m.state = stateParamList
				m.paramCursor = 0
				return m, nil
			case edFieldEnabled:
				m.editEnabled = !m.editEnabled
				return m, nil
			}

		case " ":
			if m.editCursor == edFieldEnabled {
				m.editEnabled = !m.editEnabled
				return m, nil
			}
		}
	}

	// Update the focused text input
	if m.editCursor >= 0 && m.editCursor <= 3 {
		var cmd tea.Cmd
		m.editInputs[m.editCursor], cmd = m.editInputs[m.editCursor].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) editorNextField() (tea.Model, tea.Cmd) {
	if m.editCursor < edFieldCount-1 {
		if m.editCursor >= 0 && m.editCursor <= 3 {
			m.editInputs[m.editCursor].Blur()
		}
		m.editCursor++
		if m.editCursor >= 0 && m.editCursor <= 3 {
			cmd := m.editInputs[m.editCursor].Focus()
			return m, cmd
		}
	}
	return m, nil
}

func (m model) editorPrevField() (tea.Model, tea.Cmd) {
	if m.editCursor > 0 {
		if m.editCursor >= 0 && m.editCursor <= 3 {
			m.editInputs[m.editCursor].Blur()
		}
		m.editCursor--
		if m.editCursor >= 0 && m.editCursor <= 3 {
			cmd := m.editInputs[m.editCursor].Focus()
			return m, cmd
		}
	}
	return m, nil
}

func (m model) saveEditorTask() (tea.Model, tea.Cmd) {
	// Collect values from inputs
	m.editTask.Name = m.editInputs[0].Value()
	m.editTask.Description = m.editInputs[1].Value()
	m.editTask.Branch = m.editInputs[2].Value()
	m.editTask.PipelineFile = m.editInputs[3].Value()

	if m.editEnabled {
		m.editTask.Status = "ACTIVE"
	} else {
		m.editTask.Status = "INACTIVE"
	}

	// Validation
	if m.editTask.Name == "" {
		m.errText = "Task name is required"
		return m, nil
	}
	if m.editTask.Branch == "" {
		m.errText = "Branch is required"
		return m, nil
	}
	if m.editTask.PipelineFile == "" {
		m.errText = "Pipeline file is required"
		return m, nil
	}

	// Verify pipeline file exists in branch
	check := FileExistsInBranch(m.editTask.Branch, m.editTask.PipelineFile)
	if !check.RefAvailable {
		m.statusText = fmt.Sprintf("Warning: branch %q not fetched locally, could not verify pipeline file", m.editTask.Branch)
	} else if !check.Exists {
		m.errText = fmt.Sprintf("Pipeline file %q not found in branch %q", m.editTask.PipelineFile, m.editTask.Branch)
		return m, nil
	}

	if m.editIdx == -1 {
		// New task - no ID
		m.editTask.ID = ""
		m.tasks = append(m.tasks, m.editTask)
		m.listCursor = len(m.tasks) - 1
		m.statusText = fmt.Sprintf("Added task: %s", m.editTask.Name)
	} else {
		m.tasks[m.editIdx] = m.editTask
		m.statusText = fmt.Sprintf("Updated task: %s", m.editTask.Name)
	}

	m.modified = true
	m.state = stateList
	m.errText = ""
	return m, nil
}

func (m model) editorView() string {
	var b strings.Builder

	header := "New Task"
	if m.editIdx >= 0 {
		header = fmt.Sprintf("Edit Task: %s", m.editTask.Name)
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")

	// Field renderers
	fields := []struct {
		label string
		view  string
	}{
		{"Name:", m.editInputs[0].View()},
		{"Description:", m.editInputs[1].View()},
		{"Branch:", m.editInputs[2].View() + branchHint(m.editCursor == edFieldBranch)},
		{"Pipeline File:", m.editInputs[3].View()},
		{"Enabled:", renderCheckbox(m.editEnabled, m.editCursor == edFieldEnabled)},
		{"Schedule:", renderSchedule(m.editTask.At, m.editCursor == edFieldSchedule)},
		{"Parameters:", renderParamSummary(m.editTask.Parameters, m.editCursor == edFieldParams)},
	}

	for i, f := range fields {
		lbl := f.label
		if m.editCursor == i {
			lbl = focusedLabelStyle.Render(lbl)
		} else {
			lbl = labelStyle.Render(lbl)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", lbl, f.view))
	}

	b.WriteString("\n")

	if m.errText != "" {
		b.WriteString(errorStyle.Render("  " + m.errText))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("  Tab/↑/↓: navigate  Enter: action  Space: toggle  Ctrl+S: save  Esc: cancel"))

	return b.String()
}

func branchHint(focused bool) string {
	if focused {
		return helpStyle.Render("  ↵ browse branches")
	}
	return ""
}

func renderCheckbox(checked, focused bool) string {
	var box string
	if checked {
		if focused {
			box = activeStyle.Render("[✓] Enabled")
		} else {
			box = activeStyle.Render("[✓] Enabled")
		}
	} else {
		if focused {
			box = inactiveStyle.Render("[ ] Disabled")
		} else {
			box = inactiveStyle.Render("[ ] Disabled")
		}
	}
	return box
}

func renderSchedule(at string, focused bool) string {
	if at == "" {
		s := "(not scheduled)"
		if focused {
			return inactiveStyle.Render(s) + helpStyle.Render("  ↵ set schedule")
		}
		return inactiveStyle.Render(s)
	}

	desc := describeCron(at)
	s := activeStyle.Render(at)
	if desc != "" {
		s += helpStyle.Render("  " + desc)
	}
	if focused {
		s += helpStyle.Render("  ↵ edit")
	}
	return s
}

func renderParamSummary(params []Parameter, focused bool) string {
	if len(params) == 0 {
		s := "(none)"
		if focused {
			return inactiveStyle.Render(s) + helpStyle.Render("  ↵ manage")
		}
		return inactiveStyle.Render(s)
	}

	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	s := strings.Join(names, ", ")
	if focused {
		return s + helpStyle.Render("  ↵ manage")
	}
	return s
}

// Branch picker

func (m model) initBranchPicker() (tea.Model, tea.Cmd) {
	m.state = stateBranchPicker
	m.branchFilter.SetValue("")
	m.branchFilter.Focus()
	m.branchCursor = 0
	m.filteredBranches = m.branches
	return m, m.branchFilter.Focus()
}

func (m model) branchPickerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateEditor
			// Re-focus the branch input
			cmd := m.editInputs[edFieldBranch].Focus()
			return m, cmd

		case "enter":
			if len(m.filteredBranches) > 0 && m.branchCursor < len(m.filteredBranches) {
				m.editInputs[edFieldBranch].SetValue(m.filteredBranches[m.branchCursor])
				m.editTask.Branch = m.filteredBranches[m.branchCursor]
			}
			m.state = stateEditor
			cmd := m.editInputs[edFieldBranch].Focus()
			return m, cmd

		case "up", "ctrl+p":
			if m.branchCursor > 0 {
				m.branchCursor--
			}
			return m, nil

		case "down", "ctrl+n":
			if m.branchCursor < len(m.filteredBranches)-1 {
				m.branchCursor++
			}
			return m, nil
		}
	}

	// Update filter input
	var cmd tea.Cmd
	m.branchFilter, cmd = m.branchFilter.Update(msg)

	// Filter branches
	filter := strings.ToLower(m.branchFilter.Value())
	m.filteredBranches = nil
	for _, br := range m.branches {
		if filter == "" || strings.Contains(strings.ToLower(br), filter) {
			m.filteredBranches = append(m.filteredBranches, br)
		}
	}

	// Keep cursor in bounds
	if m.branchCursor >= len(m.filteredBranches) {
		m.branchCursor = len(m.filteredBranches) - 1
	}
	if m.branchCursor < 0 {
		m.branchCursor = 0
	}

	return m, cmd
}

func (m model) branchPickerView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Branch"))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Filter: %s\n\n", m.branchFilter.View()))

	if m.branchesLoading {
		b.WriteString(helpStyle.Render("  Loading branches..."))
		b.WriteString("\n")
	} else if len(m.filteredBranches) == 0 {
		b.WriteString(helpStyle.Render("  No matching branches"))
		b.WriteString("\n")
	} else {
		maxVisible := max(m.height-8, 5)

		start := 0
		if m.branchCursor >= maxVisible {
			start = m.branchCursor - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.filteredBranches))

		for i := start; i < end; i++ {
			br := m.filteredBranches[i]
			if i == m.branchCursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", br)))
			} else {
				b.WriteString(fmt.Sprintf("    %s", br))
			}
			b.WriteString("\n")
		}

		if len(m.filteredBranches) > maxVisible {
			b.WriteString(helpStyle.Render(fmt.Sprintf("\n  [%d/%d]", m.branchCursor+1, len(m.filteredBranches))))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑/↓: navigate  Enter: select  Esc: cancel"))

	return b.String()
}

// Parameter list

func (m model) paramListUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.state = stateEditor
			return m, nil

		case "up", "k":
			if m.paramCursor > 0 {
				m.paramCursor--
			}

		case "down", "j":
			if m.paramCursor < len(m.editTask.Parameters)-1 {
				m.paramCursor++
			}

		case "a":
			m = m.initParamEditor(-1)

		case "e", "enter":
			if len(m.editTask.Parameters) > 0 {
				m = m.initParamEditor(m.paramCursor)
			}

		case "d":
			if len(m.editTask.Parameters) > 0 {
				m.editTask.Parameters = append(
					m.editTask.Parameters[:m.paramCursor],
					m.editTask.Parameters[m.paramCursor+1:]...,
				)
				if m.paramCursor >= len(m.editTask.Parameters) && m.paramCursor > 0 {
					m.paramCursor--
				}
			}
		}
	}
	return m, nil
}

func (m model) paramListView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Parameters"))
	b.WriteString("\n")

	if len(m.editTask.Parameters) == 0 {
		b.WriteString(helpStyle.Render("  No parameters defined."))
		b.WriteString("\n")
	} else {
		for i, p := range m.editTask.Parameters {
			req := ""
			if p.Required {
				req = " (required)"
			}
			line := fmt.Sprintf("  %s%s", p.Name, req)
			if p.Description != "" {
				line += helpStyle.Render(" — " + p.Description)
			}

			if i == m.paramCursor {
				line = selectedStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  a: add  e/↵: edit  d: delete  Esc: back"))

	return b.String()
}

// Parameter editor

func (m model) initParamEditor(idx int) model {
	m.state = stateParamEditor
	m.paramIdx = idx
	m.paramCursor2 = 0

	var name, desc string
	var req bool

	if idx >= 0 && idx < len(m.editTask.Parameters) {
		p := m.editTask.Parameters[idx]
		name = p.Name
		desc = p.Description
		req = p.Required
	}

	ti0 := textinput.New()
	ti0.Placeholder = "Parameter name"
	ti0.CharLimit = 100
	ti0.Width = 40
	ti0.SetValue(name)
	ti0.Focus()

	ti1 := textinput.New()
	ti1.Placeholder = "Description"
	ti1.CharLimit = 200
	ti1.Width = 60
	ti1.SetValue(desc)
	ti1.Blur()

	m.paramInputs = [2]textinput.Model{ti0, ti1}
	m.paramReq = req

	return m
}

func (m model) paramEditorUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateParamList
			return m, nil

		case "ctrl+s":
			// Save parameter
			name := m.paramInputs[0].Value()
			if name == "" {
				m.errText = "Parameter name is required"
				return m, nil
			}

			p := Parameter{
				Name:        name,
				Description: m.paramInputs[1].Value(),
				Required:    m.paramReq,
			}

			if m.paramIdx == -1 {
				m.editTask.Parameters = append(m.editTask.Parameters, p)
			} else {
				m.editTask.Parameters[m.paramIdx] = p
			}

			m.errText = ""
			m.state = stateParamList
			return m, nil

		case "tab", "down":
			if m.paramCursor2 < 2 {
				if m.paramCursor2 < 2 {
					if m.paramCursor2 < len(m.paramInputs) {
						m.paramInputs[m.paramCursor2].Blur()
					}
				}
				m.paramCursor2++
				if m.paramCursor2 < len(m.paramInputs) {
					cmd := m.paramInputs[m.paramCursor2].Focus()
					return m, cmd
				}
			}
			return m, nil

		case "shift+tab", "up":
			if m.paramCursor2 > 0 {
				if m.paramCursor2 < len(m.paramInputs) {
					m.paramInputs[m.paramCursor2].Blur()
				}
				m.paramCursor2--
				if m.paramCursor2 < len(m.paramInputs) {
					cmd := m.paramInputs[m.paramCursor2].Focus()
					return m, cmd
				}
			}
			return m, nil

		case " ":
			if m.paramCursor2 == 2 {
				m.paramReq = !m.paramReq
				return m, nil
			}
		}
	}

	// Update focused text input
	if m.paramCursor2 >= 0 && m.paramCursor2 < 2 {
		var cmd tea.Cmd
		m.paramInputs[m.paramCursor2], cmd = m.paramInputs[m.paramCursor2].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) paramEditorView() string {
	var b strings.Builder

	header := "New Parameter"
	if m.paramIdx >= 0 {
		header = "Edit Parameter"
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")

	fields := []struct {
		label string
		view  string
	}{
		{"Name:", m.paramInputs[0].View()},
		{"Description:", m.paramInputs[1].View()},
		{"Required:", renderCheckbox(m.paramReq, m.paramCursor2 == 2)},
	}

	for i, f := range fields {
		lbl := f.label
		if m.paramCursor2 == i {
			lbl = focusedLabelStyle.Render(lbl)
		} else {
			lbl = labelStyle.Render(lbl)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", lbl, f.view))
	}

	b.WriteString("\n")

	if m.errText != "" {
		b.WriteString(errorStyle.Render("  " + m.errText))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("  Tab/↑/↓: navigate  Space: toggle required  Ctrl+S: save  Esc: cancel"))

	return b.String()
}
