// Package registry provides tool registration and discovery
package registry

import "time"

// ExecutionConfig defines how a tool is executed
type ExecutionConfig struct {
	Command        string        `yaml:"command"`
	Args           string        `yaml:"args"`
	Timeout        int           `yaml:"timeout_seconds"`
	TimeoutBehavior string       `yaml:"timeout_behavior"` // "kill" or "timeout_graceful"
	RequireRoot    bool          `yaml:"require_root"`
	DefaultVariant string        `yaml:"default_variant,omitempty"`
}

// InstallationConfig defines tool installation
type InstallationConfig struct {
	CheckCommand   string            `yaml:"check_command"`
	CheckExitCode  int               `yaml:"check_exit_code"`
	MacOS          string            `yaml:"macos,omitempty"`
	Linux          string            `yaml:"linux,omitempty"`
	Windows        string            `yaml:"windows,omitempty"`
}

// OutputConfig defines how tool output is handled
type OutputConfig struct {
	Format             string `yaml:"format"` // "text", "json", "jsonl", "xml"
	Parser             string `yaml:"parser,omitempty"`
	StderrOnFailure    bool   `yaml:"stderr_on_failure"`
	CaptureHeaders     bool   `yaml:"capture_headers,omitempty"`
	CaptureBody        bool   `yaml:"capture_body,omitempty"`
}

// ToolVariant defines tool execution variant (quick, full, aggressive, etc)
type ToolVariant struct {
	Description string   `yaml:"description"`
	CommandArgs []string `yaml:"command_args"`
	Timeout     int      `yaml:"timeout_seconds,omitempty"`
}

// Tool represents a security tool definition
type Tool struct {
	ID              string                  `yaml:"id"`
	Name            string                  `yaml:"name"`
	Description     string                  `yaml:"description"`
	Phase           string                  `yaml:"phase"` // OWASP WSTG phase
	Category        string                  `yaml:"category"`
	Tags            []string                `yaml:"tags,omitempty"`
	
	Variants        map[string]ToolVariant  `yaml:"variants,omitempty"`
	Command         ExecutionConfig         `yaml:"command"`
	Execution       ExecutionConfig         `yaml:"execution"`
	Installation    InstallationConfig      `yaml:"installation"`
	Output          OutputConfig            `yaml:"output"`
	
	ModeApplicability []string              `yaml:"mode_applicability"` // which modes use this tool
	TargetTypes       []string              `yaml:"target_types"`       // what target types work
	
	Reference       map[string]string       `yaml:"reference,omitempty"` // owasp, mitre, etc
	Triggers        []RuleTrigger           `yaml:"triggers,omitempty"`
	
	// Runtime state
	Available       bool                    `yaml:"-"`
	InstalledAt     time.Time               `yaml:"-"`
	Version         string                  `yaml:"-"`
}

// RuleTrigger defines when this tool should be triggered
type RuleTrigger struct {
	Condition   string   `yaml:"condition"`     // finding type to trigger on
	NextPhase   string   `yaml:"next_phase"`    // which phase to jump to
	NextTools   []string `yaml:"next_tools,omitempty"`
}

// Phase represents an OWASP WSTG testing phase
type Phase struct {
	ID          string `yaml:"id"`
	Order       int    `yaml:"order"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	OWASPTag    string `yaml:"owasp_wstg"`
}

// EngagementMode defines execution mode (bug-bounty, red-team, ctf, network-recon)
type EngagementMode struct {
	ID                  string            `yaml:"id"`
	Name                string            `yaml:"name"`
	Description         string            `yaml:"description"`
	TargetDetection     []string          `yaml:"target_detection"`
	ExecutionOrder      []PhaseExecution  `yaml:"execution_order"`
	Reference           map[string]string `yaml:"reference,omitempty"`
	ParallelEnabled     bool              `yaml:"parallel_enabled,omitempty"`
}

// PhaseExecution defines how tools run in a phase
type PhaseExecution struct {
	Phase      string   `yaml:"phase"`
	Tools      []string `yaml:"tools"`
	Parallel   bool     `yaml:"parallel"`
}

// Methodology represents a testing methodology
type Methodology struct {
	Name          string   `yaml:"name"`
	OWASPID       string   `yaml:"owasp_id"`
	Description   string   `yaml:"description"`
	Tools         []string `yaml:"tools"`
	ManualSteps   []string `yaml:"manual_steps,omitempty"`
}

// ScopeRule defines scope validation patterns
type ScopeRule struct {
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description"`
	Example     string `yaml:"example"`
}

