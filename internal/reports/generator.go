// Package reports provides report generation
package reports

import (
	"fmt"
	"time"
)

// Report represents a complete security assessment report
type Report struct {
	Metadata    ReportMetadata `json:"report_metadata"`
	Engagement  Engagement     `json:"engagement"`
	Summary     ReportSummary  `json:"summary"`
	Findings    []*Finding     `json:"findings"`
	Playbook    PlaybookExec   `json:"playbook_execution"`
	Compliance  []Compliance   `json:"compliance,omitempty"`
	Remediation RemediationPlan `json:"remediation_roadmap,omitempty"`
}

// ReportMetadata contains report generation info
type ReportMetadata struct {
	Version      string    `json:"version"`
	GeneratedAt  time.Time `json:"generated_at"`
	Tool         string    `json:"tool"`
	Generator    string    `json:"generator"`
	ReportID     string    `json:"report_id"`
	ExportFormat string    `json:"export_format"` // json, html, pdf, markdown
}

// Engagement contains engagement details
type Engagement struct {
	ClientName   string    `json:"client_name"`
	Target       string    `json:"target"`
	Scope        []string  `json:"scope"`
	Mode         string    `json:"mode"`
	PlaybookID   string    `json:"playbook_id"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	DurationMins int       `json:"duration_minutes"`
	Assessor     string    `json:"assessor,omitempty"`
	Status       string    `json:"status"` // completed, in_progress, failed
}

// ReportSummary contains findings summary
type ReportSummary struct {
	FindingsCount    int              `json:"findings_count"`
	SeverityDistribution SeverityDist `json:"severity"`
	Metrics          ScanMetrics      `json:"metrics"`
	RiskLevel        string           `json:"risk_level"` // critical, high, medium, low
	RiskScore        float32          `json:"risk_score"` // 0-10
}

// SeverityDist counts findings by severity
type SeverityDist struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// ScanMetrics contains automated scan metrics
type ScanMetrics struct {
	MeanCVSS          float32 `json:"mean_cvss"`
	MaxCVSS           float32 `json:"max_cvss"`
	ExploitableCount  int     `json:"exploitable_count"`
	RequiresAuthCount int     `json:"requires_auth_count"`
	RemoteCount       int     `json:"remote_exploitable_count"`
	ToolsExecuted     int     `json:"tools_executed"`
	PhasesCompleted   int     `json:"phases_completed"`
}

// Finding represents a single security finding
type Finding struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"` // vulnerability, misconfiguration, info
	Category       string            `json:"category"` // injection, auth, etc
	Title          string            `json:"title"`
	Severity       string            `json:"severity"`
	CVSSV3         string            `json:"cvss_v3"`
	CWE            string            `json:"cwe"`
	OWASP          string            `json:"owasp"`
	Target         string            `json:"target"` // URL or IP
	Parameter      string            `json:"parameter,omitempty"`
	Description    string            `json:"description"`
	Impact         string            `json:"impact"`
	Evidence       FindingEvidence   `json:"evidence"`
	Remediation    RemediationSteps  `json:"remediation"`
	References     FindingReferences `json:"references"`
}

// FindingEvidence contains proof of finding
type FindingEvidence struct {
	Tool             string `json:"tool"`
	Payload          string `json:"payload,omitempty"`
	ResponseCode     int    `json:"response_code,omitempty"`
	ResponseSnippet  string `json:"response_snippet,omitempty"`
	TimeDetected     string `json:"time_detected,omitempty"`
}

// RemediationSteps contains fix information
type RemediationSteps struct {
	Priority         string   `json:"priority"` // immediate, high, medium, low
	Steps            []string `json:"steps"`
	EstimatedEffort  string   `json:"estimated_effort"` // 1-2 days, etc
	Verification     string   `json:"verification"` // how to verify fix
	RiskIfNotFixed   string   `json:"risk_if_not_fixed,omitempty"`
}

// FindingReferences contains external links
type FindingReferences struct {
	CWELink   string `json:"cwe_link,omitempty"`
	OWASPLink string `json:"owasp_link,omitempty"`
	Payloads  string `json:"payloads,omitempty"`
	NVDLink   string `json:"nvd_link,omitempty"`
}

