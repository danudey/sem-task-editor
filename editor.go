package main

import (
	"fmt"
	"slices"
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

			if detail := paramDetail(p); detail != "" {
				b.WriteString(helpStyle.Render("      " + detail))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  a: add  e/↵: edit  d: delete  Esc: back"))

	return b.String()
}

// Parameter editor

const (
	pFieldName = iota
	pFieldDesc
	pFieldReq
	pFieldDefault
	pFieldOptions
	pFieldCount
)

// paramDetail summarises the default value and options of a parameter.
func paramDetail(p Parameter) string {
	var parts []string
	if p.DefaultValue != "" {
		parts = append(parts, "default: "+p.DefaultValue)
	}
	if len(p.Options) > 0 {
		parts = append(parts, "options: "+strings.Join(p.Options, ", "))
	}
	return strings.Join(parts, "   ")
}

func (m model) initParamEditor(idx int) model {
	m.state = stateParamEditor
	m.paramIdx = idx
	m.paramCursor2 = 0

	var name, desc string
	var req bool

	m.paramDefault = ""
	m.paramOptions = nil

	if idx >= 0 && idx < len(m.editTask.Parameters) {
		p := m.editTask.Parameters[idx]
		name = p.Name
		desc = p.Description
		req = p.Required
		m.paramDefault = p.DefaultValue
		m.paramOptions = append([]string(nil), p.Options...)
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

	ti2 := textinput.New()
	ti2.Placeholder = "Default value"
	ti2.CharLimit = 200
	ti2.Width = 40
	ti2.SetValue(m.paramDefault)
	ti2.Blur()

	m.paramInputs = [3]textinput.Model{ti0, ti1, ti2}
	m.paramReq = req

	return m
}

// paramInputIdx maps a parameter editor field to an index in paramInputs, or
// -1 if the field is not a text input. The default value is only a text input
// when there are no options constraining it.
func (m model) paramInputIdx(field int) int {
	switch field {
	case pFieldName:
		return 0
	case pFieldDesc:
		return 1
	case pFieldDefault:
		if len(m.paramOptions) == 0 {
			return 2
		}
	}
	return -1
}

func (m model) paramEditorUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateParamList
			return m, nil

		case "ctrl+s":
			return m.saveParam()

		case "tab", "down":
			return m.paramEditorMove(1)

		case "shift+tab", "up":
			return m.paramEditorMove(-1)

		case "enter":
			switch m.paramCursor2 {
			case pFieldOptions:
				return m.initOptionEditor(), nil
			case pFieldReq:
				m.paramReq = !m.paramReq
				return m, nil
			case pFieldDefault:
				if len(m.paramOptions) > 0 {
					return m.cycleDefault(1), nil
				}
			}

		case "left", "right":
			if m.paramCursor2 == pFieldDefault && len(m.paramOptions) > 0 {
				delta := 1
				if msg.String() == "left" {
					delta = -1
				}
				return m.cycleDefault(delta), nil
			}

		case " ":
			switch m.paramCursor2 {
			case pFieldReq:
				m.paramReq = !m.paramReq
				return m, nil
			case pFieldDefault:
				if len(m.paramOptions) > 0 {
					return m.cycleDefault(1), nil
				}
			}
		}
	}

	// Update focused text input
	if i := m.paramInputIdx(m.paramCursor2); i >= 0 {
		var cmd tea.Cmd
		m.paramInputs[i], cmd = m.paramInputs[i].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) paramEditorMove(delta int) (tea.Model, tea.Cmd) {
	next := m.paramCursor2 + delta
	if next < 0 || next >= pFieldCount {
		return m, nil
	}

	if i := m.paramInputIdx(m.paramCursor2); i >= 0 {
		m.paramInputs[i].Blur()
	}
	m.paramCursor2 = next
	if i := m.paramInputIdx(next); i >= 0 {
		return m, m.paramInputs[i].Focus()
	}
	return m, nil
}

// cycleDefault steps the default value through "(unset)" and the options list.
// Only used when options are defined.
func (m model) cycleDefault(delta int) model {
	choices := append([]string{""}, m.paramOptions...)

	cur := 0
	for i, c := range choices {
		if c == m.paramDefault {
			cur = i
			break
		}
	}

	cur = ((cur+delta)%len(choices) + len(choices)) % len(choices)
	m.paramDefault = choices[cur]
	m.errText = ""
	return m
}

// syncDefaultFromInput copies the freeform default value input into
// paramDefault. It is a no-op when options are defined, since then the value is
// picked from the options instead.
func (m model) syncDefaultFromInput() model {
	if len(m.paramOptions) == 0 {
		m.paramDefault = m.paramInputs[2].Value()
	}
	return m
}

func (m model) saveParam() (tea.Model, tea.Cmd) {
	m = m.syncDefaultFromInput()

	name := m.paramInputs[0].Value()
	if name == "" {
		m.errText = "Parameter name is required"
		return m, nil
	}

	if len(m.paramOptions) > 0 && m.paramDefault != "" && !slices.Contains(m.paramOptions, m.paramDefault) {
		m.errText = fmt.Sprintf("Default value %q is not one of the options", m.paramDefault)
		return m, nil
	}

	p := Parameter{
		Name:         name,
		Description:  m.paramInputs[1].Value(),
		Required:     m.paramReq,
		DefaultValue: m.paramDefault,
		Options:      append([]string(nil), m.paramOptions...),
	}

	if m.paramIdx == -1 {
		m.editTask.Parameters = append(m.editTask.Parameters, p)
	} else {
		m.editTask.Parameters[m.paramIdx] = p
	}

	m.errText = ""
	m.state = stateParamList
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
		{"Required:", renderCheckbox(m.paramReq, m.paramCursor2 == pFieldReq)},
		{"Default Value:", m.renderDefaultValue()},
		{"Options:", renderOptionsSummary(m.paramOptions, m.paramCursor2 == pFieldOptions)},
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

	b.WriteString(helpStyle.Render("  Tab/↑/↓: navigate  Space: toggle/choose  Enter: manage options  Ctrl+S: save  Esc: cancel"))

	return b.String()
}

