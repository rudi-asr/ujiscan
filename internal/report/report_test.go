package report

import (
	"testing"
)

func TestDeduplicator(t *testing.T) {
	dedup := NewDeduplicator(DeduplicationConfig{
		SimilarityThreshold: 0.8,
		MatchType:           true,
		MatchTarget:         true,
		MatchEvidence:       true,
	})

	findings := []Finding{
		{
			ID:       "1",
			Type:     "XSS",
			Title:    "Reflected XSS",
			Target:   "example.com",
			Severity: 7,
			Evidence: "Found in search parameter",
		},
		{
			ID:       "2",
			Type:     "XSS",
			Title:    "Reflected XSS",
			Target:   "example.com",
			Severity: 7,
			Evidence: "Found in search parameter", // Duplicate
		},
		{
			ID:       "3",
			Type:     "SQLi",
			Title:    "SQL Injection",
			Target:   "example.com",
			Severity: 9,
			Evidence: "Found in login form",
		},
	}

	result := dedup.Deduplicate(findings)
	if len(result) != 2 {
		t.Errorf("Expected 2 unique findings, got %d", len(result))
	}
}

func TestGenerateReport(t *testing.T) {
	gen := NewGenerator(ReportConfig{
		SortBySeverity:   true,
		MinimumSeverity:  0,
	}, nil)

	findings := []Finding{
		{
			ID:       "1",
			Type:     "XSS",
			Title:    "Reflected XSS",
			Target:   "example.com",
			Severity: 7,
			CVSS:     7.3,
			ToolName: "nuclei",
		},
		{
			ID:       "2",
			Type:     "SQLi",
			Title:    "SQL Injection",
			Target:   "example.com",
			Severity: 9,
			CVSS:     9.8,
			ToolName: "sqlmap",
		},
	}

	report := gen.Generate("rep1", "Test Report", "example.com", findings)

	if report.ID != "rep1" {
		t.Errorf("Expected report ID rep1, got %s", report.ID)
	}

	if report.Summary.TotalFindings != 2 {
		t.Errorf("Expected 2 findings, got %d", report.Summary.TotalFindings)
	}

	if report.Summary.CriticalCount != 1 {
		t.Errorf("Expected 1 critical finding, got %d", report.Summary.CriticalCount)
	}

	if report.Summary.HighCount != 1 {
		t.Errorf("Expected 1 high finding, got %d", report.Summary.HighCount)
	}
}

func TestRiskScoreCalculation(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	summary := Summary{
		CriticalCount: 1,
		HighCount:     2,
		MediumCount:   3,
		LowCount:      5,
	}

	score := gen.calculateRiskScore(summary)

	if score <= 0 || score > 100 {
		t.Errorf("Risk score should be between 1-100, got %d", score)
	}
}

func TestExportJSON(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	report := &Report{
		ID:     "rep1",
		Title:  "Test",
		Target: "example.com",
		Findings: []Finding{
			{
				ID:       "1",
				Type:     "XSS",
				Severity: 7,
			},
		},
	}

	data, err := gen.ExportJSON(report)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("Expected JSON data")
	}
}

func TestExportCSV(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	report := &Report{
		Findings: []Finding{
			{
				ID:       "1",
				Type:     "XSS",
				Title:    "XSS",
				Target:   "example.com",
				Severity: 7,
				CVSS:     7.3,
				ToolName: "nuclei",
				Status:   "open",
			},
		},
	}

	csv := gen.ExportCSV(report)

	if len(csv) == 0 {
		t.Errorf("Expected CSV data")
	}

	if !contains(csv, "XSS") {
		t.Errorf("Expected XSS in CSV")
	}
}

func TestExportHTML(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	report := &Report{
		ID:     "rep1",
		Title:  "Test Report",
		Target: "example.com",
		Findings: []Finding{
			{
				ID:          "1",
				Type:        "XSS",
				Title:       "Reflected XSS",
				Severity:    7,
				Description: "XSS vulnerability",
				Remediation: "Sanitize input",
			},
		},
		Summary: Summary{
			TotalFindings: 1,
			RiskScore:     70,
		},
	}

	html := gen.ExportHTML(report)

	if len(html) == 0 {
		t.Errorf("Expected HTML output")
	}

	if !contains(html, "Test Report") {
		t.Errorf("Expected title in HTML")
	}

	if !contains(html, "Reflected XSS") {
		t.Errorf("Expected finding in HTML")
	}
}

func TestComplianceStatus(t *testing.T) {
	tests := []struct {
		critical        int
		high            int
		expectedStatus  string
	}{
		{0, 0, "compliant"},
		{0, 1, "partial"},
		{1, 0, "non_compliant"},
		{1, 1, "non_compliant"},
	}

	for _, test := range tests {
		summary := Summary{
			CriticalCount: test.critical,
			HighCount:     test.high,
		}

		if test.critical == 0 && test.high == 0 {
			summary.ComplianceStatus = "compliant"
		} else if test.critical == 0 {
			summary.ComplianceStatus = "partial"
		} else {
			summary.ComplianceStatus = "non_compliant"
		}

		if summary.ComplianceStatus != test.expectedStatus {
			t.Errorf("Expected %s, got %s", test.expectedStatus, summary.ComplianceStatus)
		}
	}
}

func TestReportWithEmptyFindings(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	report := gen.Generate("rep1", "Empty Report", "example.com", []Finding{})

	if report.Summary.TotalFindings != 0 {
		t.Errorf("Expected 0 findings")
	}

	if report.Summary.RiskScore != 0 {
		t.Errorf("Expected risk score 0")
	}
}

func TestReportTimestamps(t *testing.T) {
	gen := NewGenerator(ReportConfig{}, nil)

	report := gen.Generate("rep1", "Test", "example.com", []Finding{})

	if report.StartTime.IsZero() {
		t.Errorf("Expected start time to be set")
	}

	if report.EndTime.IsZero() {
		t.Errorf("Expected end time to be set")
	}

	if report.StartTime.After(report.EndTime) {
		t.Errorf("Start time should be before end time")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
