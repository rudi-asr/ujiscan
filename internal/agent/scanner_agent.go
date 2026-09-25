package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
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
//	params.target      (required) — URL or host to scan
//	params.templates   (optional) — custom nuclei template path
func (s *ScannerAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	if err := s.Validate(task); err != nil {
		return nil, err
	}

	target := getStringParam(task.Params, "target")
	if target == "" {
		target = getStringParam(task.Params, "target_url")
	}

	// Ensure nuclei is actually available before queuing work
	cfg := s.executor.GetToolConfig("nuclei")
	if cfg == nil || !cfg.Available {
		return nil, errors.New("nuclei tool not available on this host")
	}

	start := time.Now()

	var out *models.ToolOutput
	var err error
	if templates := getStringParam(task.Params, "templates"); templates != "" {
		out, err = s.executor.ScanWithNucleiCustom(target, templates)
	} else {
		out, err = s.executor.ScanWithNuclei(target)
	}

	if err != nil {
		s.recordFailure(start)
		return nil, fmt.Errorf("nuclei scan failed: %w", err)
	}
	if ctx.Err() != nil {
		s.recordFailure(start)
		return nil, ctx.Err()
	}

	vulns := parseNucleiJSONL(out.Stdout)

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

// --- nuclei JSONL parsing ---

type nucleiLine struct {
	TemplateID string `json:"template-id"`
	Type       string `json:"type"`
	MatchedAt  string `json:"matched-at"`
	Host       string `json:"host"`
	Info       struct {
		Name     string `json:"name"`
		Severity string `json:"severity"`
	} `json:"info"`
}

// parseNucleiJSONL parses nuclei -jsonl output (one JSON object per line).
// Malformed lines are skipped; a malformed stream yields an empty slice.
func parseNucleiJSONL(data string) []VulnResult {
	var results []VulnResult
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var nl nucleiLine
		if err := json.Unmarshal([]byte(line), &nl); err != nil {
			continue
		}
		if nl.Info.Name == "" {
			continue
		}
		results = append(results, VulnResult{
			TemplateID: nl.TemplateID,
			Type:       nl.Type,
			Name:       nl.Info.Name,
			Severity:   nl.Info.Severity,
			MatchedAt:  nl.MatchedAt,
			Host:       nl.Host,
		})
	}
	return results
}
