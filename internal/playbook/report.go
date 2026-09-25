package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// === C.5: Report Generation & Finding Prioritization ===

// PrioritizedFinding is a scan finding with a computed risk score and rank.
type PrioritizedFinding struct {
	Rank        int             `json:"rank"`
	Title       string          `json:"title"`
	Severity    models.Severity `json:"severity"`
	RiskScore   float64         `json:"risk_score"`
	ToolName    string          `json:"tool_name"`
	Hostname    string          `json:"hostname,omitempty"`
	IPAddress   string          `json:"ip_address,omitempty"`
	Service     string          `json:"service,omitempty"`
	Version     string          `json:"version,omitempty"`
	Description string          `json:"description"`
	Evidence    string          `json:"evidence"`
	Remediation string          `json:"remediation"`
}

// ReportMetrics aggregates severity counts and overall risk.
type ReportMetrics struct {
	TotalFindings int      `json:"total_findings"`
	Critical      int      `json:"critical"`
	High          int      `json:"high"`
	Medium        int      `json:"medium"`
	Low           int      `json:"low"`
	Info          int      `json:"info"`
	RiskScore     float64  `json:"risk_score"` // 0-10 weighted average
	ToolsUsed     []string `json:"tools_used"`
}

// ReportData is the complete structured agentic report (C.5).
type ReportData struct {
	ScanID           string                `json:"scan_id"`
	Target           string                `json:"target"`
	GeneratedAt      time.Time             `json:"generated_at"`
	ExecutiveSummary string                `json:"executive_summary"`
	Findings         []*PrioritizedFinding `json:"findings"`
	Metrics          ReportMetrics         `json:"metrics"`
}

// SeverityScore maps a severity to a CVSS-like base score (C.5 prioritization).
func SeverityScore(sev models.Severity) float64 {
	switch sev {
	case models.SeverityCritical:
		return 9.8
	case models.SeverityHigh:
		return 7.5
	case models.SeverityMedium:
		return 5.3
	case models.SeverityLow:
		return 2.5
	default:
		return 0.0
	}
}

// DefaultRemediation returns a generic remediation start per severity.
func DefaultRemediation(sev models.Severity) string {
	switch sev {
	case models.SeverityCritical, models.SeverityHigh:
		return "Remediate immediately: patch the affected component, add WAF/input validation, and re-test after the fix."
	case models.SeverityMedium:
		return "Schedule remediation: harden configuration, apply vendor patches, and monitor for exploitation."
	case models.SeverityLow:
		return "Plan remediation in the next maintenance window; document the accepted risk if deferred."
	default:
		return "Informational finding — no immediate action required."
	}
}

// PrioritizeFindings sorts findings by risk score (highest first) and ranks them.
func PrioritizeFindings(findings []models.Finding) []*PrioritizedFinding {
	ordered := make([]*PrioritizedFinding, 0, len(findings))
	for i := range findings {
		f := &findings[i]
		pf := &PrioritizedFinding{
			Title:       f.Title,
			Severity:    f.Severity,
			RiskScore:   SeverityScore(f.Severity),
			ToolName:    f.ToolName,
			Hostname:    f.Hostname,
			IPAddress:   f.IPAddress,
			Service:     f.Service,
			Version:     f.Version,
			Description: f.Description,
			Evidence:    f.Evidence,
			Remediation: f.Remediation,
		}
		if pf.Remediation == "" {
			pf.Remediation = DefaultRemediation(f.Severity)
		}
		ordered = append(ordered, pf)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].RiskScore > ordered[j].RiskScore
	})
	for i, pf := range ordered {
		pf.Rank = i + 1
	}
	return ordered
}

// BuildReportMetrics aggregates findings into ReportMetrics.
func BuildReportMetrics(findings []*PrioritizedFinding) ReportMetrics {
	m := ReportMetrics{TotalFindings: len(findings)}
	toolSet := map[string]bool{}
	var riskSum float64

	for _, pf := range findings {
		switch pf.Severity {
		case models.SeverityCritical:
			m.Critical++
		case models.SeverityHigh:
			m.High++
		case models.SeverityMedium:
			m.Medium++
		case models.SeverityLow:
			m.Low++
		default:
			m.Info++
		}
		riskSum += pf.RiskScore
		if pf.ToolName != "" {
			toolSet[pf.ToolName] = true
		}
	}

	for t := range toolSet {
		m.ToolsUsed = append(m.ToolsUsed, t)
	}
	sort.Strings(m.ToolsUsed)

	if len(findings) > 0 {
		m.RiskScore = riskSum / float64(len(findings))
	}
	return m
}

// GenerateFinalReport builds the complete AI-driven report for a scan (C.5).
func (ae *AgenticExecutor) GenerateFinalReport(ctx context.Context, scanID, target string) (*ReportData, error) {
	scan, err := ae.engine.scanStore.GetScan(scanID)
	if err != nil {
		return nil, fmt.Errorf("scan not found: %w", err)
	}

	priority := PrioritizeFindings(scan.Findings)
	metrics := BuildReportMetrics(priority)

	counts := map[string]int{
		"critical": metrics.Critical,
		"high":     metrics.High,
		"medium":   metrics.Medium,
		"low":      metrics.Low,
		"info":     metrics.Info,
	}

	summary, err := ae.aiClient.GenerateExecutiveSummary(ctx, target, counts)
	if err != nil {
		summary = ai.BuildFallbackSummary(target, counts)
	}

	return &ReportData{
		ScanID:           scanID,
		Target:           target,
		GeneratedAt:      time.Now().UTC(),
		ExecutiveSummary: summary,
		Findings:         priority,
		Metrics:          metrics,
	}, nil
}

