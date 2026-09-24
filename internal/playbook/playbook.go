package playbook

import (
	"github.com/rudi-asr/ujiscan/internal/models"
)

// PhaseType represents a phase in the playbook
type PhaseType string

const (
	PhaseRecon   PhaseType = "recon"
	PhaseEnum    PhaseType = "enum"
	PhaseExploit PhaseType = "exploit"
	PhaseVerify  PhaseType = "verify"
	PhaseReport  PhaseType = "report"
)

// Step represents a single step in a playbook phase
type Step struct {
	ID          string            `yaml:"id"`
	Tool        string            `yaml:"tool"`
	Args        []string          `yaml:"args"`
	Description string            `yaml:"description,omitempty"`
	Condition   string            `yaml:"condition,omitempty"`
	NextSteps   map[string]string `yaml:"next_steps,omitempty"` // condition -> step_id
}

// Playbook represents a complete penetration test playbook
type Playbook struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description,omitempty"`
	Author      string                    `yaml:"author,omitempty"`
	Version     string                    `yaml:"version,omitempty"`
	EntryPhase  PhaseType                 `yaml:"entry_phase"`
	Phases      map[PhaseType][]Step      `yaml:"-"` // populated by loader
	StepMap     map[string]*Step          `yaml:"-"` // step_id -> step
}

// ExecutionContext holds execution state during playbook run
type ExecutionContext struct {
	PlaybookName string
	Target       string
	ScanID       string
	Results      []models.ToolOutput           // all tool outputs
	Conditions   map[string]interface{}        // evaluated conditions for branching
	CurrentPhase PhaseType
	StepResults  map[string]*models.ToolOutput // step_id -> output
	
	// AI-specific fields for agentic loop
	IsAgentic    bool                      // true if using AI decision-making
	AIObjective  string                    // what the scan is trying to achieve
	AIDecisions  map[string]string         // step_id -> AI decision
	AIReasonings map[string]string         // step_id -> AI reasoning
}

// NewPlaybook creates a new playbook
func NewPlaybook(name string) *Playbook {
	return &Playbook{
		Name:    name,
		Phases:  make(map[PhaseType][]Step),
		StepMap: make(map[string]*Step),
	}
}

// AddStep adds a step to a phase
func (p *Playbook) AddStep(phase PhaseType, step Step) {
	if _, ok := p.Phases[phase]; !ok {
		p.Phases[phase] = make([]Step, 0)
	}
	p.Phases[phase] = append(p.Phases[phase], step)
	p.StepMap[step.ID] = &step
}

// GetStep retrieves a step by ID
func (p *Playbook) GetStep(stepID string) *Step {
	return p.StepMap[stepID]
}

// GetPhaseSteps returns all steps for a phase
func (p *Playbook) GetPhaseSteps(phase PhaseType) []Step {
	if steps, ok := p.Phases[phase]; ok {
		return steps
	}
	return []Step{}
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(playbookName, target, scanID string) *ExecutionContext {
	return &ExecutionContext{
		PlaybookName: playbookName,
		Target:       target,
		ScanID:       scanID,
		Results:      make([]models.ToolOutput, 0),
		Conditions:   make(map[string]interface{}),
		StepResults:  make(map[string]*models.ToolOutput),
	}
}

// AddResult adds a tool output to the context
func (ec *ExecutionContext) AddResult(output *models.ToolOutput) {
	ec.Results = append(ec.Results, *output)
}

// AddStepResult stores result for a specific step
func (ec *ExecutionContext) AddStepResult(stepID string, output *models.ToolOutput) {
	ec.StepResults[stepID] = output
	ec.AddResult(output)
}

// SetCondition sets a condition for branching logic
func (ec *ExecutionContext) SetCondition(key string, value interface{}) {
	ec.Conditions[key] = value
}

// GetCondition retrieves a condition value
func (ec *ExecutionContext) GetCondition(key string) (interface{}, bool) {
	val, ok := ec.Conditions[key]
	return val, ok
}
