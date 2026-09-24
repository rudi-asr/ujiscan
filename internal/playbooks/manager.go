// Package playbooks provides playbook management
package playbooks

import (
	"fmt"
	"log"
	"time"
)

// Playbook represents a saved execution sequence
type Playbook struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Mode        string        `json:"mode"` // reconnaissance, hunt, red-team, network-scan
	ScopeRules  []ScopeRule   `json:"scope_rules"`
	Phases      []PhaseConfig `json:"phases"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	IsTemplate  bool          `json:"is_template"` // reusable template
	Tags        []string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ScopeRule defines scope for playbook
type ScopeRule struct {
	Pattern string `json:"pattern"` // domain, CIDR, etc
	Type    string `json:"type"`    // allow, deny
}

// PhaseConfig defines tools to run in a phase
type PhaseConfig struct {
	PhaseID   string        `json:"phase_id"`
	Tools     []ToolExecConfig `json:"tools"`
	Parallel  bool          `json:"parallel"`
	Order     int           `json:"order"`
}

// ToolExecConfig defines how to execute a tool
type ToolExecConfig struct {
	ToolID    string   `json:"tool_id"`
	Variant   string   `json:"variant,omitempty"` // quick, full, aggressive
	Args      []string `json:"args,omitempty"`
	Condition string   `json:"condition,omitempty"` // run if previous finding type
}

// PlaybookExecution represents a playbook run
type PlaybookExecution struct {
	ID           string             `json:"id"`
	PlaybookID   string             `json:"playbook_id"`
	Target       string             `json:"target"`
	Status       string             `json:"status"` // pending, running, completed, failed
	StartedAt    time.Time          `json:"started_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	Duration     int                `json:"duration_ms"`
	ToolResults  []ToolExecution    `json:"tool_results"`
	Findings     []interface{}      `json:"findings"` // from finding service
	Score        float32            `json:"score,omitempty"` // overall risk score
	Error        string             `json:"error,omitempty"`
}

// ToolExecution represents a single tool run
type ToolExecution struct {
	ToolID    string      `json:"tool_id"`
	Status    string      `json:"status"`
	StartedAt time.Time   `json:"started_at"`
	EndedAt   time.Time   `json:"ended_at"`
	Duration  int         `json:"duration_ms"`
	Output    string      `json:"output"`
	Error     string      `json:"error,omitempty"`
	Findings  int         `json:"findings_count"`
}

// PlaybookManager manages playbooks
type PlaybookManager struct {
	playbooks  map[string]*Playbook
	executions map[string]*PlaybookExecution
	templates  map[string]*Playbook
}

// NewPlaybookManager creates manager
func NewPlaybookManager() *PlaybookManager {
	return &PlaybookManager{
		playbooks:  make(map[string]*Playbook),
		executions: make(map[string]*PlaybookExecution),
		templates:  make(map[string]*Playbook),
	}
}

