package playbook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// AgenticDecision represents AI's decision on next action
type AgenticDecision struct {
	Analysis      string `json:"analysis"`
	Recommendation string `json:"recommendation"`
	Reasoning      string `json:"reasoning"`
	NextPhase      string `json:"next_phase,omitempty"`
}

// ExecuteAgenticPlaybook executes playbook with AI decision-making
func (e *Engine) ExecuteAgenticPlaybook(scanID string, playbookName string, target string, objective string) error {
	// Load playbook
	fmt.Printf("[agentic] Loading playbook: %s for scan %s\n", playbookName, scanID)
	pb, err := e.loader.LoadPlaybook(playbookName)
	if err != nil {
		fmt.Printf("[agentic] Failed to load playbook: %v\n", err)
		e.scanStore.CompleteScan(scanID, err)
		return fmt.Errorf("failed to load playbook: %w", err)
	}

	// Create execution context with AI enabled
	ctx := &ExecutionContext{
		PlaybookName: pb.Name,
		Target:       target,
		ScanID:       scanID,
		Results:      []models.ToolOutput{},
		Conditions:   make(map[string]interface{}),
		StepResults:  make(map[string]*models.ToolOutput),
		IsAgentic:    true,
		AIObjective:  objective,
		AIDecisions:  make(map[string]string),
		AIReasonings: make(map[string]string),
		CurrentPhase: pb.EntryPhase,
	}

	fmt.Printf("[agentic] Objective: %s\n", objective)
	fmt.Printf("[agentic] Entry phase: %s\n", pb.EntryPhase)

	// Initialize Claude client
	aiClient := ai.NewClaudeClient("")

	// Execute phases with AI guidance
	currentPhase := pb.EntryPhase
	maxIterations := 10 // prevent infinite loops
	iteration := 0

	for currentPhase != "" && iteration < maxIterations {
		iteration++
		fmt.Printf("[agentic] Iteration %d: Executing phase %s\n", iteration, currentPhase)

		// Get steps for current phase
		steps, ok := pb.Phases[currentPhase]
		if !ok || len(steps) == 0 {
			fmt.Printf("[agentic] No steps in phase %s, moving to next\n", currentPhase)
			// Move to next phase
			currentPhase = getNextPhase(currentPhase)
			continue
		}

		// Execute steps in current phase
		shouldContinuePhase := true
		for _, step := range steps {
			// Check condition
			if step.Condition != "" && step.Condition != `""` {
				if !e.evaluateCondition(ctx, step.Condition) {
					fmt.Printf("[agentic] [SKIP] %s: condition not met (%s)\n", step.ID, step.Condition)
					continue
				}
			}

			fmt.Printf("[agentic] [EXEC] Executing step: %s (tool=%s)\n", step.ID, step.Tool)

			// Execute tool via executor
			output, err := e.scanExec.ExecuteTool(scanID, step.Tool, step.Args, models.PhaseEnum, target)
			if err != nil {
				fmt.Printf("[agentic] Tool execution failed: %v\n", err)
				// Create dummy output with error
				output = &models.ToolOutput{
					ToolName:  step.Tool,
					Target:    target,
					Success:   false,
					ExitCode:  1,
					Stdout:    "",
					Stderr:    err.Error(),
				}
			}

			ctx.Results = append(ctx.Results, *output)
			ctx.StepResults[step.ID] = output
			ctx.Conditions[fmt.Sprintf("%s_success", step.ID)] = output.Success
			ctx.Conditions["ports_open"] = output.Success && strings.Contains(strings.ToLower(step.Tool), "nmap")

			fmt.Printf("[agentic] Step %s completed: success=%v\n", step.ID, output.Success)

			// AI analyzes output
			fmt.Printf("[agentic] Asking AI to analyze output...\n")
			decision, err := aiClient.AnalyzeToolOutput(step.Tool, output.Stdout, objective)
			if err != nil {
				fmt.Printf("[agentic] AI analysis failed: %v, continuing anyway\n", err)
				ctx.AIDecisions[step.ID] = "continue"
				ctx.AIReasonings[step.ID] = "AI error: " + err.Error()
			} else {
				// Parse AI decision
				var aiDecision AgenticDecision
				err = json.Unmarshal([]byte(decision), &aiDecision)
				if err != nil {
					fmt.Printf("[agentic] Failed to parse AI response: %v\n", err)
					ctx.AIDecisions[step.ID] = "continue"
					ctx.AIReasonings[step.ID] = decision
				} else {
					ctx.AIDecisions[step.ID] = aiDecision.Recommendation
					ctx.AIReasonings[step.ID] = fmt.Sprintf("%s - %s", aiDecision.Analysis, aiDecision.Reasoning)
					fmt.Printf("[agentic] AI Decision: %s (%s)\n", aiDecision.Recommendation, aiDecision.Analysis)

					// Handle special AI recommendations
					if aiDecision.Recommendation == "stop" {
						fmt.Printf("[agentic] AI recommends stopping\n")
						shouldContinuePhase = false
						break
					}
					if aiDecision.Recommendation == "exploit" && currentPhase != PhaseExploit {
						fmt.Printf("[agentic] AI recommends exploit phase\n")
						currentPhase = PhaseExploit
						shouldContinuePhase = false
						break
					}
				}
			}
		}

		if !shouldContinuePhase && currentPhase != PhaseExploit {
			// AI stopped execution
			break
		}

		// After phase completion, ask AI for next phase recommendation
		fmt.Printf("[agentic] Phase %s complete. Asking AI for next phase...\n", currentPhase)
		scanContext := buildScanContext(ctx)
		nextPhaseRec, err := aiClient.PlanNextStep(scanContext, objective)
		if err != nil {
			fmt.Printf("[agentic] AI planning failed: %v, using default chain\n", err)
			currentPhase = getNextPhase(currentPhase)
		} else {
			var planDecision struct {
				NextPhase string `json:"next_phase"`
				Reasoning string `json:"reasoning"`
			}
			err = json.Unmarshal([]byte(nextPhaseRec), &planDecision)
			if err != nil {
				fmt.Printf("[agentic] Failed to parse phase plan: %v\n", err)
				currentPhase = getNextPhase(currentPhase)
			} else {
				fmt.Printf("[agentic] AI recommends next phase: %s (%s)\n", planDecision.NextPhase, planDecision.Reasoning)
				if planDecision.NextPhase != "" {
					currentPhase = PhaseType(planDecision.NextPhase)
				} else {
					currentPhase = getNextPhase(currentPhase)
				}
			}
		}
	}
	fmt.Printf("[agentic] Playbook execution complete. Total iterations: %d, Results: %d\n", iteration, len(ctx.Results))
	e.scanStore.CompleteScan(scanID, nil)
	return nil
}

// getNextPhase returns the next phase in sequence
func getNextPhase(current PhaseType) PhaseType {
	phases := []PhaseType{PhaseRecon, PhaseEnum, PhaseExploit, PhaseVerify, PhaseReport}
	for i, p := range phases {
		if p == current && i < len(phases)-1 {
			return phases[i+1]
		}
	}
	return ""
}

// buildScanContext creates context summary for AI reasoning
func buildScanContext(ctx *ExecutionContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Target: %s\n", ctx.Target))
	sb.WriteString(fmt.Sprintf("Current Phase: %s\n", ctx.CurrentPhase))
	sb.WriteString(fmt.Sprintf("Steps Executed: %d\n", len(ctx.Results)))
	sb.WriteString("\nKey Findings:\n")

	for id, result := range ctx.StepResults {
		if result != nil {
			sb.WriteString(fmt.Sprintf("- %s (%s): success=%v, output_len=%d\n", id, result.ToolName, result.Success, len(result.Stdout)))
		}
	}

	return sb.String()
}
