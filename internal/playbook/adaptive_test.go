package playbook

import (
	"testing"
	"time"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

func TestShouldContinueToPhase(t *testing.T) {
	ae := &AgenticExecutor{}

	healthy := &ai.AgentDecision{
		Analysis:         "proceed",
		RecommendedTools: []string{"nuclei"},
		Confidence:       0.9,
		StopScan:         false,
	}

	prevOK := &PhaseResults{
		Phase:         "recon",
		TotalResults:  3,
		FindingsCount: 2,
		Success:       true,
		ExecutionTime: time.Second,
	}

	cases := []struct {
		name     string
		prev     *PhaseResults
		decision *ai.AgentDecision
		findings int
		want     bool
	}{
		{"healthy → continue", prevOK, healthy, 2, true},
		{"explicit stop", prevOK, &ai.AgentDecision{RecommendedTools: []string{"nuclei"}, Confidence: 0.9, StopScan: true}, 2, false},
		{"low confidence", prevOK, &ai.AgentDecision{RecommendedTools: []string{"nuclei"}, Confidence: 0.2, StopScan: false}, 2, false},
		{"no findings after enum", &PhaseResults{Phase: "enum", Success: true}, &ai.AgentDecision{RecommendedTools: []string{"nuclei"}, Confidence: 0.9, StopScan: false}, 0, false},
		{"no tools recommended", prevOK, &ai.AgentDecision{Analysis: "done", RecommendedTools: []string{}, Confidence: 0.9, StopScan: false}, 2, false},
		{"all tools failed", &PhaseResults{Phase: "recon", ToolsRun: []string{"nmap", "dig"}, Success: false}, healthy, 2, false},
		{"nil decision", prevOK, nil, 2, false},
		{"recon ok with findings → continue", &PhaseResults{Phase: "recon", TotalResults: 1, Success: true}, healthy, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ae.ShouldContinueToPhase(tc.prev, tc.decision, tc.findings); got != tc.want {
				t.Errorf("ShouldContinueToPhase() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAggregatePhaseResults(t *testing.T) {
	ae := &AgenticExecutor{}

	results := []struct {
		name string
		tool string
		exit int
		ms   int
	}{
		{"ok", "nmap", 0, 1200},
		{"fail", "nuclei", 1, 300},
	}

	pr := ae.AggregatePhaseResults("recon", []string{"nmap", "nuclei"}, toolOutputsFrom(results), 2*time.Second, 1)
	if pr == nil {
		t.Fatal("AggregatePhaseResults returned nil")
	}
	if pr.Phase != "recon" {
		t.Errorf("phase = %q", pr.Phase)
	}
	if pr.TotalResults != 2 {
		t.Errorf("TotalResults = %d, want 2", pr.TotalResults)
	}
	if len(pr.ToolOutcomes) != 2 {
		t.Fatalf("ToolOutcomes = %d, want 2", len(pr.ToolOutcomes))
	}

	// Order is not guaranteed in parallel; look up by name
	byName := map[string]bool{}
	for _, o := range pr.ToolOutcomes {
		byName[o.Tool] = o.Success
	}
	if !byName["nmap"] {
		t.Error("nmap should be marked success")
	}
	if byName["nuclei"] {
		t.Error("nuclei should be marked failed")
	}
	if pr.FindingsCount != 1 {
		t.Errorf("FindingsCount = %d, want 1", pr.FindingsCount)
	}
	if !pr.Success {
		t.Error("phase should be marked success (nmap succeeded)")
	}
}

// toolOutputsFrom converts a compact test table into []models.ToolOutput.
func toolOutputsFrom(items []struct {
	name string
	tool string
	exit int
	ms   int
}) []models.ToolOutput {
	out := make([]models.ToolOutput, len(items))
	for i, it := range items {
		errMsg := ""
		if it.exit != 0 {
			errMsg = "tool failed"
		}
		out[i] = models.ToolOutput{
			ToolName: it.tool,
			ExitCode: it.exit,
			Duration: it.ms,
			Success:  it.exit == 0,
			Error:    errMsg,
		}
	}
	return out
}
