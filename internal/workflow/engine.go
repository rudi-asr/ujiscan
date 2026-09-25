package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Engine executes workflows step-by-step
type Engine struct {
	parser *Parser
	mu     sync.RWMutex
}

// NewEngine creates a new workflow engine
func NewEngine(parser *Parser) *Engine {
	return &Engine{
		parser: parser,
	}
}

// Execute runs a workflow and returns execution result
func (e *Engine) Execute(ctx context.Context, workflowName string, inputContext map[string]interface{}) (*WorkflowExecution, error) {
	workflow := e.parser.GetWorkflow(workflowName)
	if workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowName)
	}

	execution := &WorkflowExecution{
		ID:           uuid.New().String(),
		WorkflowName: workflowName,
		Status:       "RUNNING",
		StartTime:    time.Now(),
		StepResults:  make(map[string]interface{}),
		Context:      make(map[string]interface{}),
	}

	// Copy input context
	for k, v := range inputContext {
		execution.Context[k] = v
	}

	// Execute workflow steps
	err := e.executeSteps(ctx, workflow, execution)

	// Mark completion
	now := time.Now()
	execution.EndTime = &now

	if err != nil {
		execution.Status = "FAILED"
		execution.Error = err.Error()
	} else {
		execution.Status = "COMPLETED"
	}

	return execution, err
}

// executeSteps runs all steps in workflow
func (e *Engine) executeSteps(ctx context.Context, workflow *Workflow, execution *WorkflowExecution) error {
	for _, step := range workflow.Steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check condition
		if step.When != "" {
			if !e.evaluateCondition(step.When, execution.Context) {
				continue
			}
		}

		// Substitute variables in parameters
		params := e.parser.SubstituteInParameters(step.Parameters, execution.Context)

		// Execute step (simulated)
		result := e.executeStep(ctx, step, params)

		// Store result
		execution.StepResults[step.ID] = result

		// Update context with captured value
		if step.Capture != "" && result.CapturedValue != "" {
			execution.Context[step.Capture] = result.CapturedValue
		}

		// Check for errors
		if !result.Success && step.MaxRetries == 0 {
			return fmt.Errorf("step %s failed: %s", step.ID, result.Error)
		}
	}

	return nil
}

// executeStep simulates executing a single step
func (e *Engine) executeStep(ctx context.Context, step Step, params map[string]interface{}) StepResult {
	start := time.Now()

	result := StepResult{
		StepID:  step.ID,
		Success: true,
	}

	// Simulate tool execution
	select {
	case <-ctx.Done():
		result.Success = false
		result.Error = "execution canceled"
		return result
	case <-time.After(time.Duration(50) * time.Millisecond):
	}

	// Simulate output
	result.Output = fmt.Sprintf("Step %s (%s) executed with params: %v", step.ID, step.Tool, params)
	result.CapturedValue = fmt.Sprintf("result_%s", step.ID)
	result.Duration = int(time.Since(start).Seconds())

	return result
}

// evaluateCondition evaluates a condition expression
func (e *Engine) evaluateCondition(condition string, context map[string]interface{}) bool {
	// Simple condition evaluation
	// Examples: "previous_step_success", "target_type=WEB"

	// If contains "=", do equality check
	if strings.Contains(condition, "=") {
		parts := strings.Split(condition, "=")
		if len(parts) != 2 {
			return false
		}

		varName := strings.TrimSpace(parts[0])
		expectedVal := strings.TrimSpace(parts[1])

		if val, ok := context[varName]; ok {
			return fmt.Sprintf("%v", val) == expectedVal
		}
		return false
	}

	// Otherwise, check for truthy value
	if val, ok := context[condition]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != "" && v != "false" && v != "0"
		case int:
			return v != 0
		default:
			return true
		}
	}

	return false
}

// GetExecution retrieves execution details
func (e *Engine) GetExecution(executionID string) *WorkflowExecution {
	// In production, would look up in database
	return nil
}

// ListAvailableWorkflows returns all workflows
func (e *Engine) ListAvailableWorkflows() map[string]*Workflow {
	return e.parser.ListWorkflows()
}
