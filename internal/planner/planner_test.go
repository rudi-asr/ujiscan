package planner

import (
	"testing"

	"github.com/rudi-asr/ujiscan/internal/validator"
)

func TestModeDetectionWeb(t *testing.T) {
	selector := NewModeSelector()
	result := selector.DetectMode(validator.TargetTypeWeb)
	if result.Mode != ModeWebStandard {
		t.Errorf("Expected ModeWebStandard, got %v", result.Mode)
	}
	if result.Confidence < 0.8 {
		t.Errorf("Expected high confidence, got %f", result.Confidence)
	}
}

func TestModeDetectionNetwork(t *testing.T) {
	selector := NewModeSelector()
	result := selector.DetectMode(validator.TargetTypeNetwork)
	if result.Mode != ModeNetworkDeep {
		t.Errorf("Expected ModeNetworkDeep, got %v", result.Mode)
	}
}

func TestModeDetectionCloud(t *testing.T) {
	selector := NewModeSelector()
	result := selector.DetectMode(validator.TargetTypeCloud)
	if result.Mode != ModeCloudInfra {
		t.Errorf("Expected ModeCloudInfra, got %v", result.Mode)
	}
}

func TestSelectMode(t *testing.T) {
	selector := NewModeSelector()
	config := selector.SelectMode(ModeQuickScan)
	if config == nil {
		t.Errorf("Expected config for QuickScan mode")
	}
	if config.Name != ModeQuickScan {
		t.Errorf("Expected QuickScan mode, got %v", config.Name)
	}
}

func TestGeneratePlanWebStandard(t *testing.T) {
	// Create valid target profile
	profile := &validator.TargetProfile{
		ID:      "test-123",
		Target:  "https://example.com",
		Type:    validator.TargetTypeWeb,
		Hosts:   []string{"example.com"},
		IsValid: true,
	}

	generator := NewPlanGenerator()
	plan, err := generator.GeneratePlan(profile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if plan == nil {
		t.Errorf("Expected plan to be generated")
	}

	if plan.Mode != ModeWebStandard {
		t.Errorf("Expected ModeWebStandard, got %v", plan.Mode)
	}

	if len(plan.Phases) == 0 {
		t.Errorf("Expected phases in plan")
	}
}

func TestGeneratePlanInvalidProfile(t *testing.T) {
	profile := &validator.TargetProfile{
		IsValid: false,
		Reason:  "Invalid target",
	}

	generator := NewPlanGenerator()
	_, err := generator.GeneratePlan(profile)
	if err == nil {
		t.Errorf("Expected error for invalid profile")
	}
}

func TestModeConfigExists(t *testing.T) {
	selector := NewModeSelector()
	config := selector.GetMode(ModeWebStandard)
	if config == nil {
		t.Errorf("Expected WebStandard mode config")
	}
}