// PlaybookExec describes execution flow
type PlaybookExec struct {
	PlaybookName string         `json:"playbook_name"`
	Description  string         `json:"description"`
	Phases       []PhaseResult  `json:"phases"`
}

// PhaseResult contains phase execution details
type PhaseResult struct {
	Name           string   `json:"name"`
	Order          int      `json:"order"`
	Tools          []string `json:"tools"`
	DurationSecs   int      `json:"duration_seconds"`
	FindingsCount  int      `json:"findings_generated"`
	ToolsSucceeded int      `json:"tools_succeeded"`
	ToolsFailed    int      `json:"tools_failed"`
}

// Compliance maps findings to compliance standards
type Compliance struct {
	Standard      string   `json:"standard"` // PCI-DSS, HIPAA, SOC2, etc
	FindingCount  int      `json:"finding_count"`
	Status        string   `json:"status"` // compliant, non_compliant
	Mappings      []string `json:"mappings"` // PCI-DSS 6.5.1, etc
}

// RemediationPlan provides timeline for fixes
type RemediationPlan struct {
	CriticalDeadline time.Time `json:"critical_deadline"`
	HighDeadline     time.Time `json:"high_deadline"`
	Timeline         []TimelineItem `json:"timeline"`
}

// TimelineItem represents a milestone
type TimelineItem struct {
	Week        int      `json:"week"`
	Milestone   string   `json:"milestone"`
	Tasks       []string `json:"tasks"`
	Owner       string   `json:"owner,omitempty"`
	Completion  int      `json:"completion_percent"` // 0-100
}

// Generator builds reports from findings and execution data
type Generator struct{}

// NewGenerator creates a report generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateReport creates a report from execution data
func (g *Generator) GenerateReport(
	clientName string,
	target string,
	scope []string,
	mode string,
	playbookID string,
	findings []*Finding,
	duration int,
	startedAt time.Time,
) *Report {
	now := time.Now()
	
	// Calculate summary statistics
	summary := g.calculateSummary(findings)
	
	// Build playbook execution
	playbookExec := PlaybookExec{
		PlaybookName: mode,
		Description:  fmt.Sprintf("%s assessment for %s", mode, target),
		Phases:       g.buildPhases(findings),
	}

	// Build remediation roadmap
	roadmap := g.buildRemediationPlan(findings)

	report := &Report{
		Metadata: ReportMetadata{
			Version:     "1.0",
			GeneratedAt: now,
			Tool:        "ujiscan",
			Generator:   "agentic-pentest",
			ReportID:    fmt.Sprintf("rpt_%d", now.UnixNano()),
		},
		Engagement: Engagement{
			ClientName:  clientName,
			Target:      target,
			Scope:       scope,
			Mode:        mode,
			PlaybookID:  playbookID,
			StartedAt:   startedAt,
			CompletedAt: now,
			DurationMins: duration,
			Status:      "completed",
		},
		Summary:     summary,
		Findings:    findings,
		Playbook:    playbookExec,
		Remediation: roadmap,
	}

	return report
}

// calculateSummary computes summary statistics
func (g *Generator) calculateSummary(findings []*Finding) ReportSummary {
	summary := ReportSummary{
		FindingsCount: len(findings),
		Metrics: ScanMetrics{
			ToolsExecuted:   5, // TODO: from actual execution
			PhasesCompleted: 3, // TODO: from actual execution
		},
	}

	var totalCVSS float32
	var maxCVSS float32

	for _, f := range findings {
		// Count by severity
		switch f.Severity {
		case "critical":
			summary.SeverityDistribution.Critical++
		case "high":
			summary.SeverityDistribution.High++
		case "medium":
			summary.SeverityDistribution.Medium++
		case "low":
			summary.SeverityDistribution.Low++
		case "info":
			summary.SeverityDistribution.Info++
		}

		// Parse CVSS
		var cvss float32
		fmt.Sscanf(f.CVSSV3, "%f", &cvss)
		totalCVSS += cvss
		if cvss > maxCVSS {
			maxCVSS = cvss
		}

		// Count exploitable (no auth required)
		if f.Evidence.Payload != "" {
			summary.Metrics.ExploitableCount++
		}
	}

	if len(findings) > 0 {
		summary.Metrics.MeanCVSS = totalCVSS / float32(len(findings))
	}
	summary.Metrics.MaxCVSS = maxCVSS

	// Calculate risk level and score
	summary.RiskScore = g.calculateRiskScore(summary.SeverityDistribution)
	summary.RiskLevel = g.getRiskLevel(summary.RiskScore)

	return summary
}

