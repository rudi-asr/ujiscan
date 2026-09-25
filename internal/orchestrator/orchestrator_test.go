package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/rudi-asr/ujiscan/internal/planner"
)

func TestSchedulerCreation(t *testing.T) {
	config := SchedulerConfig{
		MaxParallel: 3,
		TaskTimeout: 100,
		MaxRetries:  2,
		RetryDelay:  5,
	}
	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Errorf("Expected scheduler to be created")
	}
}

func TestScheduleStep(t *testing.T) {
	scheduler := NewScheduler(SchedulerConfig{})
	step := &planner.ExecutionStep{
		ID:      "test-step-1",
		Name:    "Test Step",
		Timeout: 100,
	}
	execStep := scheduler.ScheduleStep(step)
	if execStep.Status != StatusPending {
		t.Errorf("Expected pending status, got %v", execStep.Status)
	}
}

func TestResultCollectorAddFinding(t *testing.T) {
	collector := NewResultCollector()
	finding := Finding{
		Source:      "nmap",
		Type:        "OpenPort",
		Description: "Port 22 open",
		Severity:    5,
	}
	collector.AddFinding(finding)

	findings := collector.GetFindings()
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
}

func TestResultCollectorRankFindings(t *testing.T) {
	collector := NewResultCollector()
	collector.AddFinding(Finding{Type: "Low", Severity: 1})
	collector.AddFinding(Finding{Type: "High", Severity: 9})
	collector.AddFinding(Finding{Type: "Medium", Severity: 5})

	ranked := collector.RankFindings()
	if ranked[0].Severity != 9 {
		t.Errorf("Expected first finding to have severity 9")
	}
}

func TestOrchestratorInitializeExecution(t *testing.T) {
	plan := &planner.ExecutionPlan{
		ID:     "plan-123",
		Target: "example.com",
		Mode:   planner.ModeWebStandard,
		Phases: map[planner.ExecutionPhase][]planner.ExecutionStep{
			planner.PhaseRecon: {
				{ID: "step-1", Name: "Test"},
			},
		},
	}

	orchestrator := NewOrchestrator(SchedulerConfig{})
	exec := orchestrator.InitializeExecution(plan)

	if exec == nil {
		t.Errorf("Expected execution to be initialized")
	}
	if exec.Status != StatusPending {
		t.Errorf("Expected pending status")
	}
}

func TestOrchestratorExecutePlan(t *testing.T) {
	plan := &planner.ExecutionPlan{
		ID:     "plan-123",
		Target: "example.com",
		Mode:   planner.ModeWebStandard,
		Phases: map[planner.ExecutionPhase][]planner.ExecutionStep{
			planner.PhaseRecon: {
				{
					ID:         "step-1",
					Name:       "Discovery",
					Agent:      "recon",
					Tool:       "nmap",
					Timeout:    50,
					MaxRetries: 1,
				},
			},
		},
	}

	orchestrator := NewOrchestrator(SchedulerConfig{
		MaxParallel: 1,
		TaskTimeout: 100,
		MaxRetries:  1,
		RetryDelay:  5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exec, err := orchestrator.ExecutePlan(ctx, plan)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if exec == nil {
		t.Errorf("Expected execution result")
	}

	if exec.Status != StatusCompleted {
		t.Errorf("Expected completed status, got %v", exec.Status)
	}
}

func TestOrchestratorCancelExecution(t *testing.T) {
	plan := &planner.ExecutionPlan{
		ID:     "plan-123",
		Target: "example.com",
		Phases: map[planner.ExecutionPhase][]planner.ExecutionStep{},
	}

	orchestrator := NewOrchestrator(SchedulerConfig{})
	orchestrator.InitializeExecution(plan)
	orchestrator.CancelExecution()

	if orchestrator.execution.Status != StatusCanceled {
		t.Errorf("Expected canceled status")
	}
}
