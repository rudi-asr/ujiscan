package planner

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rudi-asr/ujiscan/internal/validator"
)

// PlanGenerator generates execution plans from target profiles
type PlanGenerator struct {
	modeSelector *ModeSelector
}

// NewPlanGenerator creates a new plan generator
func NewPlanGenerator() *PlanGenerator {
	return &PlanGenerator{
		modeSelector: NewModeSelector(),
	}
}

// GeneratePlan generates an execution plan for a target
func (pg *PlanGenerator) GeneratePlan(profile *validator.TargetProfile) (*ExecutionPlan, error) {
	if profile == nil {
		return nil, fmt.Errorf("target profile cannot be nil")
	}

	if !profile.IsValid {
		return nil, fmt.Errorf("target profile is not valid: %s", profile.Reason)
	}

	// Detect mode from target type
	modeResult := pg.modeSelector.DetectMode(profile.Type)
	modeConfig := pg.modeSelector.SelectMode(modeResult.Mode)

	// Create base plan
	plan := &ExecutionPlan{
		ID:        uuid.New().String(),
		Target:    profile.Target,
		Mode:      modeResult.Mode,
		Phases:    make(map[ExecutionPhase][]ExecutionStep),
		Tools:     modeConfig.Tools,
		CreatedAt: time.Now(),
		Version:   "1.0",
	}

	// Build execution steps for each phase
	pg.buildPhases(plan, profile, modeConfig)

	// Calculate estimated duration
	plan.EstimatedDuration = modeConfig.Duration

	return plan, nil
}

// buildPhases creates execution steps for each phase
func (pg *PlanGenerator) buildPhases(plan *ExecutionPlan, profile *validator.TargetProfile, modeConfig *ModeConfig) {
	targetHost := ""
	if len(profile.Hosts) > 0 {
		targetHost = profile.Hosts[0]
	}

	for _, phase := range modeConfig.Phases {
		switch phase {
		case PhaseRecon:
			plan.Phases[PhaseRecon] = pg.buildReconPhase(targetHost, modeConfig)
		case PhaseScanning:
			plan.Phases[PhaseScanning] = pg.buildScanningPhase(targetHost, modeConfig)
		case PhaseAnalysis:
			plan.Phases[PhaseAnalysis] = pg.buildAnalysisPhase(targetHost, modeConfig)
		case PhaseReporting:
			plan.Phases[PhaseReporting] = pg.buildReportingPhase(targetHost, modeConfig)
		}
	}
}

// buildReconPhase creates reconnaissance steps
func (pg *PlanGenerator) buildReconPhase(target string, modeConfig *ModeConfig) []ExecutionStep {
	steps := []ExecutionStep{
		{
			ID:         uuid.New().String(),
			Name:       "Network Discovery",
			Agent:      "recon",
			Tool:       "nmap",
			Parameters: map[string]interface{}{
				"target":  target,
				"args":    "-sV -sC --open",
				"timeout": 300,
			},
			Timeout:    300,
			MaxRetries: 2,
			Parallel:   false,
		},
	}

	// Add additional recon steps based on mode
	if modeConfig.Name == ModeWebStandard {
		steps = append(steps, ExecutionStep{
			ID:    uuid.New().String(),
			Name:  "Subdomain Enumeration",
			Agent: "recon",
			Tool:  "subfinder",
			Parameters: map[string]interface{}{
				"target": target,
			},
			Timeout:    120,
			MaxRetries: 1,
			Parallel:   false,
		})
	}

	return steps
}

// buildScanningPhase creates scanning steps
func (pg *PlanGenerator) buildScanningPhase(target string, modeConfig *ModeConfig) []ExecutionStep {
	steps := []ExecutionStep{
		{
			ID:    uuid.New().String(),
			Name:  "Vulnerability Scanning",
			Agent: "scanner",
			Tool:  "nuclei",
			Parameters: map[string]interface{}{
				"target": target,
				"args":   "-t cves -severity high,critical",
			},
			Timeout:    300,
			MaxRetries: 1,
			Parallel:   false,
		},
	}

	// Add aggressive scanning if needed
	if modeConfig.Name == ModeAggressiveTest {
		steps = append(steps, ExecutionStep{
			ID:    uuid.New().String(),
			Name:  "Directory Brute Force",
			Agent: "scanner",
			Tool:  "ffuf",
			Parameters: map[string]interface{}{
				"target": target,
				"args":   "-w /usr/share/wordlists/dirb/common.txt",
			},
			Timeout:    300,
			MaxRetries: 1,
			Parallel:   false,
		})
	}

	return steps
}

// buildAnalysisPhase creates analysis steps
func (pg *PlanGenerator) buildAnalysisPhase(target string, modeConfig *ModeConfig) []ExecutionStep {
	return []ExecutionStep{
		{
			ID:    uuid.New().String(),
			Name:  "AI Vulnerability Analysis",
			Agent: "analyzer",
			Tool:  "claude-analyzer",
			Parameters: map[string]interface{}{
				"target": target,
				"model":  "claude-3-sonnet",
			},
			Timeout:    180,
			MaxRetries: 1,
			Parallel:   false,
		},
	}
}

// buildReportingPhase creates reporting steps
func (pg *PlanGenerator) buildReportingPhase(target string, modeConfig *ModeConfig) []ExecutionStep {
	return []ExecutionStep{
		{
			ID:    uuid.New().String(),
			Name:  "Report Generation",
			Agent: "reporter",
			Tool:  "report-generator",
			Parameters: map[string]interface{}{
				"target":  target,
				"formats": []string{"html", "pdf", "json"},
			},
			Timeout:    120,
			MaxRetries: 0,
			Parallel:   false,
		},
	}
}

// ModifyPlan allows modification of generated plan
func (pg *PlanGenerator) ModifyPlan(plan *ExecutionPlan, mode EngagementMode) *ExecutionPlan {
	modeConfig := pg.modeSelector.SelectMode(mode)
	if modeConfig != nil {
		plan.Mode = mode
		plan.EstimatedDuration = modeConfig.Duration
		plan.Tools = modeConfig.Tools
		// Phases would need to be rebuilt
	}
	return plan
}

// ExportPlan exports plan as JSON (for API response)
func (pg *PlanGenerator) ExportPlan(plan *ExecutionPlan) map[string]interface{} {
	return map[string]interface{}{
		"id":                 plan.ID,
		"target":             plan.Target,
		"mode":               plan.Mode,
		"tools":              plan.Tools,
		"estimated_duration": plan.EstimatedDuration,
		"phases":             plan.Phases,
		"created_at":         plan.CreatedAt,
		"version":            plan.Version,
	}
}
