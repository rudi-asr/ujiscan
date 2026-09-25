package playbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

func TestSeverityScore(t *testing.T) {
	cases := map[models.Severity]float64{
		models.SeverityCritical: 9.8,
		models.SeverityHigh:     7.5,
		models.SeverityMedium:   5.3,
		models.SeverityLow:      2.5,
		models.SeverityInfo:     0.0,
		"unknown":               0.0,
	}
	for sev, want := range cases {
		if got := SeverityScore(sev); got != want {
			t.Errorf("SeverityScore(%q) = %.1f, want %.1f", sev, got, want)
		}
	}
}

func TestPrioritizeFindings(t *testing.T) {
	findings := []models.Finding{
		{Title: "low issue", Severity: models.SeverityLow, ToolName: "nuclei"},
		{Title: "critical issue", Severity: models.SeverityCritical, ToolName: "nmap"},
		{Title: "medium issue", Severity: models.SeverityMedium},
		{Title: "high issue", Severity: models.SeverityHigh},
	}

	prioritized := PrioritizeFindings(findings)
	if len(prioritized) != 4 {
		t.Fatalf("expected 4 findings, got %d", len(prioritized))
	}

	// Sorted by risk desc
	if prioritized[0].Title != "critical issue" || prioritized[3].Title != "low issue" {
		t.Errorf("unexpected order: %v, %v", prioritized[0].Title, prioritized[3].Title)
	}
	if prioritized[0].Rank != 1 || prioritized[3].Rank != 4 {
		t.Errorf("unexpected ranks: %d, %d", prioritized[0].Rank, prioritized[3].Rank)
	}
	// Remediation defaults filled for all findings without explicit remediation
	if prioritized[0].Remediation == "" {
		t.Error("critical finding should have default remediation")
	}
	if prioritized[2].Remediation == "" {
		t.Error("medium finding should have default remediation")
	}
}

func TestBuildReportMetrics(t *testing.T) {
	findings := []models.Finding{
		{Title: "a", Severity: models.SeverityCritical, ToolName: "nmap"},
		{Title: "b", Severity: models.SeverityHigh, ToolName: "nmap"},
		{Title: "c", Severity: models.SeverityHigh, ToolName: "nuclei"},
		{Title: "d", Severity: models.SeverityInfo},
	}

	metrics := BuildReportMetrics(PrioritizeFindings(findings))
	if metrics.TotalFindings != 4 {
		t.Errorf("TotalFindings = %d", metrics.TotalFindings)
	}
	if metrics.Critical != 1 || metrics.High != 2 || metrics.Info != 1 {
		t.Errorf("counts wrong: %+v", metrics)
	}
	if len(metrics.ToolsUsed) != 2 {
		t.Errorf("expected 2 tools, got %v", metrics.ToolsUsed)
	}
	wantRisk := (9.8 + 7.5 + 7.5 + 0.0) / 4
	if metrics.RiskScore != wantRisk {
		t.Errorf("RiskScore = %.2f, want %.2f", metrics.RiskScore, wantRisk)
	}
}

func TestReportToJSON(t *testing.T) {
	report := &ReportData{
		ScanID:           "scan-1",
		Target:           "example.com",
		ExecutiveSummary: "summary",
		Findings: []*PrioritizedFinding{
			{Rank: 1, Title: "X", Severity: models.SeverityHigh, RiskScore: 7.5},
		},
		Metrics: ReportMetrics{TotalFindings: 1, High: 1, RiskScore: 7.5},
	}

	jsonStr, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["target"] != "example.com" {
		t.Errorf("unexpected target in JSON: %v", parsed["target"])
	}
}

func TestReportToMarkdown(t *testing.T) {
	report := &ReportData{
		ScanID:           "scan-1",
		Target:           "example.com",
		ExecutiveSummary: "Board summary here",
		Metrics:          ReportMetrics{TotalFindings: 1, Critical: 1, RiskScore: 9.8},
		Findings: []*PrioritizedFinding{
			{Rank: 1, Title: "RCE", Severity: models.SeverityCritical, RiskScore: 9.8, Remediation: "patch it"},
		},
	}

	md := report.ToMarkdown()
	for _, want := range []string{"# Security Scan Report", "example.com", "Executive Summary", "Board summary here", "RCE", "patch it"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n%s", want, md)
		}
	}
}

func TestReportToHTML(t *testing.T) {
	report := &ReportData{
		ScanID:           "scan-1",
		Target:           "example.com",
		ExecutiveSummary: "summary",
		Metrics:          ReportMetrics{TotalFindings: 0},
	}

	htmlStr := report.ToHTML()
	if !strings.Contains(htmlStr, "<html") || !strings.Contains(htmlStr, "No findings") {
		t.Errorf("empty HTML report malformed: %s", htmlStr)
	}
}

func TestBuildFallbackSummary(t *testing.T) {
	empty := ai.BuildFallbackSummary("example.com", map[string]int{})
	if !strings.Contains(empty, "no confirmed findings") {
		t.Errorf("unexpected empty summary: %s", empty)
	}

	urgent := ai.BuildFallbackSummary("example.com", map[string]int{"critical": 2, "high": 1})
	if !strings.Contains(urgent, "immediate remediation is required") {
		t.Errorf("unexpected urgent summary: %s", urgent)
	}
}