// CreatePlaybook creates a new playbook from execution
func (pm *PlaybookManager) CreatePlaybook(
	name string,
	description string,
	mode string,
	scope []ScopeRule,
	phases []PhaseConfig,
) *Playbook {
	pb := &Playbook{
		ID:          fmt.Sprintf("pb_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Mode:        mode,
		ScopeRules:  scope,
		Phases:      phases,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsTemplate:  false,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}

	pm.playbooks[pb.ID] = pb
	log.Printf("✅ Playbook created: %s (%s)", pb.Name, pb.ID)
	return pb
}

// CreateTemplate creates a reusable template
func (pm *PlaybookManager) CreateTemplate(
	name string,
	description string,
	mode string,
	phases []PhaseConfig,
) *Playbook {
	pb := &Playbook{
		ID:          fmt.Sprintf("tpl_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Mode:        mode,
		Phases:      phases,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsTemplate:  true,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}

	pm.templates[pb.ID] = pb
	log.Printf("✅ Template created: %s (%s)", pb.Name, pb.ID)
	return pb
}

// GetPlaybook retrieves a playbook
func (pm *PlaybookManager) GetPlaybook(id string) *Playbook {
	return pm.playbooks[id]
}

// GetTemplate retrieves a template
func (pm *PlaybookManager) GetTemplate(id string) *Playbook {
	return pm.templates[id]
}

// ListPlaybooks returns all playbooks
func (pm *PlaybookManager) ListPlaybooks() []*Playbook {
	result := make([]*Playbook, 0, len(pm.playbooks))
	for _, pb := range pm.playbooks {
		result = append(result, pb)
	}
	return result
}

// ListTemplates returns all templates
func (pm *PlaybookManager) ListTemplates() []*Playbook {
	result := make([]*Playbook, 0, len(pm.templates))
	for _, tpl := range pm.templates {
		result = append(result, tpl)
	}
	return result
}

// RecordExecution records a playbook execution
func (pm *PlaybookManager) RecordExecution(exec *PlaybookExecution) {
	pm.executions[exec.ID] = exec
	log.Printf("✅ Execution recorded: %s", exec.ID)
}

// GetExecution retrieves an execution
func (pm *PlaybookManager) GetExecution(id string) *PlaybookExecution {
	return pm.executions[id]
}

// GetExecutionHistory returns execution history for a playbook
func (pm *PlaybookManager) GetExecutionHistory(playbookID string) []*PlaybookExecution {
	result := make([]*PlaybookExecution, 0)
	for _, exec := range pm.executions {
		if exec.PlaybookID == playbookID {
			result = append(result, exec)
		}
	}
	return result
}

// CloneFromTemplate creates playbook from template
func (pm *PlaybookManager) CloneFromTemplate(templateID string, newName string) *Playbook {
	template := pm.GetTemplate(templateID)
	if template == nil {
		return nil
	}

	pb := &Playbook{
		ID:          fmt.Sprintf("pb_%d", time.Now().UnixNano()),
		Name:        newName,
		Description: template.Description,
		Mode:        template.Mode,
		Phases:      template.Phases,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsTemplate:  false,
		Tags:        template.Tags,
		Metadata:    make(map[string]interface{}),
	}

	pm.playbooks[pb.ID] = pb
	log.Printf("✅ Playbook cloned from template: %s → %s", templateID, pb.ID)
	return pb
}

// DeletePlaybook removes a playbook
func (pm *PlaybookManager) DeletePlaybook(id string) error {
	if pm.playbooks[id] == nil {
		return fmt.Errorf("playbook not found: %s", id)
	}
	delete(pm.playbooks, id)
	return nil
}

// UpdatePlaybook updates a playbook
func (pm *PlaybookManager) UpdatePlaybook(id string, name string, description string) error {
	pb := pm.playbooks[id]
	if pb == nil {
		return fmt.Errorf("playbook not found: %s", id)
	}

	pb.Name = name
	pb.Description = description
	pb.UpdatedAt = time.Now()
	return nil
}

// Statistics returns usage statistics
type Statistics struct {
	TotalPlaybooks  int
	TotalTemplates  int
	TotalExecutions int
	AvgDuration     float32
	SuccessRate     float32
}

// GetStatistics returns playbook statistics
func (pm *PlaybookManager) GetStatistics() Statistics {
	stats := Statistics{
		TotalPlaybooks:  len(pm.playbooks),
		TotalTemplates:  len(pm.templates),
		TotalExecutions: len(pm.executions),
	}

	var totalDuration int
	var successCount int

	for _, exec := range pm.executions {
		totalDuration += exec.Duration
		if exec.Status == "completed" {
			successCount++
		}
	}

	if len(pm.executions) > 0 {
		stats.AvgDuration = float32(totalDuration) / float32(len(pm.executions))
		stats.SuccessRate = float32(successCount) * 100 / float32(len(pm.executions))
	}

	return stats
}

// DefaultTemplates returns built-in templates
func (pm *PlaybookManager) LoadDefaultTemplates() {
	// Reconnaissance template
	reconTemplate := &Playbook{
		ID:          "tpl_recon",
		Name:        "Quick Reconnaissance",
		Description: "Fast passive information gathering",
		Mode:        "reconnaissance",
		IsTemplate:  true,
		Phases: []PhaseConfig{
			{
				PhaseID:  "information_gathering",
				Tools:    []ToolExecConfig{{ToolID: "whois"}, {ToolID: "dig"}},
				Parallel: true,
				Order:    1,
			},
			{
				PhaseID:  "configuration_analysis",
				Tools:    []ToolExecConfig{{ToolID: "nmap", Variant: "quick"}},
				Parallel: false,
				Order:    2,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []string{"fast", "passive", "recon"},
	}
	pm.templates[reconTemplate.ID] = reconTemplate

	// Hunt template
	huntTemplate := &Playbook{
		ID:          "tpl_hunt",
		Name:        "Vulnerability Hunt",
		Description: "Active vulnerability scanning workflow",
		Mode:        "hunt",
		IsTemplate:  true,
		Phases: []PhaseConfig{
			{
				PhaseID:  "information_gathering",
				Tools:    []ToolExecConfig{{ToolID: "whois"}, {ToolID: "dig"}},
				Parallel: true,
				Order:    1,
			},
			{
				PhaseID:  "configuration_analysis",
				Tools:    []ToolExecConfig{{ToolID: "nmap", Variant: "full"}, {ToolID: "curl"}},
				Parallel: true,
				Order:    2,
			},
			{
				PhaseID:  "identification",
				Tools:    []ToolExecConfig{{ToolID: "nuclei", Variant: "full"}},
				Parallel: false,
				Order:    3,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []string{"vuln-scan", "aggressive", "hunt"},
	}
	pm.templates[huntTemplate.ID] = huntTemplate

	// Red Team template
	redTeamTemplate := &Playbook{
		ID:          "tpl_redteam",
		Name:        "Red Team Assessment",
		Description: "Full adversarial evaluation workflow",
		Mode:        "red-team",
		IsTemplate:  true,
		Phases: []PhaseConfig{
			{
				PhaseID:  "information_gathering",
				Tools:    []ToolExecConfig{{ToolID: "whois"}, {ToolID: "dig"}},
				Parallel: true,
				Order:    1,
			},
			{
				PhaseID:  "configuration_analysis",
				Tools:    []ToolExecConfig{{ToolID: "nmap", Variant: "aggressive"}, {ToolID: "curl"}},
				Parallel: false,
				Order:    2,
			},
			{
				PhaseID:  "identification",
				Tools:    []ToolExecConfig{{ToolID: "nuclei", Variant: "full"}},
				Parallel: false,
				Order:    3,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []string{"red-team", "full", "aggressive"},
	}
	pm.templates[redTeamTemplate.ID] = redTeamTemplate

	log.Printf("✅ Loaded %d default templates", len(pm.templates))
}
