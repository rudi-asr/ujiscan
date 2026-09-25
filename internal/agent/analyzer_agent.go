package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/ai"
)

// AnalyzedFinding is a classified, contextualized finding produced by the analyzer.
type AnalyzedFinding struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	CVSS        string `json:"cvss"`
	CWE         string `json:"cwe,omitempty"`
	Remediation string `json:"remediation"`
}

// AnalysisResult is the structured output of an analyzer task.
type AnalysisResult struct {
	Count    int                `json:"count"`
	Findings []*AnalyzedFinding `json:"findings"`
	Method   string             `json:"method"` // "local" | "claude"
}

// AnalyzerAgent parses, correlates, and contextualizes findings.
// It classifies locally (deterministic) and, when a Claude client is
// available, enriches the corpus with remediation guidance. A Claude
// failure NEVER fails the task — local analysis is the fallback.
type AnalyzerAgent struct {
	BaseAgent
	ai *ai.ClaudeClient // optional; nil = local-only analysis
}

// NewAnalyzerAgent creates an analyzer agent. Pass nil to skip AI enrichment.
func NewAnalyzerAgent(aiClient *ai.ClaudeClient) *AnalyzerAgent {
	return &AnalyzerAgent{
		BaseAgent: NewBaseAgent(AgentTypeAnalyzer),
		ai:        aiClient,
	}
}

// Execute analyzes raw findings from task params:
//
//	params.raw_findings (required) — []interface{} of finding maps with
//	                               title/name, severity (optional), description
func (a *AnalyzerAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	if err := a.Validate(task); err != nil {
		return nil, err
	}

	raw, ok := task.Params["raw_findings"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, errors.New("params.raw_findings must be a non-empty array")
	}

	start := time.Now()
	method := "local"

	findings := make([]*AnalyzedFinding, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		f := classifyFinding(m)
		if f != nil {
			findings = append(findings, f)
		}
	}

	// Optional Claude enrichment (best-effort, never fatal)
	if a.ai != nil && len(findings) > 0 {
		enriched, err := a.enrichWithClaude(ctx, findings)
		if err == nil && len(enriched) > 0 {
			findings = enriched
			method = "claude"
		}
	}

	result := &AnalysisResult{
		Count:    len(findings),
		Findings: findings,
		Method:   method,
	}

	a.recordSuccess(start)
	return result, nil
}

// Validate ensures the analyzer task is well-formed.
func (a *AnalyzerAgent) Validate(task *Task) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	return nil
}

// Stop terminates the agent.
func (a *AnalyzerAgent) Stop() error {
	a.Status = StatusTerminated
	return nil
}

// classifyFinding maps a raw finding map to a structured AnalyzedFinding
// with deterministic severity/CVSS normalization.
func classifyFinding(m map[string]interface{}) *AnalyzedFinding {
	title := firstString(m, "title", "name", "finding")
	if title == "" {
		return nil
	}

	severity := normalizeSeverity(firstString(m, "severity"))
	if severity == "" {
		severity = "info"
	}

	cvss := firstString(m, "cvss", "cvss_v3", "cvss_score")
	if cvss == "" {
		cvss = defaultCVSS(severity)
	}

	return &AnalyzedFinding{
		Title:       title,
		Description: firstString(m, "description", "detail"),
		Severity:    severity,
		CVSS:        cvss,
		CWE:         firstString(m, "cwe"),
		Remediation: defaultRemediation(severity),
	}
}

// enrichWithClaude asks Claude to rank and enrich findings. On any error the
// original local findings are kept (graceful degradation).
func (a *AnalyzerAgent) enrichWithClaude(ctx context.Context, findings []*AnalyzedFinding) ([]*AnalyzedFinding, error) {
	summary := make([]map[string]string, 0, len(findings))
	for _, f := range findings {
		summary = append(summary, map[string]string{
			"title":    f.Title,
			"severity": f.Severity,
			"cvss":     f.CVSS,
		})
	}
	payload, _ := json.Marshal(summary)

	prompt := "You are a senior penetration testing analyst. Given these findings, " +
		"return STRICT JSON (no markdown) as an array of objects with keys " +
		"title, description, severity (info|low|medium|high|critical), cvss (string), " +
		"cwe, remediation. Findings: " + string(payload)

	resp, err := a.ai.Chat(ctx, []ai.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Best-effort JSON extraction: strip code fences if present
	text := resp
	if i := strings.Index(text, "["); i >= 0 {
		text = text[i:]
	}
	if i := strings.LastIndex(text, "]"); i >= 0 {
		text = text[:i+1]
	}

	var parsed []*AnalyzedFinding
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("claude response is not valid JSON: %w", err)
	}
	if len(parsed) == 0 {
		return nil, errors.New("claude returned no findings")
	}
	// Re-normalize whatever Claude produced
	for _, f := range parsed {
		if f == nil {
			continue
		}
		f.Severity = normalizeSeverity(f.Severity)
		if f.CVSS == "" {
			f.CVSS = defaultCVSS(f.Severity)
		}
		if f.Remediation == "" {
			f.Remediation = defaultRemediation(f.Severity)
		}
	}
	return parsed, nil
}

// firstString returns the first non-empty string value for the given keys.
func firstString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// normalizeSeverity maps free-form severity strings to the canonical set.
func normalizeSeverity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical", "crit", "9", "10":
		return "critical"
	case "high", "7", "8":
		return "high"
	case "medium", "moderate", "4", "5", "6":
		return "medium"
	case "low", "1", "2", "3":
		return "low"
	case "info", "informational", "0":
		return "info"
	default:
		return ""
	}
}

// defaultCVSS returns a representative CVSS v3.1 base score per severity.
func defaultCVSS(severity string) string {
	switch severity {
	case "critical":
		return "9.8"
	case "high":
		return "7.5"
	case "medium":
		return "5.3"
	case "low":
		return "2.5"
	default:
		return "0.0"
	}
}

// defaultRemediation returns a generic remediation start per severity.
func defaultRemediation(severity string) string {
	switch severity {
	case "critical", "high":
		return "Remediate immediately: patch the affected component, add WAF/input validation, and re-test after the fix."
	case "medium":
		return "Schedule remediation: harden configuration, apply vendor patches, and monitor for exploitation."
	case "low":
		return "Plan remediation in the next maintenance window; document the accepted risk if deferred."
	default:
		return "Informational finding — no immediate action required."
	}
}
