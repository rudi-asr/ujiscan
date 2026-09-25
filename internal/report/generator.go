package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Generator creates pentest reports
type Generator struct {
	config ReportConfig
	dedup  *Deduplicator
}

// NewGenerator creates a new report generator
func NewGenerator(config ReportConfig, dedup *Deduplicator) *Generator {
	if dedup == nil {
		dedup = NewDeduplicator(DeduplicationConfig{
			SimilarityThreshold: 0.8,
			MatchType:           true,
			MatchTarget:         true,
			MatchEvidence:       true,
		})
	}

	return &Generator{
		config: config,
		dedup:  dedup,
	}
}

// Generate creates a complete report from findings
func (g *Generator) Generate(id, title, target string, findings []Finding) *Report {
	// Deduplicate findings
	dedupFindings := g.dedup.Deduplicate(findings)

	// Filter by minimum severity
	filtered := []Finding{}
	for _, f := range dedupFindings {
		if f.Severity >= g.config.MinimumSeverity {
			filtered = append(filtered, f)
		}
	}

	// Sort by severity if configured
	if g.config.SortBySeverity {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Severity > filtered[j].Severity
		})
	}

	// Build summary
	summary := g.buildSummary(filtered)

	startTime := time.Now().Add(-1 * time.Hour) // placeholder
	endTime := time.Now()

	report := &Report{
		ID:            id,
		Title:         title,
		Target:        target,
		StartTime:     startTime,
		EndTime:       endTime,
		ExecutionTime: fmt.Sprintf("%.2f hours", endTime.Sub(startTime).Hours()),
		Findings:      filtered,
		Summary:       summary,
		Methodology:   "OWASP Testing Guide v4.2 + PTES",
		Scope:         target,
		Disclaimer:    "This report is confidential and intended only for authorized use.",
	}

	return report
}

// buildSummary creates report summary from findings
func (g *Generator) buildSummary(findings []Finding) Summary {
	summary := Summary{
		TotalFindings:    len(findings),
		ToolsExecuted:    []string{},
		ExecutionSuccess: true,
	}

	toolsMap := make(map[string]bool)

	for _, f := range findings {
		// Count by severity
		switch {
		case f.Severity >= 9:
			summary.CriticalCount++
		case f.Severity >= 7:
			summary.HighCount++
		case f.Severity >= 4:
			summary.MediumCount++
		default:
			summary.LowCount++
		}

		// Collect tools
		if f.ToolName != "" {
			toolsMap[f.ToolName] = true
		}
	}

	// Convert map to slice
	for tool := range toolsMap {
		summary.ToolsExecuted = append(summary.ToolsExecuted, tool)
	}

	// Calculate risk score (0-100)
	summary.RiskScore = g.calculateRiskScore(summary)

	// Determine compliance status
	if summary.CriticalCount == 0 && summary.HighCount == 0 {
		summary.ComplianceStatus = "compliant"
	} else if summary.CriticalCount == 0 {
		summary.ComplianceStatus = "partial"
	} else {
		summary.ComplianceStatus = "non_compliant"
	}

	return summary
}

// calculateRiskScore computes overall risk score (0-100)
func (g *Generator) calculateRiskScore(s Summary) int {
	score := 0

	// Critical: 40 points
	score += s.CriticalCount * 40
	if score > 100 {
		score = 100
	}

	// High: 25 points (if < 100)
	if score < 100 {
		highScore := s.HighCount * 25
		if score+highScore > 100 {
			score = 100
		} else {
			score += highScore
		}
	}

	// Medium: 10 points (if < 100)
	if score < 100 {
		mediumScore := s.MediumCount * 10
		if score+mediumScore > 100 {
			score = 100
		} else {
			score += mediumScore
		}
	}

	// Low: 2 points (if < 100)
	if score < 100 {
		lowScore := s.LowCount * 2
		if score+lowScore > 100 {
			score = 100
		} else {
			score += lowScore
		}
	}

	return score
}

// ExportJSON exports report as JSON
func (g *Generator) ExportJSON(report *Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ExportCSV exports findings as CSV
func (g *Generator) ExportCSV(report *Report) string {
	csv := "ID,Type,Title,Target,Severity,CVSS,ToolName,Status\n"

	for _, f := range report.Findings {
		csv += fmt.Sprintf("%s,%s,%s,%s,%d,%.1f,%s,%s\n",
			f.ID, f.Type, f.Title, f.Target, f.Severity, f.CVSS, f.ToolName, f.Status)
	}

	return csv
}

// ExportHTML exports report as HTML (basic format)
func (g *Generator) ExportHTML(report *Report) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>` + report.Title + `</title>
  <style>
    body { font-family: Arial; margin: 20px; }
    h1 { color: #333; }
    .summary { background: #f0f0f0; padding: 10px; border-radius: 5px; }
    .finding { border: 1px solid #ddd; padding: 10px; margin: 10px 0; }
    .critical { border-left: 5px solid #d32f2f; }
    .high { border-left: 5px solid #f57c00; }
    .medium { border-left: 5px solid #fbc02d; }
    .low { border-left: 5px solid #388e3c; }
  </style>
</head>
<body>
  <h1>` + report.Title + `</h1>
  <div class="summary">
    <p><strong>Target:</strong> ` + report.Target + `</p>
    <p><strong>Findings:</strong> ` + fmt.Sprintf("%d", report.Summary.TotalFindings) + `</p>
    <p><strong>Risk Score:</strong> ` + fmt.Sprintf("%d/100", report.Summary.RiskScore) + `</p>
    <p><strong>Critical:</strong> ` + fmt.Sprintf("%d", report.Summary.CriticalCount) + ` | <strong>High:</strong> ` + fmt.Sprintf("%d", report.Summary.HighCount) + `</p>
  </div>
  <h2>Findings</h2>`

	for _, f := range report.Findings {
		severity := "low"
		if f.Severity >= 9 {
			severity = "critical"
		} else if f.Severity >= 7 {
			severity = "high"
		} else if f.Severity >= 4 {
			severity = "medium"
		}

		html += `<div class="finding ` + severity + `">
    <h3>` + f.Title + `</h3>
    <p><strong>Type:</strong> ` + f.Type + `</p>
    <p><strong>Severity:</strong> ` + fmt.Sprintf("%d/10", f.Severity) + `</p>
    <p><strong>Description:</strong> ` + f.Description + `</p>
    <p><strong>Remediation:</strong> ` + f.Remediation + `</p>
  </div>`
	}

	html += `</body>
</html>`

	return html
}
