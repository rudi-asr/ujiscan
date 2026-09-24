// Package registry provides tool loading and management
package registry

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadRegistry loads tool definitions from tools.yaml
func LoadRegistry(filePath string) (*Registry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry YAML: %w", err)
	}

	// Build runtime indices for fast lookup
	reg.BuildIndices()

	return &reg, nil
}

// SaveRegistry saves registry to YAML file
func SaveRegistry(filePath string, reg *Registry) error {
	data, err := yaml.Marshal(reg)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// AddTool adds a new tool to the registry
func (r *Registry) AddTool(tool *Tool) error {
	if r.GetTool(tool.ID) != nil {
		return fmt.Errorf("tool with id %s already exists", tool.ID)
	}

	r.Tools = append(r.Tools, *tool)
	r.BuildIndices()

	return nil
}

// RemoveTool removes a tool from the registry
func (r *Registry) RemoveTool(id string) error {
	found := false
	for i, tool := range r.Tools {
		if tool.ID == id {
			r.Tools = append(r.Tools[:i], r.Tools[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("tool with id %s not found", id)
	}

	r.BuildIndices()
	return nil
}

// UpdateTool updates an existing tool
func (r *Registry) UpdateTool(id string, tool *Tool) error {
	tool.ID = id // ensure ID matches
	
	for i, t := range r.Tools {
		if t.ID == id {
			r.Tools[i] = *tool
			r.BuildIndices()
			return nil
		}
	}

	return fmt.Errorf("tool with id %s not found", id)
}

// ListToolsByMode returns all tools applicable to a mode
func (r *Registry) ListToolsByMode(modeID string) []*Tool {
	result := make([]*Tool, 0)
	for _, tool := range r.Tools {
		for _, mode := range tool.ModeApplicability {
			if mode == modeID {
				result = append(result, &tool)
				break
			}
		}
	}
	return result
}

// ListToolsByCategory returns all tools in a category
func (r *Registry) ListToolsByCategory(category string) []*Tool {
	result := make([]*Tool, 0)
	for _, tool := range r.Tools {
		if tool.Category == category {
			result = append(result, &tool)
		}
	}
	return result
}

// GetExecutionOrderForMode returns the execution order for a mode
func (r *Registry) GetExecutionOrderForMode(modeID string) []PhaseExecution {
	mode := r.GetMode(modeID)
	if mode == nil {
		return nil
	}
	return mode.ExecutionOrder
}

// ValidateTool checks if a tool is properly configured
func (r *Registry) ValidateTool(tool *Tool) []string {
	var errors []string

	if tool.ID == "" {
		errors = append(errors, "tool ID is required")
	}
	if tool.Name == "" {
		errors = append(errors, "tool name is required")
	}
	if tool.Phase == "" {
		errors = append(errors, "tool phase is required")
	}
	if tool.Command.Command == "" {
		errors = append(errors, "command is required")
	}
	if tool.Execution.Timeout == 0 {
		errors = append(errors, "execution timeout is required")
	}
	if len(tool.TargetTypes) == 0 {
		errors = append(errors, "at least one target type is required")
	}

	// Validate phase exists
	if r.GetPhase(tool.Phase) == nil {
		errors = append(errors, fmt.Sprintf("phase %s not found in registry", tool.Phase))
	}

	// Validate target types
	validTypes := map[string]bool{
		"domain":    true,
		"url":       true,
		"ip":        true,
		"ip_range":  true,
		"cidr":      true,
		"binary":    true,
	}
	for _, ttype := range tool.TargetTypes {
		if !validTypes[ttype] {
			errors = append(errors, fmt.Sprintf("invalid target type: %s", ttype))
		}
	}

	return errors
}

// ValidateRegistry checks if registry is valid
func (r *Registry) ValidateRegistry() map[string][]string {
	errors := make(map[string][]string)

	// Validate phases
	for _, phase := range r.Phases {
		if phase.ID == "" {
			errors["phases"] = append(errors["phases"], "phase ID is required")
		}
	}

	// Validate tools
	for i, tool := range r.Tools {
		toolErrors := r.ValidateTool(&tool)
		if len(toolErrors) > 0 {
			errors[fmt.Sprintf("tool_%d_%s", i, tool.ID)] = toolErrors
		}
	}

	// Validate modes
	for _, mode := range r.Modes {
		if mode.ID == "" {
			errors["modes"] = append(errors["modes"], "mode ID is required")
		}
		for _, exec := range mode.ExecutionOrder {
			if exec.Phase == "" {
				errors["modes"] = append(errors["modes"], fmt.Sprintf("execution order phase is required in mode %s", mode.ID))
			}
			if len(exec.Tools) == 0 {
				errors["modes"] = append(errors["modes"], fmt.Sprintf("execution order tools required in mode %s phase %s", mode.ID, exec.Phase))
			}
		}
	}

	return errors
}
