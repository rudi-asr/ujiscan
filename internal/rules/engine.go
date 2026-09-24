// Package rules provides rule engine for automated tool selection
package rules

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/registry"
)

// RulesEngine evaluates findings and determines next tools to run
type RulesEngine struct {
	registry *registry.Registry
	rules    []*Rule
}

// Rule defines when and which tools to run
type Rule struct {
	ID          string   // unique rule ID
	Name        string   // human readable
	Condition   Condition
	Actions     []Action
	Priority    int // higher = runs first
	Description string
}

// Condition defines when a rule matches
type Condition struct {
	FindingType  string   // finding type to match (e.g., "open_port", "http_server", "dns_record")
	ValuePattern string   // regex pattern to match against finding value
	Operator     string   // "equals", "contains", "regex", "greater_than"
	Value        string   // comparison value
	AllOf        []*Condition
	AnyOf        []*Condition
}

// Action defines what to do when rule matches
type Action struct {
	Type       string   // "run_tool", "run_phase", "set_scope", "report"
	ToolID     string   // for run_tool
	PhaseID    string   // for run_phase
	Variant    string   // tool variant (quick, full, aggressive)
	Parallel   bool     // run in parallel
	Arguments  []string // extra args to pass
	Description string
}

// Finding represents a detected issue or info
type Finding struct {
	ID       string      // unique ID
	Type     string      // "open_port", "http_server", "dns_record", "vulnerability", etc
	Severity string      // "info", "low", "medium", "high", "critical"
	Value    string      // the actual finding (port 22, service ssh, etc)
	Source   string      // which tool found this
	Data     interface{} // tool-specific data
	Timestamp int64
}

// RuleEvaluation result
type RuleEvaluation struct {
	MatchedRules []*Rule
	Actions      []Action
	NextTools    []string // tool IDs to run
	NextPhases   []string // phase IDs to run
	Recommendations []string
}

// NewRulesEngine creates a new rules engine
func NewRulesEngine(reg *registry.Registry) *RulesEngine {
	return &RulesEngine{
		registry: reg,
		rules:    make([]*Rule, 0),
	}
}

// AddRule adds a rule to the engine
func (re *RulesEngine) AddRule(rule *Rule) {
	re.rules = append(re.rules, rule)
	// Sort by priority (higher first)
	for i := len(re.rules) - 1; i > 0; i-- {
		if re.rules[i].Priority > re.rules[i-1].Priority {
			re.rules[i], re.rules[i-1] = re.rules[i-1], re.rules[i]
		}
	}
}

