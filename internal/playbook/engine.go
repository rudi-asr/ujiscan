package playbook

import (
	"fmt"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/executor"
	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/rudi-asr/ujiscan/internal/store"
)

// Engine executes playbooks
type Engine struct {
	loader      *PlaybookLoader
	scanExec    *executor.ScanExecutor
	scanStore   *store.ScanStore
}

// NewEngine creates a new playbook engine
func NewEngine(loader *PlaybookLoader, scanExec *executor.ScanExecutor, scanStore *store.ScanStore) *Engine {
	return &Engine{
		loader:    loader,
		scanExec:  scanExec,
		scanStore: scanStore,
	}
}

// ExecutePlaybook executes a playbook for a target
func (e *Engine) ExecutePlaybook(scanID string, playbookName string, target string) error {
	// Load playbook
	pb, err := e.loader.LoadPlaybook(playbookName)
	if err != nil {
		return fmt.Errorf("failed to load playbook: %w", err)
	}

	// Create execution context
	ctx := NewExecutionContext(playbookName, target, scanID)

	// Update scan status
	e.scanStore.UpdateScanStatus(scanID, models.ScanStatusRunning)

	// Execute entry phase
	err = e.executePhase(pb, ctx, pb.EntryPhase)
	if err != nil {
		e.scanStore.CompleteScan(scanID, err)
		return fmt.Errorf("playbook execution failed: %w", err)
	}

	// Continue with subsequent phases if recon is entry
	if pb.EntryPhase == PhaseRecon {
		e.executePhase(pb, ctx, PhaseEnum)
		e.executePhase(pb, ctx, PhaseExploit)
	}

	// Store all results in scan
	for _, result := range ctx.Results {
		e.scanStore.AddResult(scanID, result)
	}

	// Mark as completed
	e.scanStore.CompleteScan(scanID, nil)
	return nil
}

// executePhase executes all steps in a phase
func (e *Engine) executePhase(pb *Playbook, ctx *ExecutionContext, phase PhaseType) error {
	ctx.CurrentPhase = phase
	steps := pb.GetPhaseSteps(phase)

	for _, step := range steps {
		// Check condition
		if step.Condition != "" {
			if !e.evaluateCondition(ctx, step.Condition) {
				fmt.Printf("Skipping step %s: condition not met (%s)\n", step.ID, step.Condition)
				continue
			}
		}

		// Execute step
		output, err := e.executeStep(ctx, &step)
		if err != nil {
			fmt.Printf("Error executing step %s: %v\n", step.ID, err)
			// Don't fail entire phase on single step error
			continue
		}

		// Store result
		if output != nil {
			ctx.AddStepResult(step.ID, output)

			// Update conditions based on result
			e.updateConditions(ctx, &step, output)
		}

		// Check next steps branching
		if len(step.NextSteps) > 0 {
			for condition, nextStepID := range step.NextSteps {
				if e.evaluateCondition(ctx, condition) {
					nextStep := pb.GetStep(nextStepID)
					if nextStep != nil {
						_, err := e.executeStep(ctx, nextStep)
						if err != nil {
							fmt.Printf("Error executing conditional step %s: %v\n", nextStepID, err)
						}
					}
				}
			}
		}
	}

	return nil
}

// executeStep executes a single step
func (e *Engine) executeStep(ctx *ExecutionContext, step *Step) (*models.ToolOutput, error) {
	// Substitute {target} in args
	args := make([]string, len(step.Args))
	for i, arg := range step.Args {
		args[i] = strings.ReplaceAll(arg, "{target}", ctx.Target)
	}

	// Execute tool
	output, err := e.scanExec.ExecuteTool(ctx.ScanID, step.Tool, args, models.PhaseEnum, ctx.Target)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	fmt.Printf("[%s] Executed step %s with tool %s\n", ctx.PlaybookName, step.ID, step.Tool)
	return output, nil
}

// evaluateCondition evaluates a condition string
func (e *Engine) evaluateCondition(ctx *ExecutionContext, condition string) bool {
	// Simple condition evaluation
	// Format: "if_port_open_22", "if_service_http", etc

	if condition == "" {
		return true
	}

	// Check if condition exists in context
	_, exists := ctx.GetCondition(condition)
	return exists
}

// updateConditions updates execution context conditions based on tool output
func (e *Engine) updateConditions(ctx *ExecutionContext, step *Step, output *models.ToolOutput) {
	// Parse tool output to extract conditions
	switch step.Tool {
	case "nmap":
		e.updateNmapConditions(ctx, output)
	case "nuclei":
		e.updateNucleiConditions(ctx, output)
	}
}

// updateNmapConditions extracts port information from nmap output
func (e *Engine) updateNmapConditions(ctx *ExecutionContext, output *models.ToolOutput) {
	if !output.Success || output.Stdout == "" {
		return
	}

	// Simple parsing: look for "open" in output
	if strings.Contains(strings.ToLower(output.Stdout), "open") {
		ctx.SetCondition("ports_open", true)

		// Check for common services
		if strings.Contains(output.Stdout, "22") && strings.Contains(output.Stdout, "ssh") {
			ctx.SetCondition("if_port_open_22", true)
		}
		if strings.Contains(output.Stdout, "80") || strings.Contains(output.Stdout, "http") {
			ctx.SetCondition("if_port_open_80", true)
		}
		if strings.Contains(output.Stdout, "443") || strings.Contains(output.Stdout, "https") {
			ctx.SetCondition("if_port_open_443", true)
		}
	}
}

// updateNucleiConditions extracts vulnerability information from nuclei output
func (e *Engine) updateNucleiConditions(ctx *ExecutionContext, output *models.ToolOutput) {
	if !output.Success || output.Stdout == "" {
		return
	}

	// Simple parsing: check if findings exist
	if strings.Contains(output.Stdout, "severity") {
		ctx.SetCondition("vulnerabilities_found", true)

		if strings.Contains(output.Stdout, "critical") {
			ctx.SetCondition("critical_vulnerability", true)
		}
		if strings.Contains(output.Stdout, "high") {
			ctx.SetCondition("high_vulnerability", true)
		}
	}
}

// ListPlaybooks returns all available playbooks
func (e *Engine) ListPlaybooks() ([]*Playbook, error) {
	return e.loader.LoadAllPlaybooks()
}

// GetPlaybook retrieves a playbook by name
func (e *Engine) GetPlaybook(name string) (*Playbook, error) {
	return e.loader.LoadPlaybook(name)
}