// SeverityLevel defines CVSS severity mapping
type SeverityLevel struct {
	ID          string `yaml:"id"`
	CVSSRange   string `yaml:"cvss_range"`
	Description string `yaml:"description"`
}

// EngagementType defines engagement mode (bug-bounty, red-team, etc)
type EngagementType struct {
	ID                   string `yaml:"id"`
	Name                 string `yaml:"name"`
	Description          string `yaml:"description"`
	ReportFormat         string `yaml:"report_format"`
	ScopeRequired        bool   `yaml:"scope_required"`
	VerificationRequired bool   `yaml:"verification_required"`
	CVSSScoring          bool   `yaml:"cvss_scoring"`
}

// Registry holds all tool definitions
type Registry struct {
	SchemaVersion   string                   `yaml:"schema_version"`
	Description     string                   `yaml:"description"`
	
	Phases          []Phase                  `yaml:"phases"`
	Tools           []Tool                   `yaml:"tools"`
	Modes           []EngagementMode         `yaml:"modes"`
	EngagementTypes []EngagementType         `yaml:"engagement_types"`
	Methodologies   []Methodology            `yaml:"methodologies,omitempty"`
	ScopeRules      []ScopeRule              `yaml:"scope_patterns,omitempty"`
	SeverityLevels  []SeverityLevel          `yaml:"severity_levels,omitempty"`
	Categories      []Phase                  `yaml:"categories,omitempty"`
	
	// Runtime maps for fast lookup
	ToolsByID       map[string]*Tool         `yaml:"-"`
	ToolsByPhase    map[string][]*Tool       `yaml:"-"`
	ModesByID       map[string]*EngagementMode `yaml:"-"`
	PhasesMap       map[string]*Phase        `yaml:"-"`
}

// GetTool returns a tool by ID
func (r *Registry) GetTool(id string) *Tool {
	if r.ToolsByID == nil {
		return nil
	}
	return r.ToolsByID[id]
}

// GetToolsByPhase returns all tools in a phase
func (r *Registry) GetToolsByPhase(phaseID string) []*Tool {
	if r.ToolsByPhase == nil {
		return nil
	}
	return r.ToolsByPhase[phaseID]
}

// GetMode returns an engagement mode by ID
func (r *Registry) GetMode(id string) *EngagementMode {
	if r.ModesByID == nil {
		return nil
	}
	return r.ModesByID[id]
}

// GetPhase returns a phase by ID
func (r *Registry) GetPhase(id string) *Phase {
	if r.PhasesMap == nil {
		return nil
	}
	return r.PhasesMap[id]
}

// BuildIndices builds runtime lookup maps
func (r *Registry) BuildIndices() {
	r.ToolsByID = make(map[string]*Tool)
	r.ToolsByPhase = make(map[string][]*Tool)
	r.ModesByID = make(map[string]*EngagementMode)
	r.PhasesMap = make(map[string]*Phase)

	// Index tools
	for i := range r.Tools {
		tool := &r.Tools[i]
		r.ToolsByID[tool.ID] = tool
		if tool.Phase != "" {
			r.ToolsByPhase[tool.Phase] = append(r.ToolsByPhase[tool.Phase], tool)
		}
	}

	// Index modes
	for i := range r.Modes {
		mode := &r.Modes[i]
		r.ModesByID[mode.ID] = mode
	}

	// Index phases
	for i := range r.Phases {
		phase := &r.Phases[i]
		r.PhasesMap[phase.ID] = phase
	}
}