// calculateRiskScore computes overall risk (0-10)
func (g *Generator) calculateRiskScore(sev SeverityDist) float32 {
	// Weighted scoring: critical=4pts, high=2pts, medium=1pt, low=0.2pt
	score := float32(sev.Critical*4 + sev.High*2 + sev.Medium + sev.Low/5)
	
	// Normalize to 0-10
	if score > 10 {
		return 10
	}
	return score
}

// getRiskLevel returns risk level name
func (g *Generator) getRiskLevel(score float32) string {
	switch {
	case score >= 9:
		return "critical"
	case score >= 7:
		return "high"
	case score >= 4:
		return "medium"
	case score >= 2:
		return "low"
	default:
		return "minimal"
	}
}

// buildPhases creates phase results from findings
func (g *Generator) buildPhases(findings []*Finding) []PhaseResult {
	// Group findings by tool (source)
	phaseMap := make(map[string]*PhaseResult)
	
	toolToPhase := map[string]string{
		"whois":  "Information Gathering",
		"dig":    "Information Gathering",
		"nmap":   "Configuration Analysis",
		"curl":   "Configuration Analysis",
		"nuclei": "Vulnerability Identification",
	}

	for i, f := range findings {
		phase := "Unknown"
		if p, ok := toolToPhase[f.Evidence.Tool]; ok {
			phase = p
		}

		if _, exists := phaseMap[phase]; !exists {
			phaseMap[phase] = &PhaseResult{
				Name:          phase,
				Order:         i + 1,
				Tools:         make([]string, 0),
				FindingsCount: 0,
				ToolsSucceeded: 1,
			}
		}
		
		phaseMap[phase].FindingsCount++
		if !contains(phaseMap[phase].Tools, f.Evidence.Tool) {
			phaseMap[phase].Tools = append(phaseMap[phase].Tools, f.Evidence.Tool)
		}
	}

	// Convert to ordered array
	phases := make([]PhaseResult, 0, len(phaseMap))
	order := []string{"Information Gathering", "Configuration Analysis", "Vulnerability Identification"}
	
	for _, phaseName := range order {
		if p, ok := phaseMap[phaseName]; ok {
			phases = append(phases, *p)
		}
	}

	return phases
}

// buildRemediationPlan creates remediation timeline
func (g *Generator) buildRemediationPlan(findings []*Finding) RemediationPlan {
	now := time.Now()
	
	plan := RemediationPlan{
		CriticalDeadline: now.AddDate(0, 0, 7),  // 1 week for critical
		HighDeadline:     now.AddDate(0, 0, 30), // 1 month for high
		Timeline: []TimelineItem{
			{
				Week:       1,
				Milestone:  "Patch critical vulnerabilities",
				Tasks:      []string{"Fix RCE", "Fix authentication bypass"},
				Completion: 0,
			},
			{
				Week:       2,
				Milestone:  "Address high-severity issues",
				Tasks:      []string{"Fix SQL injection", "Implement input validation"},
				Completion: 0,
			},
			{
				Week:       3,
				Milestone:  "Deploy medium-severity fixes",
				Tasks:      []string{"Add security headers", "Update configurations"},
				Completion: 0,
			},
			{
				Week:       4,
				Milestone:  "Final assessment & validation",
				Tasks:      []string{"Re-scan", "Verify fixes", "Document changes"},
				Completion: 0,
			},
		},
	}

	return plan
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
