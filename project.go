package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Task represents a single task in the Semaphore project.
type Task struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	Scheduled    bool        `yaml:"scheduled"`
	ID           string      `yaml:"id,omitempty"`
	Branch       string      `yaml:"branch"`
	At           string      `yaml:"at"`
	PipelineFile string      `yaml:"pipeline_file"`
	Status       string      `yaml:"status"`
	Parameters   []Parameter `yaml:"parameters,omitempty"`
}

// Parameter represents a task parameter.
type Parameter struct {
	Name        string `yaml:"name"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
	// DefaultValue is the value used when the task runs without an explicit
	// value. If Options is non-empty it must be one of them.
	DefaultValue string `yaml:"default_value,omitempty"`
	// Options, when non-empty, restricts DefaultValue (and the values a user
	// may pick when launching the task) to this list.
	Options []string `yaml:"options,omitempty"`
}

// Project represents a Semaphore project YAML file.
type Project struct {
	OriginalContent string
	RootNode        *yaml.Node
	RepoURL         string
	ProjectName     string
	Tasks           []Task
}

// LoadProject reads and parses a Semaphore project YAML file.
func LoadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("invalid YAML document")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping at document root")
	}

	p := &Project{
		OriginalContent: string(data),
		RootNode:        &doc,
	}

	metaNode := findMappingValue(root, "metadata")
	if metaNode != nil {
		p.ProjectName = scalarValue(findMappingValue(metaNode, "name"))
	}

	specNode := findMappingValue(root, "spec")
	if specNode == nil {
		return nil, fmt.Errorf("missing 'spec' in YAML")
	}

	repoNode := findMappingValue(specNode, "repository")
	if repoNode != nil {
		p.RepoURL = scalarValue(findMappingValue(repoNode, "url"))
	}

	tasksNode := findMappingValue(specNode, "tasks")
	if tasksNode == nil {
		return nil, fmt.Errorf("missing 'tasks' in spec")
	}

	if tasksNode.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("tasks should be a sequence")
	}

	for _, taskNode := range tasksNode.Content {
		var t Task
		if err := taskNode.Decode(&t); err != nil {
			return nil, fmt.Errorf("decoding task: %w", err)
		}
		p.Tasks = append(p.Tasks, t)
	}

	return p, nil
}

// GenerateOutput produces the modified YAML content with updated tasks.
func (p *Project) GenerateOutput(tasks []Task) (string, error) {
	// Re-parse the original to get a fresh node tree
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(p.OriginalContent), &doc); err != nil {
		return "", fmt.Errorf("re-parsing YAML: %w", err)
	}

	root := doc.Content[0]
	specNode := findMappingValue(root, "spec")
	tasksNode := findMappingValue(specNode, "tasks")

	// Sync the scheduled flag with the at field
	tasksCopy := make([]Task, len(tasks))
	copy(tasksCopy, tasks)
	for i := range tasksCopy {
		tasksCopy[i].Scheduled = tasksCopy[i].At != ""
	}

	// Marshal tasks and parse back as nodes to replace in the tree
	tasksData, err := yaml.Marshal(tasksCopy)
	if err != nil {
		return "", fmt.Errorf("marshaling tasks: %w", err)
	}

	var newSeq yaml.Node
	if err := yaml.Unmarshal(tasksData, &newSeq); err != nil {
		return "", fmt.Errorf("parsing marshaled tasks: %w", err)
	}

	if newSeq.Kind == yaml.DocumentNode && len(newSeq.Content) > 0 {
		tasksNode.Content = newSeq.Content[0].Content
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", fmt.Errorf("encoding YAML: %w", err)
	}
	enc.Close()

	// Post-process to use compact sequence style (sequences at same indent as parent key)
	// which matches the original file's formatting convention.
	result := compactSequences(buf.String())

	// Preserve trailing whitespace from the original
	origTrimmed := strings.TrimRight(p.OriginalContent, "\n")
	origTrailing := p.OriginalContent[len(origTrimmed):]
	resultTrimmed := strings.TrimRight(result, "\n")
	result = resultTrimmed + origTrailing

	return result, nil
}

// Save writes the modified project to the given path.
func (p *Project) Save(path string, tasks []Task) error {
	output, err := p.GenerateOutput(tasks)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(output), 0644)
}

// compactSequences converts yaml.v3 indented sequence style to compact style.
// e.g., "key:\n  - item" becomes "key:\n- item" (sequence at same indent as key).
// Runs iteratively to handle nested sequences.
func compactSequences(input string) string {
	for {
		result := compactSequencesOnce(input)
		if result == input {
			return result
		}
		input = result
	}
}

func compactSequencesOnce(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	i := 0
	for i < len(lines) {
		line := lines[i]

		trimmed := strings.TrimSpace(line)

		// Check if this line is a mapping key ending with ":" (indicating a block value follows)
		if trimmed == "" || !strings.HasSuffix(trimmed, ":") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "- ") {
			result = append(result, line)
			i++
			continue
		}

		keyIndent := len(line) - len(strings.TrimLeft(line, " "))
		seqIndent := keyIndent + 2

		// Look ahead for a sequence start at the expected indent
		if i+1 < len(lines) {
			nextLine := lines[i+1]
			nextTrimmed := strings.TrimLeft(nextLine, " ")
			nextIndent := len(nextLine) - len(nextTrimmed)

			if nextIndent == seqIndent && strings.HasPrefix(nextTrimmed, "- ") {
				// Found a sequence — un-indent the block by 2
				result = append(result, line)
				i++

				for i < len(lines) {
					l := lines[i]
					if strings.TrimSpace(l) == "" {
						result = append(result, l)
						i++
						continue
					}

					lIndent := len(l) - len(strings.TrimLeft(l, " "))
					if lIndent < seqIndent {
						break
					}

					newIndent := max(lIndent-2, 0)
					result = append(result, strings.Repeat(" ", newIndent)+strings.TrimLeft(l, " "))
					i++
				}
				continue
			}
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func scalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}