// EvaluateFinding evaluates a finding against all rules
func (re *RulesEngine) EvaluateFinding(finding *Finding) *RuleEvaluation {
	result := &RuleEvaluation{
		MatchedRules: make([]*Rule, 0),
		Actions:      make([]Action, 0),
		NextTools:    make([]string, 0),
		NextPhases:   make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Evaluate each rule
	for _, rule := range re.rules {
		if re.matchesCondition(rule.Condition, finding) {
			result.MatchedRules = append(result.MatchedRules, rule)
			result.Actions = append(result.Actions, rule.Actions...)

			// Extract next tools/phases
			for _, action := range rule.Actions {
				if action.Type == "run_tool" {
					result.NextTools = append(result.NextTools, action.ToolID)
				} else if action.Type == "run_phase" {
					result.NextPhases = append(result.NextPhases, action.PhaseID)
				}
			}

			result.Recommendations = append(result.Recommendations, 
				fmt.Sprintf("Rule '%s': %s", rule.Name, rule.Description))
		}
	}

	return result
}

// matchesCondition checks if a finding matches a condition
func (re *RulesEngine) matchesCondition(cond Condition, finding *Finding) bool {
	// Handle composite conditions
	if len(cond.AllOf) > 0 {
		for _, c := range cond.AllOf {
			if !re.matchesCondition(*c, finding) {
				return false
			}
		}
		return true
	}

	if len(cond.AnyOf) > 0 {
		for _, c := range cond.AnyOf {
			if re.matchesCondition(*c, finding) {
				return true
			}
		}
		return false
	}

	// Single condition match
	if cond.FindingType != "" && cond.FindingType != finding.Type {
		return false
	}

	// Apply operator
	switch cond.Operator {
	case "equals":
		return finding.Value == cond.Value
	case "contains":
		return strings.Contains(finding.Value, cond.Value)
	case "regex":
		if re, err := regexp.Compile(cond.ValuePattern); err == nil {
			return re.MatchString(finding.Value)
		}
		return false
	case "greater_than":
		// For numeric comparisons
		return finding.Value > cond.Value
	default:
		// Default: just check finding type
		return cond.FindingType != "" && cond.FindingType == finding.Type
	}
}

// DefaultRules returns built-in rules for common scenarios
func (re *RulesEngine) DefaultRules() []*Rule {
	return []*Rule{
		// If HTTP server found, run nuclei
		{
			ID:       "http_found_scan_vulns",
			Name:     "HTTP Server Detected → Scan Vulnerabilities",
			Priority: 100,
			Description: "When HTTP server is detected, run nuclei to check for vulnerabilities",
			Condition: Condition{
				FindingType: "http_server",
			},
			Actions: []Action{
				{
					Type:    "run_tool",
					ToolID:  "nuclei",
					Variant: "quick",
					Description: "Run quick nuclei scan for common vulns",
				},
			},
		},
		// If SSH open, check for weak keys
		{
			ID:       "ssh_open_enum",
			Name:     "SSH Service Detected → Enumerate Version",
			Priority: 90,
			Description: "When SSH is found open, grab banner to identify version",
			Condition: Condition{
				FindingType: "open_port",
				ValuePattern: "22/(tcp|ssh)",
			},
			Actions: []Action{
				{
					Type:    "run_tool",
					ToolID:  "curl",
					Variant: "",
					Arguments: []string{"-v", "ssh://target:22"},
					Description: "Banner grab for SSH version detection",
				},
			},
		},
		// If DNS records found, enumerate subdomains
		{
			ID:       "dns_found_enum_subs",
			Name:     "DNS Records Found → Enumerate Subdomains",
			Priority: 85,
			Description: "When DNS resolution succeeds, look for subdomains",
			Condition: Condition{
				FindingType: "dns_record",
			},
			Actions: []Action{
				{
					Type:    "run_phase",
					PhaseID: "information_gathering",
					Description: "Run full information gathering phase",
				},
			},
		},
		// If many ports open, do OS detection
		{
			ID:       "many_ports_detect_os",
			Name:     "Multiple Ports Open → Detect OS",
			Priority: 80,
			Description: "When many ports are open, run aggressive nmap for OS detection",
			Condition: Condition{
				AnyOf: []*Condition{
					{FindingType: "open_port"},
					{FindingType: "open_port"},
					{FindingType: "open_port"},
				},
			},
			Actions: []Action{
				{
					Type:    "run_tool",
					ToolID:  "nmap",
					Variant: "aggressive",
					Description: "Run aggressive nmap scan for OS detection",
				},
			},
		},
	}
}

// LoadDefaultRules adds default rules to engine
func (re *RulesEngine) LoadDefaultRules() {
	for _, rule := range re.DefaultRules() {
		re.AddRule(rule)
	}
	log.Printf("Loaded %d default rules", len(re.DefaultRules()))
}

// EvaluateFindings evaluates multiple findings and returns consolidated actions
func (re *RulesEngine) EvaluateFindings(findings []*Finding) *RuleEvaluation {
	consolidated := &RuleEvaluation{
		MatchedRules: make([]*Rule, 0),
		Actions:      make([]Action, 0),
		NextTools:    make([]string, 0),
		NextPhases:   make([]string, 0),
		Recommendations: make([]string, 0),
	}

	seenRules := make(map[string]bool)
	seenTools := make(map[string]bool)
	seenPhases := make(map[string]bool)

	for _, finding := range findings {
		eval := re.EvaluateFinding(finding)

		// Deduplicate rules
		for _, rule := range eval.MatchedRules {
			if !seenRules[rule.ID] {
				consolidated.MatchedRules = append(consolidated.MatchedRules, rule)
				seenRules[rule.ID] = true
			}
		}

		// Deduplicate actions
		consolidated.Actions = append(consolidated.Actions, eval.Actions...)

		// Deduplicate tools
		for _, tool := range eval.NextTools {
			if !seenTools[tool] {
				consolidated.NextTools = append(consolidated.NextTools, tool)
				seenTools[tool] = true
			}
		}

		// Deduplicate phases
		for _, phase := range eval.NextPhases {
			if !seenPhases[phase] {
				consolidated.NextPhases = append(consolidated.NextPhases, phase)
				seenPhases[phase] = true
			}
		}

		consolidated.Recommendations = append(consolidated.Recommendations, eval.Recommendations...)
	}

	return consolidated
}

// String returns human-readable evaluation result
func (re *RuleEvaluation) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Matched %d rules\n", len(re.MatchedRules)))
	sb.WriteString(fmt.Sprintf("Next tools: %v\n", re.NextTools))
	sb.WriteString(fmt.Sprintf("Next phases: %v\n", re.NextPhases))
	sb.WriteString(fmt.Sprintf("Recommendations:\n"))
	for _, rec := range re.Recommendations {
		sb.WriteString(fmt.Sprintf("  • %s\n", rec))
	}
	return sb.String()
}
