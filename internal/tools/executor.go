package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// Executor handles tool execution
type Executor struct {
	toolRegistry map[string]*models.ToolConfig
	timeout      time.Duration
}

// NewExecutor creates a new tool executor
func NewExecutor() *Executor {
	return &Executor{
		toolRegistry: make(map[string]*models.ToolConfig),
		timeout:      30 * time.Second,
	}
}

// RegisterTool adds a tool to the registry
func (e *Executor) RegisterTool(cfg *models.ToolConfig) error {
	// Check if binary exists
	_, err := exec.LookPath(cfg.BinaryPath)
	if err != nil {
		cfg.Available = false
		return fmt.Errorf("tool %s not found at %s: %w", cfg.Name, cfg.BinaryPath, err)
	}

	cfg.Available = true
	e.toolRegistry[cfg.Name] = cfg
	return nil
}

// GetToolConfig returns a tool configuration
func (e *Executor) GetToolConfig(toolName string) *models.ToolConfig {
	return e.toolRegistry[toolName]
}

// ListTools returns all registered tools
func (e *Executor) ListTools() []*models.ToolConfig {
	tools := make([]*models.ToolConfig, 0, len(e.toolRegistry))
	for _, cfg := range e.toolRegistry {
		tools = append(tools, cfg)
	}
	return tools
}

// RunTool executes a tool with given arguments
func (e *Executor) RunTool(toolName string, args []string, phase models.PhaseType, target string) (*models.ToolOutput, error) {
	cfg := e.toolRegistry[toolName]
	if cfg == nil {
		return nil, fmt.Errorf("tool %s not registered", toolName)
	}

	if !cfg.Available {
		return nil, fmt.Errorf("tool %s not available", toolName)
	}

	output := &models.ToolOutput{
		ID:        uuid.New().String(),
		ToolName:  toolName,
		Target:    target,
		Phase:     phase,
		Command:   cfg.BinaryPath,
		Args:      args,
		StartedAt: time.Now(),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, cfg.BinaryPath, args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command with timeout enforcement
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
		// Context timeout - kill process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		err = ctx.Err()
	}
	output.EndedAt = time.Now()
	output.Duration = int(output.EndedAt.Sub(output.StartedAt).Milliseconds())
	output.Stdout = stdout.String()
	output.Stderr = stderr.String()

	// Handle exit code
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			output.Error = fmt.Sprintf("tool %s timed out after %d seconds", toolName, cfg.Timeout)
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

// InitializeDefaultTools registers standard tools
func (e *Executor) InitializeDefaultTools() error {
	tools := []*models.ToolConfig{
		{
			Name:        "nmap",
			BinaryPath:  "nmap",
			Description: "Network mapper - service/OS discovery",
			Timeout:     60,
			OutputType:  "json",
		},
		{
			Name:        "nuclei",
			BinaryPath:  "nuclei",
			Description: "Vulnerability scanner - template-based",
			Timeout:     120,
			OutputType:  "json",
		},
		{
			Name:        "curl",
			BinaryPath:  "curl",
			Description: "HTTP client - web probing",
			Timeout:     10,
			OutputType:  "text",
		},
		{
			Name:        "dig",
			BinaryPath:  "dig",
			Description: "DNS lookup - domain enumeration",
			Timeout:     5,
			OutputType:  "text",
		},
		{
			Name:        "whois",
			BinaryPath:  "whois",
			Description: "WHOIS lookup - domain information",
			Timeout:     10,
			OutputType:  "text",
		},
	}

	for _, tool := range tools {
		// Register but don't fail if not available
		_ = e.RegisterTool(tool)
	}

	return nil
}
