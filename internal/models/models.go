package models

import (
	"time"
)

// ScanStatus represents the current state of a scan
type ScanStatus string

const (
	ScanStatusPending    ScanStatus = "pending"
	ScanStatusRunning    ScanStatus = "running"
	ScanStatusCompleted  ScanStatus = "completed"
	ScanStatusFailed     ScanStatus = "failed"
	ScanStatusCancelled  ScanStatus = "cancelled"
)

// PhaseType represents the phase of attack
type PhaseType string

const (
	PhaseRecon    PhaseType = "recon"
	PhaseEnum     PhaseType = "enum"
	PhaseExploit  PhaseType = "exploit"
	PhaseVerify   PhaseType = "verify"
	PhaseReport   PhaseType = "report"
)

// Scan represents a single penetration test scan
type Scan struct {
	ID        string        `json:"id"`
	Target    string        `json:"target"`
	Status    ScanStatus    `json:"status"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty"`
	Results   []ToolOutput  `json:"results"`
	Error     string        `json:"error,omitempty"`
}

// PlaybookStep represents a single step in a playbook
type PlaybookStep struct {
	ID          string            `json:"id"`
	Phase       PhaseType         `json:"phase"`
	Order       int               `json:"order"`
	ToolName    string            `json:"tool_name"`
	Args        []string          `json:"args"`
	Description string            `json:"description"`
	Condition   string            `json:"condition,omitempty"`   // e.g., "if_port_open_22"
	NextSteps   map[string]string `json:"next_steps,omitempty"` // condition -> step_id
}

// ToolOutput represents the output of a tool execution
type ToolOutput struct {
	ID        string    `json:"id"`
	ToolName  string    `json:"tool_name"`
	Target    string    `json:"target"`
	Phase     PhaseType `json:"phase"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Stdout    string    `json:"stdout"`
	Stderr    string    `json:"stderr"`
	ExitCode  int       `json:"exit_code"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Duration  int       `json:"duration_ms"`
	Success   bool      `json:"success"`
	Parsed    interface{} `json:"parsed,omitempty"` // Tool-specific parsed data
	Error     string    `json:"error,omitempty"`
}

// AttackChain represents a sequence of steps
type AttackChain struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Steps []PlaybookStep  `json:"steps"`
	Entry string          `json:"entry"` // first step ID
}

// ToolConfig represents a tool in the registry
type ToolConfig struct {
	Name        string   `json:"name"`
	BinaryPath  string   `json:"binary_path"`
	Description string   `json:"description"`
	Version     string   `json:"version,omitempty"`
	Args        []string `json:"args"`
	Timeout     int      `json:"timeout"` // seconds
	OutputType  string   `json:"output_type"` // json, text, xml
	Available   bool     `json:"available"`
}

// NmapResult represents parsed nmap JSON output
type NmapResult struct {
	Host       string       `json:"host"`
	Status     string       `json:"status"`
	Ports      []NmapPort   `json:"ports"`
	OSClass    []string     `json:"os_class"`
	OSMatch    []string     `json:"os_match"`
}

// NmapPort represents a single port from nmap
type NmapPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Product  string `json:"product"`
	Version  string `json:"version"`
}

// NucleiResult represents parsed nuclei JSON output
type NucleiResult struct {
	TemplateID   string `json:"template_id"`
	Info         string `json:"info"`
	TemplateURL  string `json:"template_url"`
	MatcherName  string `json:"matcher_name"`
	Severity     string `json:"severity"`
	Tags         string `json:"tags"`
	Description  string `json:"description"`
	Timestamp    string `json:"timestamp"`
	CurleURL     string `json:"curl_url"`
}

// LogEntry represents a log line for UI
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, warning, error, success
	Message   string    `json:"message"`
	ScanID    string    `json:"scan_id,omitempty"`
	ToolName  string    `json:"tool_name,omitempty"`
}
