package agent

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/tools"
)

// VulnResult represents a single vulnerability finding detected by the scanner agent.
type VulnResult struct {
	TemplateID string `json:"template_id,omitempty"`
	Type       string `json:"type,omitempty"`
	Name       string `json:"name"`
	Severity   string `json:"severity"`
	MatchedAt  string `json:"matched_at,omitempty"`
	Host       string `json:"host,omitempty"`
}

// ScanResult is the structured output of a scanner task.
type ScanResult struct {
	Target          string       `json:"target"`
	Vulnerabilities []VulnResult `json:"vulnerabilities"`
	Count           int          `json:"count"`
	Duration        int          `json:"duration_ms"`
}

// ScannerAgent performs vulnerability scanning via nuclei.
type ScannerAgent struct {
	BaseAgent
	executor *tools.Executor
}

// NewScannerAgent creates a scanner agent backed by the tool executor.
func NewScannerAgent(executor *tools.Executor) *ScannerAgent {
	return &ScannerAgent{
		BaseAgent: NewBaseAgent(AgentTypeScanner),
		executor:  executor,
	}
}

// Execute runs a nuclei scan according to task params:
//
//	params.target (required) — URL or host to scan
//	params.args    (optional) — extra nuclei CLI arguments
func (s *ScannerAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	if err := s.Validate(task); err != nil {
		return nil, err
	}

	target := getStringParam(task.Params, "target")
	if target == "" {
		target = getStringParam(task.Params, "target_url")
	}

	// Ensure nuclei is actually registered before queuing work
	if _, ok := s.executor.ListTools()["nuclei"]; !ok {
		return nil, errors.New("nuclei tool not registered on this host")
	}

	start := time.Now()

	out, err := s.executor.Execute(ctx, "nuclei", map[string]string{
		"target": target,
		"args":   getStringParam(task.Params, "args"),
	})
	if err != nil {
		s.recordFailure(start)
		return nil, fmt.Errorf("nuclei scan failed: %w", err)
	}
	if ctx.Err() != nil {
		s.recordFailure(start)
		return nil, ctx.Err()
	}
	if out == nil || !out.Success {
		s.recordFailure(start)
		return nil, fmt.Errorf("nuclei failed (exit=%d)", func() int {
			if out != nil {
				return out.ExitCode
			}
			return -1
		}())
	}

	vulns := parseNucleiText(out.Stdout)

	result := &ScanResult{
		Target:          target,
		Vulnerabilities: vulns,
		Count:           len(vulns),
		Duration:        int(time.Since(start).Milliseconds()),
	}

	s.recordSuccess(start)
	return result, nil
}

// Validate ensures the scanner task is well-formed.
func (s *ScannerAgent) Validate(task *Task) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	if getStringParam(task.Params, "target") == "" && getStringParam(task.Params, "target_url") == "" {
		return errors.New("params.target (or target_url) required")
	}
	return nil
}

// Stop terminates the agent.
func (s *ScannerAgent) Stop() error {
	s.Status = StatusTerminated
	return nil
}

// --- nuclei text output parsing ---

// nucleiMatchRe matches nuclei's default console lines:
//
//	[template-name] [severity] name [matched-at]
var nucleiMatchRe = regexp.MustCompile(`^\[([^\]]+)\]\s*\[([^\]]+)\]\s*(.+?)(?:\s*\[(http[^\]]*)\])?\s*$`)

// parseNucleiText parses nuclei's human-readable output. Malformed lines are
// skipped; a match without a severity tag defaults to info.
func parseNucleiText(output string) []VulnResult {
	var results []VulnResult

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := nucleiMatchRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		severity := normalizeSeverity(m[2])
		if severity == "" {
			severity = "info"
		}
		results = append(results, VulnResult{
			TemplateID: m[1],
			Type:       "",
			Name:       strings.TrimSpace(m[3]),
			Severity:   severity,
			MatchedAt:  strings.TrimSpace(m[4]),
			Host:       "",
		})
	}

	return results
}
