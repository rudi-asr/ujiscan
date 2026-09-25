package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rudi-asr/ujiscan/internal/planner"
)

// Orchestrator manages the execution of engagement plans
type Orchestrator struct {
	scheduler *Scheduler
	collector *ResultCollector
	execution *EngagementExecution
	config    SchedulerConfig
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(config SchedulerConfig) *Orchestrator {
	return &Orchestrator{
		scheduler: NewScheduler(config),
		collector: NewResultCollector(),
		config:    config,
	}
}

// InitializeExecution prepares for plan execution
func (o *Orchestrator) InitializeExecution(plan *planner.ExecutionPlan) *EngagementExecution {
	o.execution = &EngagementExecution{
		ID:        uuid.New().String(),
		Target:    plan.Target,
		Status:    StatusPending,
		PlanID:    plan.ID,
		Phases:    make([]*PhaseExecution, 0),
		StartTime: time.Now(),
		Progress:  0,
	}

	// Create phase executions
	phaseMap := map[planner.ExecutionPhase]bool{
		planner.PhaseRecon:     true,
		planner.PhaseScanning:  true,
		planner.PhaseAnalysis:  true,
		planner.PhaseReporting: true,
	}

	for phaseKey := range phaseMap {
		if steps, exists := plan.Phases[phaseKey]; exists {
			phaseExec := o.scheduler.SchedulePhase(string(phaseKey), steps)
			o.execution.Phases = append(o.execution.Phases, phaseExec)
		}
	}

	return o.execution
}

// ExecutePlan executes the entire plan
func (o *Orchestrator) ExecutePlan(ctx context.Context, plan *planner.ExecutionPlan) (*EngagementExecution, error) {
	if o.execution == nil {
		o.InitializeExecution(plan)
	}

	o.execution.Status = StatusRunning

	// Execute phases sequentially
	for _, phase := range o.execution.Phases {
		select {
		case <-ctx.Done():
			o.execution.Status = StatusCanceled
			return o.execution, ctx.Err()
		default:
		}

		// Execute all steps in phase
		if err := o.executePhase(ctx, phase); err != nil {
			o.execution.Status = StatusFailed
			return o.execution, err
		}

		// Update progress
		o.execution.Progress = o.scheduler.CalculateProgress(o.execution.Phases)
	}

	// All phases completed
	o.execution.Status = StatusCompleted
	now := time.Now()
	o.execution.EndTime = &now

	// Aggregate findings
	o.execution.Findings = o.collector.RankFindings()

	return o.execution, nil
}

// executePhase executes all steps in a phase
func (o *Orchestrator) executePhase(ctx context.Context, phase *PhaseExecution) error {
	phase.Status = StatusRunning

	for _, step := range phase.Steps {
		select {
		case <-ctx.Done():
			phase.Status = StatusCanceled
			return ctx.Err()
		default:
		}

		// Execute step with retry logic
		if err := o.executeStepWithRetry(ctx, step); err != nil {
			step.Status = StatusFailed
			step.Error = err.Error()
			// Continue with other steps (don't fail entire phase)
		}

		o.execution.CurrentStep = step.Name
		o.execution.Progress = o.scheduler.CalculateProgress(o.execution.Phases)
	}

	// Mark phase complete if all steps done
	if o.scheduler.CheckPhaseCompletion(phase) {
		phase.Status = StatusCompleted
	}

	return nil
}

// executeStepWithRetry executes a step with retry logic
func (o *Orchestrator) executeStepWithRetry(ctx context.Context, step *StepExecution) error {
	for attempt := 0; attempt <= o.config.MaxRetries; attempt++ {
		err := o.scheduler.ExecuteStep(ctx, step)
		if err == nil && step.Status == StatusCompleted {
			return nil // Success
		}

		if err != nil && step.RetryCount < o.config.MaxRetries {
			// Retry
			if err := o.scheduler.RetryStep(ctx, step); err != nil {
				return err
			}
		} else if step.Status == StatusFailed {
			return fmt.Errorf("step %s failed after %d retries", step.Name, step.RetryCount)
		}
	}

	return fmt.Errorf("step %s exceeded max retries", step.Name)
}

// GetExecutionStatus returns current execution status
func (o *Orchestrator) GetExecutionStatus() *EngagementExecution {
	return o.execution
}

// GetFindings returns collected findings
func (o *Orchestrator) GetFindings() []Finding {
	return o.collector.RankFindings()
}

// AddFinding adds a finding (from agent)
func (o *Orchestrator) AddFinding(finding Finding) {
	o.collector.AddFinding(finding)
}

// DeduplicateFindings removes duplicates
func (o *Orchestrator) DeduplicateFindings() []Finding {
	return o.collector.DeduplicateFindings()
}

// CancelExecution cancels ongoing execution
func (o *Orchestrator) CancelExecution() {
	if o.execution != nil {
		o.execution.Status = StatusCanceled
		now := time.Now()
		o.execution.EndTime = &now
	}
}

// GetSummary returns execution summary
func (o *Orchestrator) GetSummary() map[string]interface{} {
	if o.execution == nil {
		return nil
	}

	return map[string]interface{}{
		"id":        o.execution.ID,
		"target":    o.execution.Target,
		"status":    o.execution.Status,
		"progress":  o.execution.Progress,
		"start_time": o.execution.StartTime,
		"end_time":   o.execution.EndTime,
		"findings":   o.collector.GetFindingsSummary(),
	}
}
