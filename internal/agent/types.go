package agent

import (
	"context"
	"errors"
)

// AgentType defines the type of agent
type AgentType string

const (
	AgentTypeReconnaissance AgentType = "reconnaissance"
	AgentTypeScanner        AgentType = "scanner"
	AgentTypeAnalyzer       AgentType = "analyzer"
	AgentTypeReporter       AgentType = "reporter"
	AgentTypeOrchestrator   AgentType = "orchestrator"
)

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	StatusIdle        AgentStatus = "idle"
	StatusRunning     AgentStatus = "running"
	StatusPaused      AgentStatus = "paused"
	StatusCompleted   AgentStatus = "completed"
	StatusError       AgentStatus = "error"
	StatusTerminated  AgentStatus = "terminated"
)

// Task represents a unit of work for an agent
type Task struct {
	ID            string                 `json:"id"`
	EngagementID  string                 `json:"engagement_id"`
	AgentType     AgentType              `json:"agent_type"`
	Priority      int                    `json:"priority"` // 0-100, higher = more urgent
	Status        AgentStatus            `json:"status"`
	Params        map[string]interface{} `json:"params"`
	CreatedAt     int64                  `json:"created_at"`
	StartedAt     int64                  `json:"started_at,omitempty"`
	CompletedAt   int64                  `json:"completed_at,omitempty"`
	Result        interface{}            `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Retries       int                    `json:"retries"`
	MaxRetries    int                    `json:"max_retries"`
	Dependencies  []string               `json:"dependencies"` // Task IDs this depends on
	TimeoutSeconds int                    `json:"timeout_seconds"`
}

// Agent interface defines behavior for all agent types
type Agent interface {
	// GetType returns the agent type
	GetType() AgentType

	// GetStatus returns current agent status
	GetStatus() AgentStatus

	// Execute runs the agent on a task
	Execute(ctx context.Context, task *Task) (interface{}, error)

	// Validate checks if agent can execute the task
	Validate(task *Task) error

	// Stop gracefully stops the agent
	Stop() error

	// GetMetrics returns performance metrics
	GetMetrics() *Metrics
}

// Metrics tracks agent performance
type Metrics struct {
	TasksCompleted   int           `json:"tasks_completed"`
	TasksFailed      int           `json:"tasks_failed"`
	TotalExecutionMS int64         `json:"total_execution_ms"`
	AverageExecutionMS int64       `json:"average_execution_ms"`
	LastExecutionMS  int64         `json:"last_execution_ms"`
	SuccessRate      float64       `json:"success_rate"`
	LastUpdated      int64         `json:"last_updated"`
}

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	AgentType AgentType
	Status    AgentStatus
	Metrics   *Metrics
}

// GetType returns the agent type
func (b *BaseAgent) GetType() AgentType {
	return b.AgentType
}

// GetStatus returns current agent status
func (b *BaseAgent) GetStatus() AgentStatus {
	return b.Status
}

// GetMetrics returns performance metrics
func (b *BaseAgent) GetMetrics() *Metrics {
	return b.Metrics
}

// ReconnaissanceAgent performs reconnaissance tasks
type ReconnaissanceAgent struct {
	BaseAgent
	// Add recon-specific fields
}

// Execute runs reconnaissance
func (r *ReconnaissanceAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	// TODO: Implement reconnaissance logic
	return nil, nil
}

// Validate checks if task is valid for recon
func (r *ReconnaissanceAgent) Validate(task *Task) error {
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop stops the reconnaissance agent
func (r *ReconnaissanceAgent) Stop() error {
	r.Status = StatusTerminated
	return nil
}

// ScannerAgent performs vulnerability scanning
type ScannerAgent struct {
	BaseAgent
	// Add scanner-specific fields
}

// Execute runs scanning
func (s *ScannerAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	// TODO: Implement scanner logic
	return nil, nil
}

// Validate checks if task is valid for scanning
func (s *ScannerAgent) Validate(task *Task) error {
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop stops the scanner agent
func (s *ScannerAgent) Stop() error {
	s.Status = StatusTerminated
	return nil
}

// AnalyzerAgent analyzes scan results
type AnalyzerAgent struct {
	BaseAgent
	// Add analyzer-specific fields
}

// Execute runs analysis
func (a *AnalyzerAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	// TODO: Implement analysis logic
	return nil, nil
}

// Validate checks if task is valid for analysis
func (a *AnalyzerAgent) Validate(task *Task) error {
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop stops the analyzer agent
func (a *AnalyzerAgent) Stop() error {
	a.Status = StatusTerminated
	return nil
}

// ReporterAgent generates reports
type ReporterAgent struct {
	BaseAgent
	// Add reporter-specific fields
}

// Execute runs report generation
func (r *ReporterAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	// TODO: Implement reporting logic
	return nil, nil
}

// Validate checks if task is valid for reporting
func (r *ReporterAgent) Validate(task *Task) error {
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop stops the reporter agent
func (r *ReporterAgent) Stop() error {
	r.Status = StatusTerminated
	return nil
}

// OrchestratorAgent coordinates other agents
type OrchestratorAgent struct {
	BaseAgent
	agents map[AgentType]Agent
	// Add orchestrator-specific fields
}

// Execute coordinates agent execution
func (o *OrchestratorAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	// TODO: Implement orchestration logic
	return nil, nil
}

// Validate checks if task is valid for orchestration
func (o *OrchestratorAgent) Validate(task *Task) error {
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop stops the orchestrator agent
func (o *OrchestratorAgent) Stop() error {
	o.Status = StatusTerminated
	return nil
}
