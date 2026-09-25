package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Executor runs tools and captures their output
type Executor struct {
	registry *Registry
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry: registry,
	}
}

// Execute runs a tool and returns results
func (e *Executor) Execute(ctx context.Context, toolName string, params map[string]string) (*ToolExecutionResult, error) {
	result := &ToolExecutionResult{
		ToolName: toolName,
		Success:  false,
	}

	// Get tool definition
	fmt.Printf("[executor] Looking up tool: '%s'\n", toolName)
	tool := e.registry.GetTool(toolName)
	if tool == nil {
		result.Error = fmt.Sprintf("Tool %s not found in registry", toolName)
		return result, fmt.Errorf(result.Error)
	}
	fmt.Printf("[executor] Found tool, execute_template: '%s'\n", tool.ExecuteTemplate)

	// Check if installed
	if !e.registry.IsInstalled(toolName) {
		if err := e.InstallTool(ctx, toolName); err != nil {
			result.Error = fmt.Sprintf("Failed to install tool: %v", err)
			return result, err
		}
	}

	// Build execute command by substituting params
	cmdStr := e.buildCommand(tool.ExecuteTemplate, params)

	// Execute
	start := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

	output, err := cmd.CombinedOutput()
	result.Duration = int(time.Since(start).Seconds())

	fmt.Printf("[executor] Command: %s\n", cmdStr)
	fmt.Printf("[executor] Output length: %d bytes\n", len(output))
	fmt.Printf("[executor] Output preview: %s\n", string(output)[:min(len(output), 100)])

	if err != nil {
		if ctx.Err() != nil {
			result.Error = "Execution canceled"
		} else {
			result.Error = fmt.Sprintf("Execution failed: %v", err)
		}
		result.Stderr = string(output)
		return result, err
	}

	result.Success = true
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.Stdout = string(output)

	// Update last used
	e.registry.UpdateLastUsed(toolName)

	return result, nil
}

// min returns minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InstallTool installs a tool
func (e *Executor) InstallTool(ctx context.Context, toolName string) error {
	tool := e.registry.GetTool(toolName)
	if tool == nil {
		return fmt.Errorf("Tool %s not found", toolName)
	}

	// Check platform compatibility
	if !e.isPlatformSupported(tool) {
		return fmt.Errorf("Tool %s not supported on this platform", toolName)
	}

	// Run install command
	cmd := exec.CommandContext(ctx, "sh", "-c", tool.InstallCommand)
	if output, err := cmd.CombinedOutput(); err != nil {
		e.registry.SetInstallationError(toolName, fmt.Sprintf("Install failed: %s", string(output)))
		return err
	}

	// Verify installation
	if err := e.VerifyTool(ctx, toolName); err != nil {
		e.registry.SetInstallationError(toolName, fmt.Sprintf("Verification failed: %v", err))
		return err
	}

	// Mark as installed
	e.registry.SetInstalled(toolName, "latest", "")
	return nil
}

// VerifyTool checks if a tool is properly installed
func (e *Executor) VerifyTool(ctx context.Context, toolName string) error {
	tool := e.registry.GetTool(toolName)
	if tool == nil {
		return fmt.Errorf("Tool %s not found", toolName)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", tool.VerifyCommand)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	return nil
}

// buildCommand substitutes parameters into execute template
func (e *Executor) buildCommand(template string, params map[string]string) string {
	fmt.Printf("[executor-build] Template: '%s'\n", template)
	fmt.Printf("[executor-build] Params: %v\n", params)
	cmd := template
	for key, value := range params {
		placeholder := "{{ " + key + " }}"
		fmt.Printf("[executor-build] Replacing '%s' with '%s'\n", placeholder, value)
		cmd = strings.ReplaceAll(cmd, placeholder, value)
	}
	fmt.Printf("[executor-build] Result: '%s'\n", cmd)
	return cmd
}

// isPlatformSupported checks if tool runs on current OS
func (e *Executor) isPlatformSupported(tool *Tool) bool {
	currentOS := os.Getenv("GOOS")
	if currentOS == "" {
		// Default to current system
		return true
	}

	for _, platform := range tool.Platforms {
		if platform == currentOS || platform == "all" {
			return true
		}
	}
	return false
}

// ListTools returns all available tools (stub for legacy API)
func (e *Executor) ListTools() map[string]*Tool {
	if e.registry != nil {
		return e.registry.ListTools()
	}
	return make(map[string]*Tool)
}
