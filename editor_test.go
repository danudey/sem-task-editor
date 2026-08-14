package main

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// send feeds keys to the model, dispatching on its current state.
func send(t *testing.T, m model, keys ...string) model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(model)
	}
	return m
}

func editorWithParam(p Parameter) model {
	m := model{editTask: Task{Parameters: []Parameter{p}}}
	m = m.initParamEditor(0)
	return m
}

func TestParamEditorLoadsDefaultAndOptions(t *testing.T) {
	m := editorWithParam(Parameter{
		Name:         "BUILD",
		Required:     true,
		DefaultValue: "true",
		Options:      []string{"true", "false"},
	})

	if m.paramDefault != "true" {
		t.Errorf("paramDefault = %q, want %q", m.paramDefault, "true")
	}
	if !slices.Equal(m.paramOptions, []string{"true", "false"}) {
		t.Errorf("paramOptions = %v", m.paramOptions)
	}
	// With options present the default value is a chooser, not a text input.
	if got := m.paramInputIdx(pFieldDefault); got != -1 {
		t.Errorf("paramInputIdx(pFieldDefault) = %d, want -1", got)
	}
}

func TestParamEditorDefaultIsFreeformWithoutOptions(t *testing.T) {
	m := editorWithParam(Parameter{Name: "PLUGIN_BRANCH", DefaultValue: "release-v1.4"})

	if got := m.paramInputIdx(pFieldDefault); got != 2 {
		t.Errorf("paramInputIdx(pFieldDefault) = %d, want 2", got)
	}
	if got := m.paramInputs[2].Value(); got != "release-v1.4" {
		t.Errorf("default input = %q, want %q", got, "release-v1.4")
	}
}

func TestParamEditorCycleDefault(t *testing.T) {
	m := editorWithParam(Parameter{Name: "BUILD", Options: []string{"true", "false"}})

	// Choices are "(unset)", then each option, wrapping around.
	want := []string{"true", "false", "", "true"}
	for i, w := range want {
		m = m.cycleDefault(1)
		if m.paramDefault != w {
			t.Fatalf("step %d: paramDefault = %q, want %q", i, m.paramDefault, w)
		}
	}

	m = m.cycleDefault(-1)
	if m.paramDefault != "" {
		t.Errorf("after backwards step: paramDefault = %q, want unset", m.paramDefault)
	}
}

func TestParamEditorCycleDefaultFromInvalidValue(t *testing.T) {
	// A file may contain a default value that is not in the options list.
	m := editorWithParam(Parameter{Name: "BUILD", DefaultValue: "maybe", Options: []string{"true", "false"}})

	if m.paramDefault != "maybe" {
		t.Fatalf("paramDefault = %q, want the value from the file", m.paramDefault)
	}
	m = m.cycleDefault(1)
	if m.paramDefault != "true" {
		t.Errorf("paramDefault = %q, want %q", m.paramDefault, "true")
	}
}

func TestParamEditorSaveRejectsDefaultNotInOptions(t *testing.T) {
	m := editorWithParam(Parameter{Name: "BUILD", DefaultValue: "maybe", Options: []string{"true", "false"}})

	next, _ := m.saveParam()
	m = next.(model)

	if m.errText == "" {
		t.Error("expected an error for a default value outside the options")
	}
	if m.state == stateParamList {
		t.Error("should not have left the parameter editor")
	}
	if m.editTask.Parameters[0].DefaultValue != "maybe" {
		t.Error("parameter should be unchanged while invalid")
	}
}

func TestParamEditorSaveKeepsDefaultAndOptions(t *testing.T) {
	m := editorWithParam(Parameter{
		Name:         "BUILD",
		Required:     true,
		Description:  "build images",
		DefaultValue: "true",
		Options:      []string{"true", "false"},
	})

	next, _ := m.saveParam()
	m = next.(model)

	if m.errText != "" {
		t.Fatalf("unexpected error: %s", m.errText)
	}
	got := m.editTask.Parameters[0]
	if got.DefaultValue != "true" {
		t.Errorf("DefaultValue = %q", got.DefaultValue)
	}
	if !slices.Equal(got.Options, []string{"true", "false"}) {
		t.Errorf("Options = %v", got.Options)
	}
}

