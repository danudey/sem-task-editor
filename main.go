package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	skipRepoCheck := flag.Bool("skip-repo-check", false, "Skip repository validation")
	debug := flag.Bool("debug", false, "Enable debug logging to sem-task-editor.log")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <yaml-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A TUI editor for Semaphore CI task definitions.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  SEM_TASK_EDITOR_SKIP_REPO_CHECK=1  Skip repository validation\n")
		fmt.Fprintf(os.Stderr, "  SEM_TASK_EDITOR_DEBUG=1            Enable debug logging\n")
	}
	flag.Parse()

	// Environment variable overrides
	if os.Getenv("SEM_TASK_EDITOR_SKIP_REPO_CHECK") == "1" {
		*skipRepoCheck = true
	}
	if os.Getenv("SEM_TASK_EDITOR_DEBUG") == "1" {
		*debug = true
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	filePath := flag.Arg(0)

	// Load the project YAML
	project, err := LoadProject(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", filePath, err)
		os.Exit(1)
	}

	// Repository validation
	if !*skipRepoCheck {
		if project.RepoURL == "" {
			fmt.Fprintf(os.Stderr, "Warning: no repository URL found in YAML, skipping repo check\n")
		} else {
			if err := CheckRepository(project.RepoURL); err != nil {
				fmt.Fprintf(os.Stderr, "Repository check failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "\nYou must run this command from a clone of the repository.\n")
				fmt.Fprintf(os.Stderr, "To skip this check, use --skip-repo-check or set SEM_TASK_EDITOR_SKIP_REPO_CHECK=1\n")
				os.Exit(1)
			}
		}
	}

	m := newModel(project, filePath, *skipRepoCheck, *debug)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
