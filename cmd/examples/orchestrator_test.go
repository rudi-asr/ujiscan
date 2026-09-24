// Example of using the orchestrator with rules and scope
package main

import (
	"fmt"

	"github.com/rudi-asr/ujiscan/internal/orchestrator"
	"github.com/rudi-asr/ujiscan/internal/registry"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

func exampleOrchestrator() error {
	// Load registry
	reg, err := registry.LoadRegistry("tools.yaml")
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Create executor
	regExecutor := tools.NewRegistryExecutor(reg)

	// Create orchestrator with scope validation (strict mode)
	orch := orchestrator.NewOrchestrator(reg, regExecutor, true)

	// Define scope for this engagement
	fmt.Println("📋 Configuring scope...")
	orch.AddScopePattern("example.com")
	orch.AddScopePattern("*.example.com")
	orch.AddScopePattern("192.168.0.0/16")

	// Deny specific targets
	orch.AddDenyPattern("internal-api.example.com")

	fmt.Println(orch.ScopeConfig())

	// Test targets
	testTargets := []string{
		"example.com",
		"api.example.com",
		"internal-api.example.com", // Should be denied
		"192.168.1.100",
		"192.168.2.50",
		"10.0.0.1", // Out of scope in strict mode
	}

	fmt.Println("\n🔍 Validating targets...")
	for _, target := range testTargets {
		validation := orch.ValidateTarget(target)
		status := "✓"
		if !validation.Valid {
			status = "✗"
		}
		fmt.Printf("  %s %s - %s\n", status, target, validation.Reason)
	}

	// If desired, execute a mode
	// orch.ValidateAndExecuteMode("example.com", "reconnaissance")
	// fmt.Println(orch.Summary())

	return nil
}

// This would be run as a separate example/test
// To use: go run cmd/examples/orchestrator/main.go
