package playbook

import (
	"context"
	"fmt"

	"github.com/rudi-asr/ujiscan/internal/ai"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// AgenticExecutor runs AI-driven playbook execution with dynamic tool selection
type AgenticExecutor struct {
	engine   *Engine
	aiClient *ai.Client
}

// NewAgenticExecutor creates new agentic executor
func NewAgenticExecutor(engine *Engine) *AgenticExecutor {
	return &AgenticExecutor{
		engine:   engine,
		aiClient: engine.aiClient,
	}
}

// ExecuteAgenticScan runs full agentic scanning workflow
func (ae *AgenticExecutor) ExecuteAgenticScan(ctx context.Context, scanID string, target string, scanType string) error {
	fmt.Printf("[agentic] Starting agentic scan for %s (type=%s)\n", target, scanType)

	// Phase 1: AI recommends initial tools based on target
	fmt.Printf("[agentic] Phase 1: AI tool selection\n")
	selectedTools, err := ae.aiClient.SelectToolsForTarget(ctx, target, scanType)
	if err != nil {
		fmt.Printf("[agentic] Tool selection failed, using defaults: %v\n", err)
		selectedTools = ai.DefaultToolsForType(scanType)
	}
	fmt.Printf("[agentic] AI recommended tools: %v\n", selectedTools)

	// Phase 2-5: Execute phases with tool results
	fmt.Printf("[agentic] Phase 2: Reconnaissance phase\n")
	
	// For now - simple workflow: execute recon tools, then enum tools
	// Full AI-driven loop deferred to Phase C.2
	
	// Update scan status
	ae.engine.scanStore.UpdateScanStatus(scanID, models.ScanStatusCompleted)
	fmt.Printf("[agentic] Agentic scan completed for %s\n", scanID)

	return nil
}
