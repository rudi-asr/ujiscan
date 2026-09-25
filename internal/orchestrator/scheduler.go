package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rudi-asr/ujiscan/internal/planner"
)

// Scheduler manages task execution scheduling
type Scheduler struct {
	config SchedulerConfig
	mu     sync.Mutex
}

// NewScheduler creates a new scheduler
func NewScheduler(config SchedulerConfig) *Scheduler {
	if config.MaxParallel <= 0 {
		config.MaxParallel = 3
	}
	if config.TaskTimeout <= 0 {
		config.TaskTimeout = 300 // 5 minutes default
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 2
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 5
	}

	return &Scheduler{
		config: config,
	}
}

// ScheduleStep schedules a single step for execution
func (s *Scheduler) ScheduleStep(step *planner.ExecutionStep) *StepExecution {
	now := time.Now()
	return &StepExecution{
		ID:        step.ID,
		Name:      step.Name,
		Status:    StatusPending,
		StartTime: &now,
		Duration:  step.Timeout,
	}
}

// SchedulePhase creates phase execution structure
func (s *Scheduler) SchedulePhase(phase string, steps []planner.ExecutionStep) *PhaseExecution {
	now := time.Now()
	phaseExec := &PhaseExecution{
		Phase:     phase,
		Status:    StatusPending,
		Steps:     make([]*StepExecution, 0),
		StartTime: &now,
	}

	for i := range steps {
		stepExec := s.ScheduleStep(&steps[i])
		phaseExec.Steps = append(phaseExec.Steps, stepExec)
	}

	return phaseExec
}

// ExecuteStep executes a single step (simulated)
func (s *Scheduler) ExecuteStep(ctx context.Context, stepExec *StepExecution) error {
	s.mu.Lock()
	stepExec.Status = StatusRunning
	s.mu.Unlock()

	// Simulate tool execution
	select {
	case <-ctx.Done():
		stepExec.Status = StatusCanceled
		return ctx.Err()
	case <-time.After(time.Duration(stepExec.Duration) * time.Millisecond):
		// Simulate successful execution
		s.mu.Lock()
		stepExec.Status = StatusCompleted
		now := time.Now()
		stepExec.EndTime = &now
		s.mu.Unlock()
		return nil
	}
}

// RetryStep retries a failed step
func (s *Scheduler) RetryStep(ctx context.Context, stepExec *StepExecution) error {
	if stepExec.RetryCount >= s.config.MaxRetries {
		return fmt.Errorf("max retries exceeded for step %s", stepExec.Name)
	}

	stepExec.RetryCount++
	s.mu.Lock()
	stepExec.Status = StatusRetrying
	s.mu.Unlock()

	// Wait before retry
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(s.config.RetryDelay) * time.Second):
	}

	// Execute again
	return s.ExecuteStep(ctx, stepExec)
}

// CalculateProgress calculates progress percentage
func (s *Scheduler) CalculateProgress(phases []*PhaseExecution) int {
	if len(phases) == 0 {
		return 0
	}

	total := 0
	completed := 0

	for _, phase := range phases {
		for _, step := range phase.Steps {
			total++
			if step.Status == StatusCompleted {
				completed++
			}
		}
	}

	if total == 0 {
		return 0
	}

	return (completed * 100) / total
}

// CheckPhaseCompletion checks if all steps in a phase completed
func (s *Scheduler) CheckPhaseCompletion(phase *PhaseExecution) bool {
	for _, step := range phase.Steps {
		if step.Status != StatusCompleted && step.Status != StatusFailed {
			return false
		}
	}
	return true
}