// ToJSON serializes the report as indented JSON.
func (r *ReportData) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToMarkdown renders the report as Markdown.
func (r *ReportData) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Security Scan Report — %s\n\n", r.Target))
	sb.WriteString(fmt.Sprintf("- **Scan ID:** `%s`\n", r.ScanID))
	sb.WriteString(fmt.Sprintf("- **Generated:** %s\n", r.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Risk Score:** %.1f / 10\n\n", r.Metrics.RiskScore))

	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(r.ExecutiveSummary + "\n\n")

	sb.WriteString("## Metrics\n\n")
	sb.WriteString("| Severity | Count |\n|---|---|\n")
	sb.WriteString(fmt.Sprintf("| Critical | %d |\n", r.Metrics.Critical))
	sb.WriteString(fmt.Sprintf("| High | %d |\n", r.Metrics.High))
	sb.WriteString(fmt.Sprintf("| Medium | %d |\n", r.Metrics.Medium))
	sb.WriteString(fmt.Sprintf("| Low | %d |\n", r.Metrics.Low))
	sb.WriteString(fmt.Sprintf("| Info | %d |\n\n", r.Metrics.Info))

	if len(r.Findings) == 0 {
		sb.WriteString("## Findings\n\n_No findings._\n")
		return sb.String()
	}

	sb.WriteString("## Findings (by priority)\n\n")
	for _, f := range r.Findings {
		sb.WriteString(fmt.Sprintf("### %d. [%s] %s (score %.1f)\n\n", f.Rank, f.Severity, f.Title, f.RiskScore))
		if f.Description != "" {
			sb.WriteString(f.Description + "\n\n")
		}
		if f.Evidence != "" {
			sb.WriteString(fmt.Sprintf("**Evidence:** `%s`\n\n", truncateString(f.Evidence, 300)))
		}
		if f.Remediation != "" {
			sb.WriteString(fmt.Sprintf("**Remediation:** %s\n\n", f.Remediation))
		}
	}

	return sb.String()
}

// ToHTML renders the report as a self-contained dark-themed HTML page.
func (r *ReportData) ToHTML() string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html><html lang=\"en\" data-theme=\"dark\"><head><meta charset=\"UTF-8\">")
	sb.WriteString("<title>ujiscan Report — " + r.Target + "</title><style>")
	sb.WriteString("body{background:#0a0e27;color:#e0e0e0;font-family:system-ui,sans-serif;max-width:900px;margin:0 auto;padding:2rem}")
	sb.WriteString("h1{color:#00d4ff}table{border-collapse:collapse;width:100%}td,th{border:1px solid #333;padding:.5rem;text-align:left}")
	sb.WriteString(".critical{color:#ff4444}.high{color:#ff8800}.medium{color:#ffdd00}.low{color:#88ff88}.info{color:#888}")
	sb.WriteString("pre{background:#1a1f3a;padding:.8rem;overflow-x:auto;border-radius:6px}</style></head><body>")

	sb.WriteString(fmt.Sprintf("<h1>Security Scan Report — %s</h1>\n", r.Target))
	sb.WriteString(fmt.Sprintf("<p><strong>Scan ID:</strong> <code>%s</code> — <strong>Generated:</strong> %s — <strong>Risk:</strong> %.1f/10</p>\n",
		r.ScanID, r.GeneratedAt.Format(time.RFC3339), r.Metrics.RiskScore))

	sb.WriteString("<h2>Executive Summary</h2><p>" + r.ExecutiveSummary + "</p>\n")

	sb.WriteString("<h2>Metrics</h2><table><tr><th>Severity</th><th>Count</th></tr>")
	sb.WriteString(fmt.Sprintf("<tr><td class=\"critical\">Critical</td><td>%d</td></tr>", r.Metrics.Critical))
	sb.WriteString(fmt.Sprintf("<tr><td class=\"high\">High</td><td>%d</td></tr>", r.Metrics.High))
	sb.WriteString(fmt.Sprintf("<tr><td class=\"medium\">Medium</td><td>%d</td></tr>", r.Metrics.Medium))
	sb.WriteString(fmt.Sprintf("<tr><td class=\"low\">Low</td><td>%d</td></tr>", r.Metrics.Low))
	sb.WriteString(fmt.Sprintf("<tr><td class=\"info\">Info</td><td>%d</td></tr></table>\n", r.Metrics.Info))

	if len(r.Findings) == 0 {
		sb.WriteString("<h2>Findings</h2><p><em>No findings.</em></p></body></html>")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("<h2>Findings (%d)</h2>\n", len(r.Findings)))
	for _, f := range r.Findings {
		sb.WriteString(fmt.Sprintf("<h3 class=\"%s\">%d. [%s] %s (score %.1f)</h3>\n", f.Severity, f.Rank, f.Severity, f.Title, f.RiskScore))
		if f.Description != "" {
			sb.WriteString("<p>" + f.Description + "</p>")
		}
		if f.Evidence != "" {
			sb.WriteString("<pre>" + htmlEscape(truncateString(f.Evidence, 300)) + "</pre>")
		}
		if f.Remediation != "" {
			sb.WriteString("<p><strong>Remediation:</strong> " + f.Remediation + "</p>")
		}
	}

	sb.WriteString("</body></html>")
	return sb.String()
}

// truncateString limits a string to maxLen runes.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// htmlEscape escapes HTML-sensitive characters in evidence text.
func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}
