package playbook

import (
	"fmt"
	"testing"
)

func TestPlaybookExecution(t *testing.T) {
	loader := NewPlaybookLoader("../../playbooks")

	pb, err := loader.LoadPlaybook("network-discovery")
	if err != nil {
		t.Fatalf("Failed to load playbook: %v", err)
	}

	fmt.Printf("Playbook: %s\n", pb.Name)
	fmt.Printf("Entry phase: %s\n", pb.EntryPhase)

	// List all steps
	for phase, steps := range pb.Phases {
		fmt.Printf("Phase %s: %d steps\n", phase, len(steps))
		for _, step := range steps {
			fmt.Printf("  - %s (tool=%s, args=%v)\n", step.ID, step.Tool, step.Args)
		}
	}

	// Check if steps are parseable
	reconSteps := pb.GetPhaseSteps(PhaseRecon)
	enumSteps := pb.GetPhaseSteps(PhaseEnum)

	if len(reconSteps) == 0 {
		t.Fatal("No recon steps found!")
	}

	if len(enumSteps) == 0 {
		fmt.Printf("Warning: No enum steps found (expected)\n")
	}
}
