package playbook

import (
	"testing"
)

func TestLoadPlaybook(t *testing.T) {
	loader := NewPlaybookLoader("../../playbooks")

	pb, err := loader.LoadPlaybook("web-full-scan")
	if err != nil {
		t.Fatalf("Failed to load playbook: %v", err)
	}

	if pb.Name != "Web Server Full Scan" {
		t.Fatalf("Expected name 'Web Server Full Scan', got '%s'", pb.Name)
	}

	if pb.EntryPhase != PhaseRecon {
		t.Fatalf("Expected entry phase 'recon', got '%s'", pb.EntryPhase)
	}

	// Check phases exist
	reconSteps := pb.GetPhaseSteps(PhaseRecon)
	if len(reconSteps) == 0 {
		t.Fatal("No steps found in recon phase")
	}

	enumSteps := pb.GetPhaseSteps(PhaseEnum)
	if len(enumSteps) == 0 {
		t.Fatal("No steps found in enum phase")
	}

	t.Logf("Playbook loaded: %s (v%s)", pb.Name, pb.Version)
	t.Logf("  Recon steps: %d", len(reconSteps))
	t.Logf("  Enum steps: %d", len(enumSteps))
}

func TestLoadAllPlaybooks(t *testing.T) {
	loader := NewPlaybookLoader("../../playbooks")

	playbooks, err := loader.LoadAllPlaybooks()
	if err != nil {
		t.Fatalf("Failed to load playbooks: %v", err)
	}

	if len(playbooks) == 0 {
		t.Fatal("No playbooks loaded")
	}

	t.Logf("Loaded %d playbooks", len(playbooks))
	for _, pb := range playbooks {
		t.Logf("  - %s (v%s)", pb.Name, pb.Version)
	}
}

func TestPlaybookStructure(t *testing.T) {
	loader := NewPlaybookLoader("../../playbooks")

	pb, err := loader.LoadPlaybook("network-discovery")
	if err != nil {
		t.Fatalf("Failed to load playbook: %v", err)
	}

	// Check step details
	reconSteps := pb.GetPhaseSteps(PhaseRecon)
	for _, step := range reconSteps {
		if step.Tool == "" {
			t.Fatalf("Step %s has no tool", step.ID)
		}
		if len(step.Args) == 0 {
			t.Fatalf("Step %s has no args", step.ID)
		}
		t.Logf("Step %s: tool=%s, args=%v", step.ID, step.Tool, step.Args)
	}
}
