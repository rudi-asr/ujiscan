package report

import "time"

// Finding represents a security finding
type Finding struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Target          string    `json:"target"`
	Severity        int       `json:"severity"`    // 1-10
	CVSS            float64   `json:"cvss"`       // 0.0-10.0
	Evidence        string    `json:"evidence"`
	Remediation     string    `json:"remediation"`
	References      []string  `json:"references"`
	DetectionTime   time.Time `json:"detection_time"`
	ToolName        string    `json:"tool_name"`
	Status          string    `json:"status"`     // open, closed, false_positive
	Confidence      int       `json:"confidence"` // 0-100
}

// Report represents the complete pentest report
type Report struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Target          string     `json:"target"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	ExecutionTime   string     `json:"execution_time"`
	Findings        []Finding  `json:"findings"`
	Summary         Summary    `json:"summary"`
	Methodology     string     `json:"methodology"`
	Scope           string     `json:"scope"`
	Disclaimer      string     `json:"disclaimer"`
	ExternalRefs    []string   `json:"external_references"`
}

// Summary contains aggregated report statistics
type Summary struct {
	TotalFindings     int    `json:"total_findings"`
	CriticalCount     int    `json:"critical_count"`
	HighCount         int    `json:"high_count"`
	MediumCount       int    `json:"medium_count"`
	LowCount          int    `json:"low_count"`
	RiskScore         int    `json:"risk_score"`        // 0-100
	ComplianceStatus  string `json:"compliance_status"` // compliant, non_compliant, partial
	ToolsExecuted     []string `json:"tools_executed"`
	ExecutionSuccess  bool   `json:"execution_success"`
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatPDF  ExportFormat = "pdf"
	FormatHTML ExportFormat = "html"
	FormatCSV  ExportFormat = "csv"
)

// ReportConfig contains report generation configuration
type ReportConfig struct {
	IncludeSummary    bool
	IncludeReferences bool
	IncludeFalsePositive bool
	GroupByType       bool
	SortBySeverity    bool
	MinimumSeverity   int
}

// DeduplicationConfig configures finding deduplication
type DeduplicationConfig struct {
	SimilarityThreshold float64
	MatchType           bool
	MatchTarget         bool
	MatchEvidence       bool
}