func (m model) renderDefaultValue() string {
	focused := m.paramCursor2 == pFieldDefault

	if len(m.paramOptions) == 0 {
		s := m.paramInputs[2].View()
		if focused {
			s += helpStyle.Render("  freeform (add options to restrict)")
		}
		return s
	}

	var s string
	switch {
	case m.paramDefault == "":
		s = inactiveStyle.Render("(unset)")
	case slices.Contains(m.paramOptions, m.paramDefault):
		s = activeStyle.Render(m.paramDefault)
	default:
		s = errorStyle.Render(m.paramDefault + "  ✗ not one of the options")
	}

	if focused {
		s += helpStyle.Render(fmt.Sprintf("  Space/←/→: choose (%d options)", len(m.paramOptions)))
	}
	return s
}

func renderOptionsSummary(options []string, focused bool) string {
	var s string
	if len(options) == 0 {
		s = inactiveStyle.Render("(none — default value is freeform)")
	} else {
		s = strings.Join(options, ", ")
	}
	if focused {
		s += helpStyle.Render("  ↵ manage")
	}
	return s
}

// Option list editor

func (m model) initOptionEditor() model {
	// Capture the freeform default before options take over the field.
	m = m.syncDefaultFromInput()
	m.state = stateParamOptions
	m.optCursor = 0
	m.optIdx = -1
	m.optEditing = false
	m.errText = ""
	return m
}

func (m model) startOptionInput(idx int) model {
	m.optEditing = true
	m.optIdx = idx
	m.errText = ""

	ti := textinput.New()
	ti.Placeholder = "Option value"
	ti.CharLimit = 200
	ti.Width = 40
	if idx >= 0 && idx < len(m.paramOptions) {
		ti.SetValue(m.paramOptions[idx])
	}
	ti.Focus()
	m.optInput = ti

	return m
}

func (m model) paramOptionsUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.optEditing {
		return m.optionInputUpdate(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m.closeOptionEditor()

		case "up", "k":
			if m.optCursor > 0 {
				m.optCursor--
			}

		case "down", "j":
			if m.optCursor < len(m.paramOptions)-1 {
				m.optCursor++
			}

		case "a":
			m = m.startOptionInput(-1)

		case "e", "enter":
			if len(m.paramOptions) > 0 {
				m = m.startOptionInput(m.optCursor)
			}

		case "d":
			if len(m.paramOptions) > 0 {
				removed := m.paramOptions[m.optCursor]
				m.paramOptions = append(
					m.paramOptions[:m.optCursor],
					m.paramOptions[m.optCursor+1:]...,
				)
				if m.optCursor >= len(m.paramOptions) && m.optCursor > 0 {
					m.optCursor--
				}
				// The default value must stay one of the options.
				if m.paramDefault == removed {
					m.paramDefault = ""
				}
				m.errText = ""
			}
		}
	}

	return m, nil
}

func (m model) optionInputUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.optEditing = false
			m.errText = ""
			return m, nil

		case "enter":
			val := m.optInput.Value()
			if strings.TrimSpace(val) == "" {
				m.errText = "Option value cannot be empty"
				return m, nil
			}
			for i, o := range m.paramOptions {
				if o == val && i != m.optIdx {
					m.errText = fmt.Sprintf("Option %q already exists", val)
					return m, nil
				}
			}

			if m.optIdx == -1 {
				m.paramOptions = append(m.paramOptions, val)
				m.optCursor = len(m.paramOptions) - 1
			} else {
				old := m.paramOptions[m.optIdx]
				m.paramOptions[m.optIdx] = val
				if m.paramDefault == old {
					m.paramDefault = val
				}
			}

			m.optEditing = false
			m.errText = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.optInput, cmd = m.optInput.Update(msg)
	return m, cmd
}

func (m model) closeOptionEditor() (tea.Model, tea.Cmd) {
	m.state = stateParamEditor

	// With no options left the default value becomes a freeform field again,
	// so push the current value back into the text input.
	if len(m.paramOptions) == 0 {
		m.paramInputs[2].SetValue(m.paramDefault)
	}

	if i := m.paramInputIdx(m.paramCursor2); i >= 0 {
		return m, m.paramInputs[i].Focus()
	}
	return m, nil
}

func (m model) paramOptionsView() string {
	var b strings.Builder

	name := m.paramInputs[0].Value()
	header := "Options"
	if name != "" {
		header = fmt.Sprintf("Options: %s", name)
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")

	if len(m.paramOptions) == 0 {
		b.WriteString(helpStyle.Render("  No options — the default value is a freeform text field."))
		b.WriteString("\n")
	} else {
		for i, o := range m.paramOptions {
			line := "  " + o
			if o == m.paramDefault {
				line += helpStyle.Render("  (default)")
			}
			if i == m.optCursor && !m.optEditing {
				line = selectedStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.errText != "" {
		b.WriteString(errorStyle.Render("  " + m.errText))
		b.WriteString("\n\n")
	}

	if m.optEditing {
		lbl := "New option:"
		if m.optIdx >= 0 {
			lbl = "Edit option:"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n\n", focusedLabelStyle.Render(lbl), m.optInput.View()))
		b.WriteString(helpStyle.Render("  Enter: confirm  Esc: cancel"))
	} else {
		b.WriteString(helpStyle.Render("  a: add  e/↵: edit  d: delete  Esc: back"))
	}

	return b.String()
}
