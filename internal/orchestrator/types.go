package orchestrator

import (
	"time"
)

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "PENDING"
	StatusRunning   ExecutionStatus = "RUNNING"
	StatusCompleted ExecutionStatus = "COMPLETED"
	StatusFailed    ExecutionStatus = "FAILED"
	StatusRetrying  ExecutionStatus = "RETRYING"
	StatusCanceled   ExecutionStatus = "CANCELED"
)

// StepExecution represents a single step execution
type StepExecution struct {
	// Unique ID
	ID string `json:"id"`
	// Step name
	Name string `json:"name"`
	// Current status
	Status ExecutionStatus `json:"status"`
	// Start time
	StartTime *time.Time `json:"start_time,omitempty"`
	// End time
	EndTime *time.Time `json:"end_time,omitempty"`
	// Duration (seconds)
	Duration int `json:"duration"`
	// Tool output
	Output string `json:"output,omitempty"`
	// Error message (if failed)
	Error string `json:"error,omitempty"`
	// Retry count
	RetryCount int `json:"retry_count"`
	// Raw findings from tool
	Findings []Finding `json:"findings,omitempty"`
}

// PhaseExecution represents execution of a phase
type PhaseExecution struct {
	// Phase name
	Phase string `json:"phase"`
	// Status
	Status ExecutionStatus `json:"status"`
	// Steps in this phase
	Steps []*StepExecution `json:"steps"`
	// Start time
	StartTime *time.Time `json:"start_time,omitempty"`
	// End time
	EndTime *time.Time `json:"end_time,omitempty"`
}

// EngagementExecution represents full engagement execution state
type EngagementExecution struct {
	// Unique ID
	ID string `json:"id"`
	// Engagement target
	Target string `json:"target"`
	// Overall status
	Status ExecutionStatus `json:"status"`
	// Execution plan ID
	PlanID string `json:"plan_id"`
	// Phases
	Phases []*PhaseExecution `json:"phases"`
	// Start time
	StartTime time.Time `json:"start_time"`
	// End time
	EndTime *time.Time `json:"end_time,omitempty"`
	// Progress (0-100)
	Progress int `json:"progress"`
	// Current step
	CurrentStep string `json:"current_step,omitempty"`
	// Aggregated findings
	Findings []Finding `json:"findings,omitempty"`
}

// Finding represents a discovered security finding
type Finding struct {
	// Unique ID
	ID string `json:"id"`
	// Discovery source (tool name)
	Source string `json:"source"`
	// Finding type (XSS, SQLi, etc)
	Type string `json:"type"`
	// Description
	Description string `json:"description"`
	// Severity (1-10)
	Severity int `json:"severity"`
	// CWE ID (if applicable)
	CWE string `json:"cwe,omitempty"`
	// Evidence/POC
	Evidence string `json:"evidence,omitempty"`
	// Remediation
	Remediation string `json:"remediation,omitempty"`
	// Timestamp
	DiscoveredAt time.Time `json:"discovered_at"`
	// Raw data from tool
	RawData map[string]interface{} `json:"raw_data,omitempty"`
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	// Max parallel tasks
	MaxParallel int
	// Task timeout (seconds)
	TaskTimeout int
	// Retry policy
	MaxRetries int
	RetryDelay int // seconds
}
