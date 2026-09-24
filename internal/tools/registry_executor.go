// Package tools provides tool execution with registry-based management
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/rudi-asr/ujiscan/internal/registry"
)

// RegistryExecutor wraps registry and handles tool execution
type RegistryExecutor struct {
	reg         *registry.Registry
	toolCache   map[string]bool // cache of available tools
}

// NewRegistryExecutor creates a new registry-based executor
func NewRegistryExecutor(reg *registry.Registry) *RegistryExecutor {
	return &RegistryExecutor{
		reg:       reg,
		toolCache: make(map[string]bool),
	}
}

// GetTool returns a tool by ID from registry
func (re *RegistryExecutor) GetTool(toolID string) *registry.Tool {
	return re.reg.GetTool(toolID)
}

// GetToolsByPhase returns tools for a phase
func (re *RegistryExecutor) GetToolsByPhase(phaseID string) []*registry.Tool {
	return re.reg.GetToolsByPhase(phaseID)
}

// GetToolsByMode returns tools for a mode
func (re *RegistryExecutor) GetToolsByMode(modeID string) []*registry.Tool {
	return re.reg.ListToolsByMode(modeID)
}

// IsToolAvailable checks if a tool binary is available
func (re *RegistryExecutor) IsToolAvailable(toolID string) bool {
	// Check cache first
	if available, cached := re.toolCache[toolID]; cached {
		return available
	}

	tool := re.GetTool(toolID)
	if tool == nil {
		re.toolCache[toolID] = false
		return false
	}

	// Check if binary exists in PATH
	_, err := exec.LookPath(tool.Command.Command)
	available := err == nil
	re.toolCache[toolID] = available
	return available
}

// ListAvailableTools returns all tools that are available on this system
func (re *RegistryExecutor) ListAvailableTools() []*registry.Tool {
	var available []*registry.Tool
	for _, tool := range re.reg.Tools {
		if re.IsToolAvailable(tool.ID) {
			tool := tool // local copy
			tool.Available = true
			available = append(available, &tool)
		}
	}
	return available
}

// RunTool executes a tool from registry
func (re *RegistryExecutor) RunTool(
	toolID string,
	args []string,
	target string,
	variant string,
) (*models.ToolOutput, error) {
	tool := re.GetTool(toolID)
	if tool == nil {
		return nil, fmt.Errorf("tool %s not found in registry", toolID)
	}

	if !re.IsToolAvailable(toolID) {
		return nil, fmt.Errorf("tool %s is not available on this system", toolID)
	}

	// Get execution config
	execCfg := tool.Execution

	// Override timeout if variant specified
	if variant != "" {
		variantCfg, ok := tool.Variants[variant]
		if !ok {
			return nil, fmt.Errorf("variant %s not found for tool %s", variant, toolID)
		}
		if variantCfg.Timeout > 0 {
			execCfg.Timeout = variantCfg.Timeout
		}
		// Prepend variant args
		args = append(variantCfg.CommandArgs, args...)
	}

	// Create tool output record
	output := &models.ToolOutput{
		ID:        uuid.New().String(),
		ToolName:  tool.Name,
		Target:    target,
		Phase:     models.PhaseType(tool.Phase),
		Command:   tool.Command.Command,
		Args:      args,
		StartedAt: time.Now(),
	}

	// Create context with timeout
	timeout := time.Duration(execCfg.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, tool.Command.Command, args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	var err error
	select {
	case err = <-done:
		// Command finished
	case <-ctx.Done():
		// Timeout - kill process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		err = ctx.Err()
	}

	// Populate output
	output.EndedAt = time.Now()
	output.Duration = int(output.EndedAt.Sub(output.StartedAt).Milliseconds())
	output.Stdout = stdout.String()
	output.Stderr = stderr.String()

	// Handle exit code
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			output.Error = fmt.Sprintf("tool %s timed out after %d seconds", tool.Name, execCfg.Timeout)
			output.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
			output.Error = exitErr.Error()
		} else {
			output.ExitCode = -1
			output.Error = err.Error()
		}
		output.Success = false
	} else {
		output.ExitCode = 0
		output.Success = true
	}

	return output, nil
}

// ExecutePhase runs all tools in a phase sequentially or in parallel
func (re *RegistryExecutor) ExecutePhase(
	phaseID string,
	modeID string,
	target string,
	parallel bool,
) ([]*models.ToolOutput, error) {
	tools := re.GetToolsByPhase(phaseID)
	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools found for phase %s", phaseID)
	}

	var outputs []*models.ToolOutput

	if parallel {
		// Run tools in parallel
		type result struct {
			output *models.ToolOutput
			err    error
		}
		results := make(chan result, len(tools))

		for _, tool := range tools {
			go func(t *registry.Tool) {
				out, err := re.RunTool(t.ID, []string{target}, target, "")
				results <- result{output: out, err: err}
			}(tool)
		}

		for i := 0; i < len(tools); i++ {
			res := <-results
			if res.err != nil {
				// Log error but continue
				fmt.Printf("Error running tool: %v\n", res.err)
				continue
			}
			outputs = append(outputs, res.output)
		}
	} else {
		// Run tools sequentially
		for _, tool := range tools {
			out, err := re.RunTool(tool.ID, []string{target}, target, "")
			if err != nil {
				// Log error but continue
				fmt.Printf("Error running tool %s: %v\n", tool.Name, err)
				continue
			}
			outputs = append(outputs, out)
		}
	}

	return outputs, nil
}

// ExecuteMode runs full execution for a mode (all phases in order)
func (re *RegistryExecutor) ExecuteMode(
	modeID string,
	target string,
) ([]*models.ToolOutput, error) {
	mode := re.reg.GetMode(modeID)
	if mode == nil {
		return nil, fmt.Errorf("mode %s not found", modeID)
	}

	var allOutputs []*models.ToolOutput

	for _, phaseExec := range mode.ExecutionOrder {
		outputs, err := re.ExecutePhase(phaseExec.Phase, modeID, target, phaseExec.Parallel)
		if err != nil {
			fmt.Printf("Error executing phase %s: %v\n", phaseExec.Phase, err)
			continue
		}
		allOutputs = append(allOutputs, outputs...)
	}

	return allOutputs, nil
}

// GetModeForTarget detects appropriate mode based on target
func (re *RegistryExecutor) GetModeForTarget(target string) string {
	// Simple detection logic
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return "web-app"
	}
	if strings.Contains(target, "/") {
		return "network"
	}
	if strings.Contains(target, ".") && !strings.Contains(target, ":") {
		// Looks like domain or IP
		return "web-app"
	}
	return "network"
}

// ValidateRegistry checks registry integrity
func (re *RegistryExecutor) ValidateRegistry() error {
	errors := re.reg.ValidateRegistry()
	if len(errors) > 0 {
		var msg strings.Builder
		msg.WriteString("Registry validation errors:\n")
		for section, errs := range errors {
			msg.WriteString(fmt.Sprintf("  %s:\n", section))
			for _, err := range errs {
				msg.WriteString(fmt.Sprintf("    - %s\n", err))
			}
		}
		msgStr := msg.String()
		return fmt.Errorf("%s", msgStr)
	}
	return nil
}
