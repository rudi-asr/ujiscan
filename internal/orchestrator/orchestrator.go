// Package orchestrator coordinates execution flow with rules and scope
package orchestrator

import (
	"fmt"
	"log"

	"github.com/rudi-asr/ujiscan/internal/registry"
	"github.com/rudi-asr/ujiscan/internal/rules"
	"github.com/rudi-asr/ujiscan/internal/scope"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

// Orchestrator coordinates tool execution with rules and scope validation
type Orchestrator struct {
	registry       *registry.Registry
	executor       *tools.RegistryExecutor
	rulesEngine    *rules.RulesEngine
	scopeValidator *scope.ScopeValidator
	findings       []*rules.Finding // accumulated findings
	executionLog   []ExecutionStep
	strict         bool
}

// ExecutionStep records what was executed
type ExecutionStep struct {
	Phase       string
	ToolID      string
	Target      string
	Status      string // pending, running, completed, failed
	Reason      string // why this tool ran
	TriggerRule string // which rule triggered this
	Output      interface{}
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(
	reg *registry.Registry,
	exec *tools.RegistryExecutor,
	strict bool,
) *Orchestrator {
	orch := &Orchestrator{
		registry:       reg,
		executor:       exec,
		rulesEngine:    rules.NewRulesEngine(reg),
		scopeValidator: scope.NewScopeValidator(strict),
		findings:       make([]*rules.Finding, 0),
		executionLog:   make([]ExecutionStep, 0),
		strict:         strict,
	}

	// Load default rules
	orch.rulesEngine.LoadDefaultRules()

	return orch
}

// AddScopePattern adds a domain or CIDR to allowed scope
func (o *Orchestrator) AddScopePattern(pattern string) error {
	return o.scopeValidator.AddPattern(pattern)
}

// AddDenyPattern adds a pattern to deny list
func (o *Orchestrator) AddDenyPattern(pattern string) error {
	return o.scopeValidator.DenyPattern(pattern)
}

// AddRule adds a custom rule
func (o *Orchestrator) AddRule(rule *rules.Rule) {
	o.rulesEngine.AddRule(rule)
}

// ValidateTarget checks if target is in scope
func (o *Orchestrator) ValidateTarget(target string) *scope.ValidationResult {
	return o.scopeValidator.ValidateTarget(target)
}

// ValidateAndExecuteMode validates target then executes mode
func (o *Orchestrator) ValidateAndExecuteMode(
	target string,
	modeID string,
) error {
	// Validate scope first
	validation := o.ValidateTarget(target)
	if !validation.Valid {
		return fmt.Errorf("scope validation failed: %s", validation.Reason)
	}

	log.Printf("✅ Target %s is in scope", target)

	// Execute mode
	mode := o.registry.GetMode(modeID)
	if mode == nil {
		return fmt.Errorf("mode %s not found", modeID)
	}

	// Execute phase by phase
	for _, phaseExec := range mode.ExecutionOrder {
		o.executePhaseWithRules(phaseExec.Phase, modeID, target, phaseExec.Parallel)
	}

	return nil
}

// executePhaseWithRules executes a phase and evaluates rules on findings
func (o *Orchestrator) executePhaseWithRules(
	phaseID string,
	modeID string,
	target string,
	parallel bool,
) error {
	log.Printf("Executing phase: %s", phaseID)

	// Get tools for phase
	tools := o.registry.GetToolsByPhase(phaseID)
	if len(tools) == 0 {
		return fmt.Errorf("no tools found for phase %s", phaseID)
	}

	// Execute tools
	for _, tool := range tools {
		// Check if tool is applicable to this mode
		applicable := false
		for _, mode := range tool.ModeApplicability {
			if mode == modeID {
				applicable = true
				break
			}
		}
		if !applicable {
			log.Printf("  Tool %s not applicable to mode %s, skipping", tool.Name, modeID)
			continue
		}

		// Execute tool
		o.logExecution(phaseID, tool.ID, target, "pending", "Mode execution", "")

		output, err := o.executor.RunTool(tool.ID, []string{target}, target, "")
		if err != nil {
			o.logExecution(phaseID, tool.ID, target, "failed", err.Error(), "")
			continue
		}

		o.logExecution(phaseID, tool.ID, target, "completed", "Tool executed", "")

		// Parse findings from output (simplified - just create generic finding)
		finding := &rules.Finding{
			ID:        output.ID,
			Type:      fmt.Sprintf("tool_output_%s", tool.ID),
			Source:    tool.Name,
			Value:     fmt.Sprintf("%d bytes", len(output.Stdout)),
			Data:      output,
			Timestamp: output.StartedAt.Unix(),
		}
		o.findings = append(o.findings, finding)

		// Evaluate rules against new finding
		evaluation := o.rulesEngine.EvaluateFinding(finding)
		if len(evaluation.MatchedRules) > 0 {
			log.Printf("⚡ %d rules matched for finding from %s", 
				len(evaluation.MatchedRules), tool.Name)

			// Execute recommended tools
			for _, nextToolID := range evaluation.NextTools {
				nextTool := o.registry.GetTool(nextToolID)
				if nextTool == nil {
					continue
				}

				log.Printf("  → Running %s (recommended by rules)", nextTool.Name)
				o.logExecution(phaseID, nextToolID, target, "pending", 
					"Rule-triggered execution", evaluation.MatchedRules[0].ID)

				_, err := o.executor.RunTool(nextToolID, []string{target}, target, "")
				if err != nil {
					o.logExecution(phaseID, nextToolID, target, "failed", 
						err.Error(), evaluation.MatchedRules[0].ID)
					continue
				}

				o.logExecution(phaseID, nextToolID, target, "completed", 
					"Rule-triggered tool executed", evaluation.MatchedRules[0].ID)
			}
		}
	}

	return nil
}

// logExecution records an execution step
func (o *Orchestrator) logExecution(
	phase string,
	toolID string,
	target string,
	status string,
	reason string,
	triggerRule string,
) {
	step := ExecutionStep{
		Phase:       phase,
		ToolID:      toolID,
		Target:      target,
		Status:      status,
		Reason:      reason,
		TriggerRule: triggerRule,
	}
	o.executionLog = append(o.executionLog, step)
}

// GetFindings returns accumulated findings
func (o *Orchestrator) GetFindings() []*rules.Finding {
	return o.findings
}

// GetExecutionLog returns execution history
func (o *Orchestrator) GetExecutionLog() []ExecutionStep {
	return o.executionLog
}

// ScopeConfig returns current scope configuration
func (o *Orchestrator) ScopeConfig() string {
	return o.scopeValidator.Summary()
}

// Summary returns execution summary
func (o *Orchestrator) Summary() string {
	return fmt.Sprintf(
		"Orchestration Summary:\n"+
			"  Findings: %d\n"+
			"  Execution steps: %d\n"+
			"  Rules triggered: %d\n"+
			"  Scope validation: strict=%v",
		len(o.findings),
		len(o.executionLog),
		len(o.executionLog), // TODO: track actual rule triggers
		o.strict,
	)
}
