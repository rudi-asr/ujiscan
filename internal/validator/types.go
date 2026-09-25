package validator

import (
	"net"
	"time"
)

// TargetType represents the classification of the target
type TargetType string

const (
	TargetTypeWeb      TargetType = "WEB"
	TargetTypeNetwork  TargetType = "NETWORK"
	TargetTypeCloud    TargetType = "CLOUD"
	TargetTypeMobile   TargetType = "MOBILE"
	TargetTypeDatabase TargetType = "DATABASE"
	TargetTypeUnknown  TargetType = "UNKNOWN"
)

// ScopeDefinition represents the scope of the engagement
type ScopeDefinition struct {
	// Include patterns (CIDR, domain wildcard, exact IPs)
	Include []string `json:"include"`
	// Exclude patterns (out-of-scope targets)
	Exclude []string `json:"exclude"`
	// Parsed includes (for efficient checking)
	IncludedNets     []*net.IPNet
	IncludedDomains  []string
	// Parsed excludes
	ExcludedNets     []*net.IPNet
	ExcludedDomains  []string
}

// TargetProfile represents the validated target information
type TargetProfile struct {
	// Unique identifier
	ID string `json:"id"`
	// Target URL or IP
	Target string `json:"target"`
	// Classified type
	Type TargetType `json:"type"`
	// Parsed targets (URLs, IPs, domains)
	Hosts []string `json:"hosts"`
	// Scope definition (include/exclude)
	Scope *ScopeDefinition `json:"scope"`
	// Is valid for execution
	IsValid bool `json:"is_valid"`
	// Validation errors/warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	// Reason if invalid
	Reason string `json:"reason,omitempty"`
	// Metadata
	CreatedAt time.Time `json:"created_at"`
	// Additional properties detected
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Profile *TargetProfile
	Error   error
	Warning string
}

// TargetClassification represents the result of target type detection
type TargetClassification struct {
	Type       TargetType
	Confidence float64 // 0.0-1.0
	Reason     string
	Indicators []string
}
