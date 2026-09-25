package workflow

import "time"

// Workflow represents a complete workflow definition
type Workflow struct {
	// Workflow name
	Name string `yaml:"name"`
	// Description
	Description string `yaml:"description"`
	// Version
	Version string `yaml:"version"`
	// Timeout (seconds)
	Timeout int `yaml:"timeout"`
	// Steps in order
	Steps []Step `yaml:"steps"`
	// Metadata
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// Step represents a single workflow step
type Step struct {
	// Unique step ID
	ID string `yaml:"id"`
	// Step name
	Name string `yaml:"name"`
	// Tool to execute
	Tool string `yaml:"tool"`
	// Parameters (can use {{ template }} variables)
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
	// Timeout override (seconds)
	Timeout int `yaml:"timeout,omitempty"`
	// Retry count
	MaxRetries int `yaml:"max_retries,omitempty"`
	// Can run in parallel
	Parallel bool `yaml:"parallel,omitempty"`
	// Condition to execute (e.g., "previous_step_success")
	When string `yaml:"when,omitempty"`
	// Output capture
	Capture string `yaml:"capture,omitempty"`
}

// WorkflowExecution tracks execution state
type WorkflowExecution struct {
	// Execution ID
	ID string `json:"id"`
	// Workflow name
	WorkflowName string `json:"workflow_name"`
	// Status (RUNNING, COMPLETED, FAILED)
	Status string `json:"status"`
	// Start time
	StartTime time.Time `json:"start_time"`
	// End time
	EndTime *time.Time `json:"end_time,omitempty"`
	// Step results
	StepResults map[string]interface{} `json:"step_results"`
	// Error message
	Error string `json:"error,omitempty"`
	// Context variables
	Context map[string]interface{} `json:"context,omitempty"`
}

// StepResult represents the result of step execution
type StepResult struct {
	// Step ID
	StepID string `json:"step_id"`
	// Success status
	Success bool `json:"success"`
	// Output
	Output string `json:"output,omitempty"`
	// Error message
	Error string `json:"error,omitempty"`
	// Duration (seconds)
	Duration int `json:"duration"`
	// Captured value
	CapturedValue string `json:"captured_value,omitempty"`
}

// WorkflowRegistry holds all workflow definitions
type WorkflowRegistry struct {
	Workflows map[string]*Workflow `yaml:"workflows"`
}
