package main

import (
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type appState int

const (
	stateList appState = iota
	stateEditor
	stateCronEditor
	stateBranchPicker
	stateParamList
	stateParamEditor
	stateDiff
	stateConfirmDelete
	stateConfirmQuit
)

// Messages
type branchesLoadedMsg struct{ branches []string }
type branchesErrMsg struct{ err error }
type statusMsg string

func loadBranchesCmd() tea.Msg {
	branches, err := ListBranches()
	if err != nil {
		return branchesErrMsg{err}
	}
	return branchesLoadedMsg{branches}
}

type model struct {
	state    appState
	project  *Project
	tasks    []Task
	filePath string
	modified bool
	width    int
	height   int

	statusText string
	errText    string

	// List view
	listCursor int
	listScroll int

	// Editor view
	editIdx     int // index in tasks, -1 for new
	editTask    Task
	editInputs  [4]textinput.Model // name, desc, pipeline, (branch is special)
	editCursor  int                // 0=name, 1=desc, 2=branch, 3=pipeline, 4=enabled, 5=schedule, 6=params
	editEnabled bool

	// Cron editor
	cronInputs     [4]textinput.Model // minute, hour, dom, month
	cronDays       [7]bool
	cronCursor     int // 0-3: text inputs, 4: day of week area
	cronDaysCursor int // 0-6: which day is highlighted

	// Branch picker
	branches         []string
	branchesLoading  bool
	branchFilter     textinput.Model
	branchCursor     int
	filteredBranches []string

	// Parameter list
	paramCursor int

	// Parameter editor
	paramIdx     int                // -1 for new
	paramInputs  [2]textinput.Model // name, description
	paramReq     bool
	paramCursor2 int // 0=name, 1=desc, 2=required

	// Diff view
	diffViewport viewport.Model
	diffContent  string
	diffReady    bool

	// Config
	skipRepoCheck bool
	debug         bool
	logger        *log.Logger
}

func newModel(project *Project, filePath string, skipRepoCheck, debug bool) model {
	m := model{
		state:           stateList,
		project:         project,
		tasks:           make([]Task, len(project.Tasks)),
		filePath:        filePath,
		skipRepoCheck:   skipRepoCheck,
		debug:           debug,
		branchesLoading: true,
	}
	copy(m.tasks, project.Tasks)

	// Branch filter input
	m.branchFilter = textinput.New()
	m.branchFilter.Placeholder = "Type to filter..."
	m.branchFilter.CharLimit = 100

	if debug {
		f, err := os.OpenFile("sem-task-editor.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			m.logger = log.New(f, "", log.LstdFlags)
		}
	}

	return m
}

func (m model) logf(format string, args ...any) {
	if m.logger != nil {
		m.logger.Printf(format, args...)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadBranchesCmd,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == stateDiff && m.diffReady {
			m.diffViewport.Width = m.width
			m.diffViewport.Height = m.height - 4
		}
		return m, nil

	case branchesLoadedMsg:
		m.branches = msg.branches
		m.branchesLoading = false
		m.logf("Loaded %d branches", len(m.branches))
		return m, nil

	case branchesErrMsg:
		m.branchesLoading = false
		m.errText = "Failed to load branches: " + msg.err.Error()
		m.logf("Branch loading error: %v", msg.err)
		return m, nil

	case statusMsg:
		m.statusText = string(msg)
		return m, nil
	}

	switch m.state {
	case stateList:
		return m.listUpdate(msg)
	case stateEditor:
		return m.editorUpdate(msg)
	case stateCronEditor:
		return m.cronUpdate(msg)
	case stateBranchPicker:
		return m.branchPickerUpdate(msg)
	case stateParamList:
		return m.paramListUpdate(msg)
	case stateParamEditor:
		return m.paramEditorUpdate(msg)
	case stateDiff:
		return m.diffUpdate(msg)
	case stateConfirmDelete:
		return m.confirmDeleteUpdate(msg)
	case stateConfirmQuit:
		return m.confirmQuitUpdate(msg)
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateList:
		return m.listView()
	case stateEditor:
		return m.editorView()
	case stateCronEditor:
		return m.cronView()
	case stateBranchPicker:
		return m.branchPickerView()
	case stateParamList:
		return m.paramListView()
	case stateParamEditor:
		return m.paramEditorView()
	case stateDiff:
		return m.diffViewFn()
	case stateConfirmDelete:
		return m.confirmDeleteView()
	case stateConfirmQuit:
		return m.confirmQuitView()
	}
	return ""
}
