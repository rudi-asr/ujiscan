package playbook

import (
	"fmt"
	"os"
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
	logFile, _ := os.OpenFile("/tmp/ujiscan_playbook.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()
	
	log := func(msg string) {
		fmt.Fprintf(logFile, "%s\n", msg)
		fmt.Fprint(logFile, msg+"\n")
	}
	
	// Load playbook
	log(fmt.Sprintf("[playbook] Loading playbook: %s for scan %s", playbookName, scanID))
	log(fmt.Sprintf("[playbook] CWD: %s", os.Getenv("PWD")))
	wd, _ := os.Getwd()
	log(fmt.Sprintf("[playbook] os.Getwd(): %s", wd))
	pb, err := e.loader.LoadPlaybook(playbookName)
	if err != nil {
		fmt.Printf("[playbook] Failed to load playbook: %v\n", err)
		return fmt.Errorf("failed to load playbook: %w", err)
	}
	fmt.Printf("[playbook] Playbook loaded: %s (entry_phase=%s)\n", pb.Name, pb.EntryPhase)

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
	// (Note: ExecuteTool already stores results, so context results are duplicate)
	// Just mark as completed

	// Mark as completed
	e.scanStore.CompleteScan(scanID, nil)
	return nil
}

// executePhase executes all steps in a phase
func (e *Engine) executePhase(pb *Playbook, ctx *ExecutionContext, phase PhaseType) error {
	ctx.CurrentPhase = phase
	steps := pb.GetPhaseSteps(phase)
	
	fmt.Printf("[%s] Executing phase %s with %d steps\n", ctx.PlaybookName, phase, len(steps))

	for _, step := range steps {
		// Check condition - empty condition means "always execute"
		if step.Condition != "" && step.Condition != "\"\"" {
			if !e.evaluateCondition(ctx, step.Condition) {
				fmt.Printf("  [SKIP] Step %s: condition not met (%s)\n", step.ID, step.Condition)
				continue
			}
		}

		fmt.Printf("  [EXEC] Step %s: tool=%s\n", step.ID, step.Tool)

		// Execute step
		output, err := e.executeStep(ctx, &step)
		if err != nil {
			fmt.Printf("  [ERROR] Step %s: %v\n", step.ID, err)
			// Don't fail entire phase on single step error
			continue
		}

		// Store result
		if output != nil {
			fmt.Printf("  [RESULT] Step %s: success=%v, exit_code=%d\n", step.ID, output.Success, output.ExitCode)
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

	fmt.Printf("    ExecuteTool: tool=%s, target=%s, args=%v\n", step.Tool, ctx.Target, args)

	// Execute tool
	output, err := e.scanExec.ExecuteTool(ctx.ScanID, step.Tool, args, models.PhaseEnum, ctx.Target)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	fmt.Printf("    Tool completed: success=%v, stdout_len=%d\n", output.Success, len(output.Stdout))
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