func TestParamEditorSaveFreeformDefault(t *testing.T) {
	m := editorWithParam(Parameter{Name: "PLUGIN_BRANCH"})
	m.paramCursor2 = pFieldDefault
	m.paramInputs[2].SetValue("release-v1.4")

	next, _ := m.saveParam()
	m = next.(model)

	got := m.editTask.Parameters[0]
	if got.DefaultValue != "release-v1.4" {
		t.Errorf("DefaultValue = %q, want %q", got.DefaultValue, "release-v1.4")
	}
	if got.Options != nil {
		t.Errorf("Options = %v, want nil", got.Options)
	}
}

func TestOptionEditorAddEditDelete(t *testing.T) {
	m := editorWithParam(Parameter{Name: "BUILD"})
	m.paramCursor2 = pFieldOptions
	m = m.initOptionEditor()

	// Add "true", then "false".
	m = send(t, m, "a")
	m.optInput.SetValue("true")
	m = send(t, m, "enter", "a")
	m.optInput.SetValue("false")
	m = send(t, m, "enter")

	if !slices.Equal(m.paramOptions, []string{"true", "false"}) {
		t.Fatalf("paramOptions = %v", m.paramOptions)
	}

	// Make "false" the default, then rename it — the default follows.
	m.paramDefault = "false"
	m.optCursor = 1
	m = send(t, m, "e")
	m.optInput.SetValue("no")
	m = send(t, m, "enter")

	if m.paramDefault != "no" {
		t.Errorf("paramDefault = %q, want %q", m.paramDefault, "no")
	}

	// Deleting the option that is the default clears the default.
	m = send(t, m, "d")
	if !slices.Equal(m.paramOptions, []string{"true"}) {
		t.Errorf("paramOptions = %v, want [true]", m.paramOptions)
	}
	if m.paramDefault != "" {
		t.Errorf("paramDefault = %q, want unset after deleting it", m.paramDefault)
	}
}

func TestOptionEditorRejectsEmptyAndDuplicate(t *testing.T) {
	m := editorWithParam(Parameter{Name: "BUILD", Options: []string{"true"}})
	m.paramCursor2 = pFieldOptions
	m = m.initOptionEditor()

	m = send(t, m, "a")
	m.optInput.SetValue("  ")
	m = send(t, m, "enter")
	if m.errText == "" || len(m.paramOptions) != 1 {
		t.Errorf("empty option should be rejected: err=%q options=%v", m.errText, m.paramOptions)
	}

	m.optInput.SetValue("true")
	m = send(t, m, "enter")
	if m.errText == "" || len(m.paramOptions) != 1 {
		t.Errorf("duplicate option should be rejected: err=%q options=%v", m.errText, m.paramOptions)
	}
}

func TestOptionEditorEmptyingOptionsRestoresFreeformDefault(t *testing.T) {
	m := editorWithParam(Parameter{Name: "BUILD", DefaultValue: "true", Options: []string{"true"}})
	m.paramCursor2 = pFieldOptions
	m = m.initOptionEditor()

	// Delete the only option, then go back to the parameter editor.
	m = send(t, m, "d", "esc")

	if m.state != stateParamEditor {
		t.Fatalf("state = %v, want the parameter editor", m.state)
	}
	if got := m.paramInputIdx(pFieldDefault); got != 2 {
		t.Errorf("default should be freeform again, paramInputIdx = %d", got)
	}
	// The default was the deleted option, so it is cleared rather than left invalid.
	if m.paramInputs[2].Value() != "" {
		t.Errorf("default input = %q, want empty", m.paramInputs[2].Value())
	}
}

func TestOptionEditorFreeformDefaultSurvivesRoundTrip(t *testing.T) {
	m := editorWithParam(Parameter{Name: "PLUGIN_BRANCH"})
	m.paramCursor2 = pFieldDefault
	m.paramInputs[2].SetValue("release-v1.4")

	// Visiting the options editor without adding anything must not lose the
	// freeform default value.
	m.paramCursor2 = pFieldOptions
	m = m.initOptionEditor()
	m = send(t, m, "esc")

	if m.paramInputs[2].Value() != "release-v1.4" {
		t.Errorf("default input = %q, want %q", m.paramInputs[2].Value(), "release-v1.4")
	}

	next, _ := m.saveParam()
	m = next.(model)
	if m.editTask.Parameters[0].DefaultValue != "release-v1.4" {
		t.Errorf("DefaultValue = %q", m.editTask.Parameters[0].DefaultValue)
	}
}
