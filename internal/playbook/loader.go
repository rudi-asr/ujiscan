package playbook

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PlaybookLoader loads playbooks from disk
type PlaybookLoader struct {
	playbooksDir string
}

// NewPlaybookLoader creates a new playbook loader
func NewPlaybookLoader(playbooksDir string) *PlaybookLoader {
	return &PlaybookLoader{
		playbooksDir: playbooksDir,
	}
}

// LoadPlaybook loads a playbook by name
func (pl *PlaybookLoader) LoadPlaybook(name string) (*Playbook, error) {
	// Find playbook file
	path := filepath.Join(pl.playbooksDir, name+".md")
	
	fmt.Printf("Loading playbook from: %s\n", path)
	
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbook file %s: %w", path, err)
	}

	return parsePlaybook(string(content))
}

// LoadAllPlaybooks loads all playbooks from directory
func (pl *PlaybookLoader) LoadAllPlaybooks() ([]*Playbook, error) {
	files, err := ioutil.ReadDir(pl.playbooksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbooks directory: %w", err)
	}

	playbooks := make([]*Playbook, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		// Extract playbook name without extension
		name := strings.TrimSuffix(file.Name(), ".md")
		pb, err := pl.LoadPlaybook(name)
		if err != nil {
			// Log but continue loading others
			fmt.Printf("Warning: failed to load playbook %s: %v\n", name, err)
			continue
		}

		playbooks = append(playbooks, pb)
	}

	return playbooks, nil
}

// parsePlaybook parses markdown playbook content
func parsePlaybook(content string) (*Playbook, error) {
	// Split YAML frontmatter from markdown
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid playbook format: missing YAML frontmatter")
	}

	// Parse YAML metadata
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Author      string `yaml:"author"`
		Version     string `yaml:"version"`
		EntryPhase  string `yaml:"entry_phase"`
	}

	yamlContent := parts[1]
	err := yaml.Unmarshal([]byte(yamlContent), &meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	playbook := NewPlaybook(meta.Name)
	playbook.Description = meta.Description
	playbook.Author = meta.Author
	playbook.Version = meta.Version
	playbook.EntryPhase = PhaseType(meta.EntryPhase)

	// Parse markdown content
	markdownContent := strings.Join(parts[2:], "---")
	err = parsePlaybookPhases(markdownContent, playbook)
	if err != nil {
		return nil, fmt.Errorf("failed to parse playbook phases: %w", err)
	}

	return playbook, nil
}

// parsePlaybookPhases extracts phases and steps from markdown
func parsePlaybookPhases(content string, pb *Playbook) error {
	fmt.Printf("[parser] Starting to parse playbook phases\\n")
	lines := strings.Split(content, "\n")
	var currentPhase PhaseType
	var currentStep *Step
	var stepCount int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "<!--") {
			continue
		}

		// Detect phase header (## recon, ## enum, etc)
		if strings.HasPrefix(line, "## ") {
			// Save previous step if exists BEFORE changing phase!
			if currentStep != nil {
				fmt.Printf("[parser] Saving step: %s (tool=%s, phase at save=%s) [triggered by phase change]\n", currentStep.ID, currentStep.Tool, currentPhase)
				stepCount++
				pb.AddStep(currentPhase, *currentStep)
				currentStep = nil
			}
		
			phaseName := strings.TrimPrefix(line, "## ")
			newPhase := PhaseType(strings.ToLower(strings.TrimSpace(phaseName)))
			fmt.Printf("[parser] Found phase header: %s → %s\n", phaseName, newPhase)
			currentPhase = newPhase
			if currentPhase != PhaseRecon && currentPhase != PhaseEnum && 
				currentPhase != PhaseExploit && currentPhase != PhaseVerify && 
				currentPhase != PhaseReport {
				// Unknown phase, skip
				fmt.Printf("[parser] Phase %s unknown, skipping\n", currentPhase)
				continue
			}
			continue
		}

		// Detect step header (### step-id)
		if strings.HasPrefix(line, "### ") {
			// Save previous step if exists
			if currentStep != nil {
				fmt.Printf("[parser] Saving step: %s (tool=%s, phase at save=%s)\n", currentStep.ID, currentStep.Tool, currentPhase)
				stepCount++
				pb.AddStep(currentPhase, *currentStep)
			}

			stepID := strings.TrimPrefix(line, "### ")
			stepID = strings.TrimSpace(stepID)
			fmt.Printf("[parser] Found step header: %s (phase=%s)\n", stepID, currentPhase)
			currentStep = &Step{
				ID:        stepID,
				NextSteps: make(map[string]string),
			}
			continue
		}

		// Parse step fields
		if currentStep != nil {
			if strings.HasPrefix(line, "Tool:") {
				currentStep.Tool = strings.TrimSpace(strings.TrimPrefix(line, "Tool:"))
			} else if strings.HasPrefix(line, "Args:") {
				argsStr := strings.TrimSpace(strings.TrimPrefix(line, "Args:"))
				// Parse YAML list format [arg1, arg2, ...]
				var args []string
				err := yaml.Unmarshal([]byte(argsStr), &args)
				if err == nil {
					currentStep.Args = args
				} else {
					// Fallback: split by comma and brackets
					argsStr = strings.Trim(argsStr, "[]")
					parts := strings.Split(argsStr, ",")
					for _, part := range parts {
						trimmed := strings.TrimSpace(part)
						trimmed = strings.Trim(trimmed, "\"'")
						if trimmed != "" {
							args = append(args, trimmed)
						}
					}
					currentStep.Args = args
				}
			} else if strings.HasPrefix(line, "Description:") {
				currentStep.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
			} else if strings.HasPrefix(line, "Condition:") {
				currentStep.Condition = strings.TrimSpace(strings.TrimPrefix(line, "Condition:"))
			}
		}
	}

	// Add last step
	if currentStep != nil && currentStep.ID != "" && currentStep.Tool != "" {
		fmt.Printf("[parser] Saving final step: %s (tool=%s)\n", currentStep.ID, currentStep.Tool)
		stepCount++
		pb.AddStep(currentPhase, *currentStep)
	}

	fmt.Printf("[parser] Playbook parsing complete - total steps parsed: %d\n", stepCount)
	return nil
}
