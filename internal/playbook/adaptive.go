package playbook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// === C.4: Adaptive Execution Loop ===

// ToolOutcome records the result of one tool execution within a phase.
type ToolOutcome struct {
	Tool     string        `json:"tool"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration_ms"`
}

// PhaseResults aggregates the outcome of one adaptive phase (C.4.1).
type PhaseResults struct {
	Phase         string        `json:"phase"`
	ToolsRun      []string      `json:"tools_run"`
	TotalResults  int           `json:"total_results"`
	FindingsCount int           `json:"findings_count"`
	Success       bool          `json:"success"`
	ExecutionTime time.Duration `json:"execution_time_ms"`
	ToolOutcomes  []ToolOutcome `json:"tool_outcomes"`
}

// AggregatePhaseResults builds PhaseResults from raw tool outputs (C.4.1).
func (ae *AgenticExecutor) AggregatePhaseResults(phase string, requestedTools []string, results []models.ToolOutput, elapsed time.Duration, findingsDelta int) *PhaseResults {
	pr := &PhaseResults{
		Phase:         phase,
		ToolsRun:      requestedTools,
		TotalResults:  len(results),
		FindingsCount: findingsDelta,
		ExecutionTime: elapsed,
		ToolOutcomes:  make([]ToolOutcome, 0, len(results)),
	}

	for _, r := range results {
		success := r.ExitCode == 0 && r.Error == ""
		if success {
			pr.Success = true
		}
		pr.ToolOutcomes = append(pr.ToolOutcomes, ToolOutcome{
			Tool:     r.ToolName,
			Success:  success,
			Duration: time.Duration(r.Duration) * time.Millisecond,
		})
	}

	return pr
}

// ShouldContinueToPhase decides whether the adaptive loop proceeds (C.4.2).
// Heuristics:
//  1. AI explicitly recommends stopping (StopScan)            → stop
//  2. AI confidence below 0.4                                  → stop
//  3. No findings after a non-recon phase (diminishing returns) → stop
//  4. No tools recommended for the next phase                  → stop
//  5. Every tool in this phase failed                          → stop
func (ae *AgenticExecutor) ShouldContinueToPhase(prevResults *PhaseResults, decision *ai.AgentDecision, totalFindings int) bool {
	if decision == nil {
		fmt.Printf("[agentic] Decision is nil, stopping\n")
		return false
	}

	// 1. Explicit AI stop
	if decision.StopScan {
		fmt.Printf("[agentic] AI requested stop (stop_scan=true)\n")
		return false
	}

	// 2. Low confidence
	if decision.Confidence < 0.4 {
		fmt.Printf("[agentic] AI confidence %.2f below 0.4, stopping\n", decision.Confidence)
		return false
	}

	// 3. No findings after a non-recon phase
	if prevResults != nil && prevResults.Phase != "recon" && totalFindings == 0 {
		fmt.Printf("[agentic] No findings after %s phase, stopping (diminishing returns)\n", prevResults.Phase)
		return false
	}

	// 4. Nothing recommended for the next phase
	if len(decision.RecommendedTools) == 0 {
		fmt.Printf("[agentic] No tools recommended for next phase, stopping\n")
		return false
	}

	// 5. All tools failed in this phase
	if prevResults != nil && len(prevResults.ToolsRun) > 0 && !prevResults.Success {
		fmt.Printf("[agentic] All %d tools failed in %s phase, stopping\n", len(prevResults.ToolsRun), prevResults.Phase)
		return false
	}

	return true
}

// ExecuteToolsPhaseParallel runs a phase's tools concurrently (C.4.3).
// Up to maxConcurrentTools run at once; each tool gets a per-tool timeout.
func (ae *AgenticExecutor) ExecuteToolsPhaseParallel(ctx context.Context, scanID, phase, target string, tools []string) []models.ToolOutput {
	if len(tools) == 0 {
		return nil
	}

	const maxConcurrentTools = 4
	const perToolTimeout = 120 * time.Second

	fmt.Printf("[agentic] ⚡ Executing %d tools in %s phase (parallel, max %d): %v\n",
		len(tools), phase, maxConcurrentTools, tools)

	sem := make(chan struct{}, maxConcurrentTools)
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]models.ToolOutput, 0, len(tools))

	for _, toolName := range tools {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("[agentic]   └ %s…\n", name)
			start := time.Now()

			toolCtx, cancel := context.WithTimeout(ctx, perToolTimeout)
			defer cancel()

			out, err := ae.engine.executor.Execute(toolCtx, name, map[string]string{"target": target})
			duration := time.Since(start)

			to := models.ToolOutput{
				ToolName: name,
				Target:   target,
				Phase:    models.PhaseType(phase),
				ExitCode: -1,
				Duration: int(duration.Milliseconds()),
			}
			if err == nil && out != nil {
				to.Stdout = out.Stdout
				to.Stderr = out.Stderr
				to.ExitCode = out.ExitCode
				to.Success = out.ExitCode == 0
			} else if err != nil {
				to.Stderr = err.Error()
				to.Error = err.Error()
			}

			mu.Lock()
			results = append(results, to)
			mu.Unlock()

			fmt.Printf("[agentic]   └ %s done (exit=%d, %.1fs)\n", name, to.ExitCode, duration.Seconds())

			if err := ae.engine.scanStore.AddResult(scanID, to); err != nil {
				fmt.Printf("[agentic] Failed to store result: %v\n", err)
			}
		}(toolName)
	}

	wg.Wait()
	return results
}
