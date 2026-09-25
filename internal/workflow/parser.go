package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser loads and parses workflow YAML files
type Parser struct {
	registry *WorkflowRegistry
}

// NewParser creates a new workflow parser
func NewParser() *Parser {
	return &Parser{
		registry: &WorkflowRegistry{
			Workflows: make(map[string]*Workflow),
		},
	}
}

// LoadFromFile loads workflows from YAML file
func (p *Parser) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	if err := yaml.Unmarshal(data, p.registry); err != nil {
		return fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Validate loaded workflows
	for name, workflow := range p.registry.Workflows {
		if err := p.validateWorkflow(workflow); err != nil {
			return fmt.Errorf("workflow %s validation failed: %w", name, err)
		}
	}

	return nil
}

// GetWorkflow retrieves a workflow by name
func (p *Parser) GetWorkflow(name string) *Workflow {
	return p.registry.Workflows[name]
}

// ListWorkflows returns all available workflows
func (p *Parser) ListWorkflows() map[string]*Workflow {
	return p.registry.Workflows
}

// validateWorkflow validates workflow structure
func (p *Parser) validateWorkflow(w *Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Validate each step
	seenIDs := make(map[string]bool)
	for _, step := range w.Steps {
		if step.ID == "" {
			return fmt.Errorf("step must have an ID")
		}
		if step.Tool == "" {
			return fmt.Errorf("step %s must have a tool", step.ID)
		}
		if seenIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		seenIDs[step.ID] = true
	}

	return nil
}

// SubstituteVariables replaces {{ var }} placeholders with context values
func (p *Parser) SubstituteVariables(input string, context map[string]interface{}) string {
	// Match {{ variable_name }}
	re := regexp.MustCompile(`\{\{\s*\.?(\w+)\s*\}\}`)

	result := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		varName := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{"))
		varName = strings.TrimPrefix(varName, ".")

		// Look up in context
		if val, ok := context[varName]; ok {
			return fmt.Sprintf("%v", val)
		}

		// Return placeholder if not found
		return match
	})

	return result
}

// SubstituteInParameters replaces variables in step parameters
func (p *Parser) SubstituteInParameters(params map[string]interface{}, context map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			result[key] = p.SubstituteVariables(v, context)
		case map[string]interface{}:
			result[key] = p.SubstituteInParameters(v, context)
		default:
			result[key] = value
		}
	}

	return result
}
