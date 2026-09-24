// Package reports provides JSON and export functionality
package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// JSONExporter exports reports as JSON
type JSONExporter struct{}

// NewJSONExporter creates a JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export writes report as JSON file
func (e *JSONExporter) Export(report *Report, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(outputPath, data, 0644)
}

// ExportString returns report as JSON string
func (e *JSONExporter) ExportString(report *Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MarkdownExporter exports reports as Markdown
type MarkdownExporter struct{}

// NewMarkdownExporter creates a Markdown exporter
func NewMarkdownExporter() *MarkdownExporter {
	return &MarkdownExporter{}
}

// Export writes report as Markdown file
func (e *MarkdownExporter) Export(report *Report, outputPath string) string {
	return e.generateMarkdown(report)
}

// ExportToFile writes report as Markdown file to disk
func (e *MarkdownExporter) ExportToFile(report *Report, outputPath string) error {
	content := e.generateMarkdown(report)
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// generateMarkdown creates markdown content
func (e *MarkdownExporter) generateMarkdown(report *Report) string {
	md := fmt.Sprintf(`# Security Assessment Report

**Report ID:** %s  
**Generated:** %s  
**Tool:** ujiscan v1.0

---

## Executive Summary

- **Target:** %s
- **Scope:** %s
- **Assessment Date:** %s
- **Duration:** %d minutes
- **Total Findings:** %d

### Risk Assessment

- **Risk Level:** %s
- **Risk Score:** %.1f/10
- **Critical Issues:** %d
- **High Issues:** %d

---

## Findings Summary

| Severity | Count |
|----------|-------|
| 🔴 Critical | %d |
| 🟠 High | %d |
| 🟡 Medium | %d |
| 🟢 Low | %d |
| ℹ️ Info | %d |

---

## Detailed Findings

`,
		report.Metadata.ReportID,
		report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"),
		report.Engagement.Target,
		fmt.Sprintf("%v", report.Engagement.Scope),
		report.Engagement.CompletedAt.Format("2006-01-02"),
		report.Engagement.DurationMins,
		report.Summary.FindingsCount,
		report.Summary.RiskLevel,
		report.Summary.RiskScore,
		report.Summary.SeverityDistribution.Critical,
		report.Summary.SeverityDistribution.High,
		report.Summary.SeverityDistribution.Critical,
		report.Summary.SeverityDistribution.High,
		report.Summary.SeverityDistribution.Medium,
		report.Summary.SeverityDistribution.Low,
		report.Summary.SeverityDistribution.Info,
	)

	for i, f := range report.Findings {
		md += fmt.Sprintf(`
### Finding %d: %s

**Severity:** %s | **CVSS:** %s | **CWE:** %s

**Description:**
%s

**Impact:**
%s

**Remediation:**
`,
			i+1, f.Title, f.Severity, f.CVSSV3, f.CWE, f.Description, f.Impact)

		for _, step := range f.Remediation.Steps {
			md += fmt.Sprintf("- %s\n", step)
		}

		md += fmt.Sprintf(`
**Estimated Effort:** %s

---

`, f.Remediation.EstimatedEffort)
	}

	md += fmt.Sprintf(`
## Remediation Roadmap

| Week | Milestone | Tasks |
|------|-----------|-------|
`)

	for _, item := range report.Remediation.Timeline {
		md += fmt.Sprintf("| %d | %s | %v |\n", item.Week, item.Milestone, item.Tasks)
	}

	return md
}

// ExportManager handles multi-format export
type ExportManager struct {
	jsonExp  *JSONExporter
	htmlRend *HTMLRenderer
	mdExp    *MarkdownExporter
}

// NewExportManager creates an export manager
func NewExportManager() *ExportManager {
	return &ExportManager{
		jsonExp:  NewJSONExporter(),
		htmlRend: NewHTMLRenderer("", "#667eea"),
		mdExp:    NewMarkdownExporter(),
	}
}

// ExportAll exports report in all formats
func (m *ExportManager) ExportAll(report *Report, outputDir string) (map[string]string, error) {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	paths := make(map[string]string)
	timestamp := time.Now().Format("20060102_150405")
	
	// JSON
	jsonPath := filepath.Join(outputDir, fmt.Sprintf("report_%s.json", timestamp))
	if err := m.jsonExp.Export(report, jsonPath); err != nil {
		return nil, err
	}
	paths["json"] = jsonPath

	// HTML
	htmlPath := filepath.Join(outputDir, fmt.Sprintf("report_%s.html", timestamp))
	htmlContent := m.htmlRend.Render(report)
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return nil, err
	}
	paths["html"] = htmlPath

	// Markdown
	mdPath := filepath.Join(outputDir, fmt.Sprintf("report_%s.md", timestamp))
	if err := m.mdExp.ExportToFile(report, mdPath); err != nil {
		return nil, err
	}
	paths["markdown"] = mdPath

	return paths, nil
}

// ExportFormat exports in specific format
func (m *ExportManager) ExportFormat(report *Report, outputPath, format string) error {
	switch format {
	case "json":
		return m.jsonExp.Export(report, outputPath)
	case "html":
		htmlContent := m.htmlRend.Render(report)
		return os.WriteFile(outputPath, []byte(htmlContent), 0644)
	case "markdown", "md":
		return m.mdExp.ExportToFile(report, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// GetReportSummary returns a brief summary
func GetReportSummary(report *Report) string {
	return fmt.Sprintf(`
Report Summary
──────────────
ID: %s
Target: %s
Mode: %s
Findings: %d
Critical: %d
Risk Level: %s (%.1f/10)
Duration: %d mins
Generated: %s
`,
		report.Metadata.ReportID,
		report.Engagement.Target,
		report.Engagement.Mode,
		report.Summary.FindingsCount,
		report.Summary.SeverityDistribution.Critical,
		report.Summary.RiskLevel,
		report.Summary.RiskScore,
		report.Engagement.DurationMins,
		report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"),
	)
}
