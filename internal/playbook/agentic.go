package playbook

import (
	"context"
	"fmt"
	"time"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// AgenticExecutor runs AI-driven playbook execution with dynamic tool selection
type AgenticExecutor struct {
	engine   *Engine
	aiClient *ai.Client
}

// NewAgenticExecutor creates new agentic executor
func NewAgenticExecutor(engine *Engine) *AgenticExecutor {
	return &AgenticExecutor{
		engine:   engine,
		aiClient: engine.aiClient,
	}
}

// ExecuteAgenticScan runs full agentic scanning workflow with the
// C.4 adaptive execution loop: phase-wise parallel execution, result
// aggregation, AI-driven continue/stop decisions, and learning feedback.
func (ae *AgenticExecutor) ExecuteAgenticScan(ctx context.Context, scanID string, target string, scanType string) error {
	fmt.Printf("[agentic] Starting agentic scan for %s (type=%s, scan_id=%s)\n", target, scanType, scanID)

	// Phase 1: AI recommends initial tools based on target
	fmt.Printf("[agentic] Phase 1: AI tool selection for %s\n", target)
	selectedTools, err := ae.aiClient.SelectToolsForTarget(ctx, target, scanType)
	if err != nil {
		fmt.Printf("[agentic] Tool selection failed: %v, using defaults\n", err)
		selectedTools = ai.DefaultToolsForType(scanType)
	}
	fmt.Printf("[agentic] AI recommended tools: %v\n", selectedTools)

	// Adaptive loop over phases (C.4): recon → enum → exploit
	phaseSequence := []string{"recon", "enum", "exploit"}
	lastDecision := &ai.AgentDecision{RecommendedTools: selectedTools}
	totalFindings := 0

	for _, phase := range phaseSequence {
		tools := ae.aiClient.GetPreferredTools(lastDecision.RecommendedTools)
		if len(tools) == 0 {
			tools = ai.DefaultToolsForType(phase)
		}

		// Findings before this phase (to compute the phase delta)
		findingsBefore := totalFindings
		if scan, err := ae.engine.scanStore.GetScan(scanID); err == nil {
			findingsBefore = len(scan.Findings)
		}

		// C.4.3: parallel execution with per-tool timeout
		fmt.Printf("[agentic] Phase %s: executing %d tools in parallel\n", phase, len(tools))
		phaseStart := time.Now()
		results := ae.ExecuteToolsPhaseParallel(ctx, scanID, phase, target, tools)
		phaseElapsed := time.Since(phaseStart)

		// Findings after this phase
		if scan, err := ae.engine.scanStore.GetScan(scanID); err == nil {
			totalFindings = len(scan.Findings)
		}

		// C.4.1: aggregate phase results
		phaseResults := ae.AggregatePhaseResults(phase, tools, results, phaseElapsed, totalFindings-findingsBefore)
		fmt.Printf("[agentic] Phase %s done: %d results, +%d findings, %.1fs\n",
			phase, phaseResults.TotalResults, phaseResults.FindingsCount, phaseElapsed.Seconds())

		// C.4.4: learning feedback from tool outcomes
		for _, outcome := range phaseResults.ToolOutcomes {
			ae.aiClient.UpdateToolFeedback(outcome.Tool, outcome.Success, outcome.Duration)
		}

		// AI analyzes phase results and recommends next steps
		decision, err := ae.analyzePhaseResults(ctx, results, phase, target)
		if err != nil {
			fmt.Printf("[agentic] Analysis failed: %v\n", err)
			decision = &ai.AgentDecision{
				RecommendedTools: ai.DefaultToolsForType(nextPhaseAfter(phase)),
				Confidence:       0.5,
				StopScan:         false,
			}
		}
		lastDecision = decision

		// C.4.2: decide whether to continue to the next phase
		if !ae.ShouldContinueToPhase(phaseResults, decision, totalFindings) {
			fmt.Printf("[agentic] Adaptive loop stopping after %s phase\n", phase)
			break
		}
		fmt.Printf("[agentic] Continuing to next phase (confidence=%.2f, tools=%v)\n",
			decision.Confidence, decision.RecommendedTools)
	}

	// Final phase: AI generates final report
	fmt.Printf("[agentic] Generating final AI report\n")
	finalReport := ae.generateFinalReport(ctx, scanID, target)
	fmt.Printf("[agentic] Final report:\n%s\n", finalReport)

	// Mark scan complete
	ae.engine.scanStore.UpdateScanStatus(scanID, models.ScanStatusCompleted)
	fmt.Printf("[agentic] ✅ Agentic scan completed for %s\n", scanID)

	return nil
}

// nextPhaseAfter returns the default tool-set name for the phase following `phase`.
func nextPhaseAfter(phase string) string {
	switch phase {
	case "recon":
		return "enum"
	case "enum":
		return "exploit"
	default:
		return "enum"
	}
}

// analyzePhaseResults asks AI what to do next based on phase results
func (ae *AgenticExecutor) analyzePhaseResults(ctx context.Context, results []models.ToolOutput, phase string, target string) (*ai.AgentDecision, error) {
	if len(results) == 0 {
		fmt.Printf("[agentic] No results from %s phase\n", phase)
		return &ai.AgentDecision{
			Analysis:         "No tools completed",
			RecommendedTools: []string{},
			Confidence:       0.0,
			StopScan:         false,
		}, nil
	}

	// For recon phase, try to detect services and recommend tools dynamically
	if phase == "recon" {
		return ae.analyzeReconAndDetectServices(ctx, results, target)
	}

	// For other phases, use standard analysis
	combinedOutput := ""
	for i, result := range results {
		out := result.Stdout
		if len(out) > 500 {
			out = out[:500] + "...[truncated]"
		}
		combinedOutput += fmt.Sprintf("=== Tool %d: %s (exit=%d) ===\n%s\n\n", i+1, result.ToolName, result.ExitCode, out)
	}

	var prompt string
	switch phase {
	case "enum":
		prompt = fmt.Sprintf("Enumeration phase on %s completed. Should we run exploitation tools? Which ones?", target)
	case "exploit":
		prompt = fmt.Sprintf("Exploitation phase on %s completed. All scanning done. Summarize findings.", target)
	default:
		prompt = "Analyze these tool results and recommend next steps."
	}

	fmt.Printf("[agentic] Sending analysis prompt to AI (phase=%s, tools=%d)\n", phase, len(results))
	decision, err := ae.engine.aiClient.AnalyzeToolOutput(ctx, "combined-tools", combinedOutput, prompt)
	if err != nil {
		fmt.Printf("[agentic] AI analysis failed: %v\n", err)
		return nil, err
	}

	fmt.Printf("[agentic] AI Decision: %s (confidence=%.2f, tools=%v)\n", decision.Analysis, decision.Confidence, decision.RecommendedTools)
	return decision, nil
}

// analyzeReconAndDetectServices performs service detection during recon phase
func (ae *AgenticExecutor) analyzeReconAndDetectServices(ctx context.Context, results []models.ToolOutput, target string) (*ai.AgentDecision, error) {
	fmt.Printf("[agentic] Phase 3.1: Detecting services from nmap output\n")

	// Find nmap output
	var nmapOutput string
	for _, result := range results {
		if result.ToolName == "nmap" {
			nmapOutput = result.Stdout
			break
		}
	}

	if nmapOutput == "" {
		fmt.Printf("[agentic] ⚠️  No nmap output available for service detection\n")
		return &ai.AgentDecision{
			RecommendedTools: ai.DefaultToolsForType("enum"),
			Analysis:         "No nmap output to analyze",
			Confidence:       0.5,
			StopScan:         false,
		}, nil
	}

	// Detect services
	services := ae.engine.aiClient.DetectServicesFromNmap(nmapOutput)
	if len(services) > 0 {
		summary := ae.engine.aiClient.SummarizeServices(services)
		fmt.Printf("[agentic] Service Detection Summary:\n%s", summary)

		// Get tools for detected services
		recommendedTools := ae.engine.aiClient.GetToolsForServices(services)
		fmt.Printf("[agentic] Recommended tools based on services: %v\n", recommendedTools)

		// Add default enum tools if none recommended
		if len(recommendedTools) == 0 {
			recommendedTools = ai.DefaultToolsForType("enum")
		}

		return &ai.AgentDecision{
			Analysis:         fmt.Sprintf("Service detection identified %d service(s). Proceeding to enumeration.", len(services)),
			RecommendedTools: recommendedTools,
			Reasoning:        summary,
			Confidence:       0.9,
			StopScan:         false,
		}, nil
	}

	fmt.Printf("[agentic] No services detected, using default tools\n")
	return &ai.AgentDecision{
		Analysis:         "No open ports detected",
		RecommendedTools: ai.DefaultToolsForType("enum"),
		Confidence:       0.5,
		StopScan:         false,
	}, nil
}

// generateFinalReport creates the AI-powered security assessment (C.5).
// Returns the report as Markdown; a JSON preview is printed to the console.
func (ae *AgenticExecutor) generateFinalReport(ctx context.Context, scanID string, target string) string {
	reportData, err := ae.GenerateFinalReport(ctx, scanID, target)
	if err != nil {
		return fmt.Sprintf("Report generation failed: %v", err)
	}

	if jsonStr, err := reportData.ToJSON(); err == nil {
		fmt.Printf("[agentic] Report JSON preview:\n%s\n", truncateString(jsonStr, 2000))
	}

	return reportData.ToMarkdown()
}
