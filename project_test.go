package main

import (
	"strings"
	"testing"
)

func TestLoadProject(t *testing.T) {
	p, err := LoadProject("sample.yml")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	if p.ProjectName != "calico-private" {
		t.Errorf("ProjectName = %q, want %q", p.ProjectName, "calico-private")
	}

	if p.RepoURL != "git@github.com:tigera/calico-private.git" {
		t.Errorf("RepoURL = %q", p.RepoURL)
	}

	if len(p.Tasks) != 14 {
		t.Errorf("got %d tasks, want 14", len(p.Tasks))
	}

	// Check first task
	first := p.Tasks[0]
	if first.Name != "create release branch" {
		t.Errorf("first task name = %q", first.Name)
	}
	if first.Status != "INACTIVE" {
		t.Errorf("first task status = %q", first.Status)
	}
	if first.Scheduled {
		t.Error("first task should not be scheduled")
	}
	if len(first.Parameters) != 1 {
		t.Errorf("first task has %d parameters, want 1", len(first.Parameters))
	}

	// Check a scheduled task
	second := p.Tasks[1]
	if second.At != "30 3 * * *" {
		t.Errorf("second task at = %q", second.At)
	}
	if !second.Scheduled {
		t.Error("second task should be scheduled")
	}
}

func TestGenerateOutput(t *testing.T) {
	p, err := LoadProject("sample.yml")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	// Generate output with unchanged tasks
	output, err := p.GenerateOutput(p.Tasks)
	if err != nil {
		t.Fatalf("GenerateOutput: %v", err)
	}

	if !strings.Contains(output, "calico-private") {
		t.Error("output should contain project name")
	}

	if !strings.Contains(output, "hashrelease: master") {
		t.Error("output should contain task name")
	}

	// Verify comments are preserved
	if !strings.Contains(output, "Editing Projects/calico-private") {
		t.Error("output should preserve comments")
	}
}

func TestGenerateOutputNewTask(t *testing.T) {
	p, err := LoadProject("sample.yml")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	// Add a new task
	tasks := make([]Task, len(p.Tasks))
	copy(tasks, p.Tasks)
	tasks = append(tasks, Task{
		Name:         "test task",
		Description:  "a test",
		Branch:       "master",
		At:           "0 12 * * 1-5",
		PipelineFile: ".semaphore/test.yml",
		Status:       "ACTIVE",
	})

	output, err := p.GenerateOutput(tasks)
	if err != nil {
		t.Fatalf("GenerateOutput: %v", err)
	}

	if !strings.Contains(output, "test task") {
		t.Error("output should contain new task")
	}

	// New task should not have an ID
	if strings.Contains(output, "id: ") {
		// This is OK - existing tasks have IDs. Let's check more carefully
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.Contains(line, "test task") {
				// Check the next few lines for an id field
				for j := i + 1; j < i+10 && j < len(lines); j++ {
					trimmed := strings.TrimSpace(lines[j])
					if strings.HasPrefix(trimmed, "id:") {
						t.Error("new task should not have an id field")
					}
					if strings.HasPrefix(trimmed, "- name:") || trimmed == "" {
						break
					}
				}
				break
			}
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p, err := LoadProject("sample.yml")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	output, err := p.GenerateOutput(p.Tasks)
	if err != nil {
		t.Fatalf("GenerateOutput: %v", err)
	}

	if output != p.OriginalContent {
		// Show the first difference
		origLines := strings.Split(p.OriginalContent, "\n")
		outLines := strings.Split(output, "\n")

		for i := 0; i < len(origLines) || i < len(outLines); i++ {
			var orig, out string
			if i < len(origLines) {
				orig = origLines[i]
			}
			if i < len(outLines) {
				out = outLines[i]
			}
			if orig != out {
				t.Errorf("First difference at line %d:\n  original: %q\n  output:   %q", i+1, orig, out)
				break
			}
		}

		t.Errorf("Round-trip output differs from original (%d vs %d bytes)", len(output), len(p.OriginalContent))
	}
}

func TestCronParsing(t *testing.T) {
	tests := []struct {
		input string
		days  [7]bool
	}{
		{"*", [7]bool{}},
		{"1-5", [7]bool{false, true, true, true, true, true, false}},
		{"0,6", [7]bool{true, false, false, false, false, false, true}},
		{"1", [7]bool{false, true, false, false, false, false, false}},
		{"1-6", [7]bool{false, true, true, true, true, true, true}},
	}

	for _, tt := range tests {
		got := parseDOW(tt.input)
		if got != tt.days {
			t.Errorf("parseDOW(%q) = %v, want %v", tt.input, got, tt.days)
		}
	}
}

func TestFormatDOW(t *testing.T) {
	tests := []struct {
		days [7]bool
		want string
	}{
		{[7]bool{}, "*"},
		{[7]bool{true, true, true, true, true, true, true}, "*"},
		{[7]bool{false, true, true, true, true, true, false}, "1-5"},
		{[7]bool{true, false, false, false, false, false, true}, "0,6"},
		{[7]bool{false, true, false, false, false, false, false}, "1"},
	}

	for _, tt := range tests {
		got := formatDOW(tt.days)
		if got != tt.want {
			t.Errorf("formatDOW(%v) = %q, want %q", tt.days, got, tt.want)
		}
	}
}

func TestNormalizeRepoID(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"git@github.com:tigera/calico-private.git", "tigera/calico-private"},
		{"https://github.com/tigera/calico-private.git", "tigera/calico-private"},
		{"https://github.com/tigera/calico-private", "tigera/calico-private"},
	}

	for _, tt := range tests {
		got := normalizeRepoID(tt.input)
		if got != tt.want {
			t.Errorf("normalizeRepoID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
