package playbook

import (
	"context"
	"fmt"

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

// ExecuteAgenticScan runs full agentic scanning workflow
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

	// Phase 2: Execute reconnaissance phase
	fmt.Printf("[agentic] Phase 2: Executing reconnaissance tools\n")
	reconResults := ae.executeToolsPhase(ctx, scanID, "recon", selectedTools)
	fmt.Printf("[agentic] Recon phase completed: %d results\n", len(reconResults))

	// Phase 3: AI analyzes recon and recommends enumeration tools
	fmt.Printf("[agentic] Phase 3: AI analysis of reconnaissance findings\n")
	enumDecision, err := ae.analyzePhaseResults(ctx, reconResults, "recon", target)
	if err != nil {
		fmt.Printf("[agentic] Analysis failed: %v\n", err)
		enumDecision = &ai.AgentDecision{
			RecommendedTools: ai.DefaultToolsForType("enum"),
			StopScan:         false,
		}
	}

	if enumDecision.StopScan {
		fmt.Printf("[agentic] AI recommended stopping scan after recon\n")
		ae.engine.scanStore.UpdateScanStatus(scanID, models.ScanStatusCompleted)
		return nil
	}

	// Phase 4: Execute enumeration phase
	fmt.Printf("[agentic] Phase 4: Executing enumeration tools\n")
	enumResults := ae.executeToolsPhase(ctx, scanID, "enum", enumDecision.RecommendedTools)
	fmt.Printf("[agentic] Enum phase completed: %d results\n", len(enumResults))

	// Phase 5: AI analyzes enum results
	fmt.Printf("[agentic] Phase 5: AI analysis of enumeration findings\n")
	exploitDecision, err := ae.analyzePhaseResults(ctx, enumResults, "enum", target)
	if err != nil {
		exploitDecision = &ai.AgentDecision{
			RecommendedTools: ai.DefaultToolsForType("exploit"),
			StopScan:         false,
		}
	}

	// Phase 6: Execute exploitation/specialized tools if recommended
	if !exploitDecision.StopScan && len(exploitDecision.RecommendedTools) > 0 {
		fmt.Printf("[agentic] Phase 6: Executing exploitation tools\n")
		exploitResults := ae.executeToolsPhase(ctx, scanID, "exploit", exploitDecision.RecommendedTools)
		fmt.Printf("[agentic] Exploit phase completed: %d results\n", len(exploitResults))
	}

	// Phase 7: AI generates final report
	fmt.Printf("[agentic] Phase 7: Generating final AI report\n")
	finalReport := ae.generateFinalReport(ctx, scanID, target)
	fmt.Printf("[agentic] Final report: %s\n", finalReport)

	// Mark scan complete
	ae.engine.scanStore.UpdateScanStatus(scanID, models.ScanStatusCompleted)
	fmt.Printf("[agentic] ✅ Agentic scan completed for %s\n", scanID)

	return nil
}

// executeToolsPhase runs a set of tools and captures results
func (ae *AgenticExecutor) executeToolsPhase(ctx context.Context, scanID string, phase string, tools []string) []models.ToolOutput {
	var results []models.ToolOutput

	fmt.Printf("[agentic] Executing %d tools in %s phase: %v\n", len(tools), phase, tools)

	for _, toolName := range tools {
		// Execute tool via executor (target is scanme.nmap.org)
		fmt.Printf("[agentic] Executing %s...\n", toolName)
		
		params := map[string]string{
			"target": "scanme.nmap.org",
		}
		
		output, err := ae.engine.executor.Execute(ctx, toolName, params)
		if err != nil {
			fmt.Printf("[agentic] ⚠️  Tool %s failed: %v\n", toolName, err)
			continue
		}

		// Store result
		toolOutput := models.ToolOutput{
			ToolName: toolName,
			Stdout:   output.Stdout,
			Stderr:   output.Stderr,
			ExitCode: output.ExitCode,
		}

		results = append(results, toolOutput)

		// Store in scan
		if err := ae.engine.scanStore.AddResult(scanID, toolOutput); err != nil {
			fmt.Printf("[agentic] Failed to store result: %v\n", err)
		}

		fmt.Printf("[agentic] ✓ %s completed (exit=%d, stdout=%d bytes)\n", toolName, output.ExitCode, len(output.Stdout))
	}

	return results
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

// generateFinalReport creates final AI-powered security assessment
func (ae *AgenticExecutor) generateFinalReport(ctx context.Context, scanID string, target string) string {
	scan, err := ae.engine.scanStore.GetScan(scanID)
	if err != nil {
		return "Report generation failed"
	}

	summary := fmt.Sprintf("🔍 Security Scan Report for %s\n", target)
	summary += fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	summary += fmt.Sprintf("Scan ID: %s\n", scanID)
	summary += fmt.Sprintf("Tools Executed: %d\n", len(scan.Results))
	summary += fmt.Sprintf("Findings Discovered: %d\n\n", len(scan.Findings))

	// Group by severity
	severityMap := make(map[string]int)
	for _, finding := range scan.Findings {
		severityMap[string(finding.Severity)]++
	}

	summary += "Findings by Severity:\n"
	for severity, count := range severityMap {
		summary += fmt.Sprintf("  • %s: %d\n", severity, count)
	}

	summary += "\nTop Findings:\n"
	for i, finding := range scan.Findings {
		if i >= 5 {
			break
		}
		summary += fmt.Sprintf("  %d. [%s] %s (%s)\n", i+1, finding.Severity, finding.Title, finding.ToolName)
	}

	return summary
}
