package planner

import (
	"time"
)

// EngagementMode represents the type of penetration testing engagement
type EngagementMode string

const (
	ModeWebStandard    EngagementMode = "WEB_STANDARD"
	ModeNetworkDeep    EngagementMode = "NETWORK_DEEP"
	ModeCloudInfra     EngagementMode = "CLOUD_INFRASTRUCTURE"
	ModeAggressiveTest EngagementMode = "AGGRESSIVE_TEST"
	ModeQuickScan      EngagementMode = "QUICK_SCAN"
)

// ExecutionPhase represents a phase in the execution plan
type ExecutionPhase string

const (
	PhaseRecon     ExecutionPhase = "RECONNAISSANCE"
	PhaseScanning  ExecutionPhase = "SCANNING"
	PhaseAnalysis  ExecutionPhase = "ANALYSIS"
	PhaseReporting ExecutionPhase = "REPORTING"
)

// ExecutionStep represents a single step in a workflow
type ExecutionStep struct {
	// Unique ID
	ID string `json:"id"`
	// Step name
	Name string `json:"name"`
	// Which agent executes this (recon, scanner, analyzer)
	Agent string `json:"agent"`
	// Which tool to use
	Tool string `json:"tool"`
	// Tool parameters
	Parameters map[string]interface{} `json:"parameters"`
	// Timeout (seconds)
	Timeout int `json:"timeout"`
	// Retry count if fails
	MaxRetries int `json:"max_retries"`
	// Can run in parallel with other steps (optional)
	Parallel bool `json:"parallel"`
}

// ExecutionPlan represents the full plan for an engagement
type ExecutionPlan struct {
	// Unique ID
	ID string `json:"id"`
	// Engagement target
	Target string `json:"target"`
	// Detected mode
	Mode EngagementMode `json:"mode"`
	// Steps grouped by phase
	Phases map[ExecutionPhase][]ExecutionStep `json:"phases"`
	// Total estimated time (minutes)
	EstimatedDuration int `json:"estimated_duration"`
	// Tools that will be used
	Tools []string `json:"tools"`
	// Metadata
	CreatedAt time.Time `json:"created_at"`
	// Version of the plan
	Version string `json:"version"`
}

// ModeConfig represents configuration for an engagement mode
type ModeConfig struct {
	// Mode name
	Name EngagementMode `json:"name"`
	// Description
	Description string `json:"description"`
	// Tools to use (in priority order)
	Tools []string `json:"tools"`
	// Phases to execute (in order)
	Phases []ExecutionPhase `json:"phases"`
	// Estimated duration (minutes)
	Duration int `json:"duration"`
	// When to use this mode
	Indicators []string `json:"indicators"`
}

// ModeDetectionResult represents the result of mode detection
type ModeDetectionResult struct {
	Mode       EngagementMode
	Confidence float64
	Reason     string
	Indicators []string
}
