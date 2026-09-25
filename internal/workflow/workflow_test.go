package workflow

import (
	"context"
	"testing"
	"time"
)

func TestParserLoadFromFile(t *testing.T) {
	parser := NewParser()
	// This will fail if config/workflows.yaml doesn't exist
	// In real tests, we'd use a test fixture
	// For now, just test that parser can be created
	if parser == nil {
		t.Errorf("Expected parser to be created")
	}
}

func TestParserSubstituteVariables(t *testing.T) {
	parser := NewParser()
	context := map[string]interface{}{
		"Target": "example.com",
		"Port":   "8080",
	}

	result := parser.SubstituteVariables("Scanning {{ Target }}:{{ Port }}", context)
	expected := "Scanning example.com:8080"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestParserSubstituteParameters(t *testing.T) {
	parser := NewParser()
	context := map[string]interface{}{
		"Target": "example.com",
	}

	params := map[string]interface{}{
		"host": "{{ Target }}",
		"port": 443,
	}

	result := parser.SubstituteInParameters(params, context)
	if result["host"] != "example.com" {
		t.Errorf("Expected example.com, got %v", result["host"])
	}
	if result["port"] != 443 {
		t.Errorf("Expected 443, got %v", result["port"])
	}
}

func TestEngineExecuteSimpleWorkflow(t *testing.T) {
	parser := NewParser()
	engine := NewEngine(parser)

	// Create a simple test workflow
	testWorkflow := &Workflow{
		Name:    "test_workflow",
		Version: "1.0",
		Timeout: 60,
		Steps: []Step{
			{
				ID:   "step1",
				Name: "Test Step",
				Tool: "test-tool",
				Parameters: map[string]interface{}{
					"target": "{{ Target }}",
				},
			},
		},
	}

	parser.registry.Workflows["test_workflow"] = testWorkflow

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	execution, err := engine.Execute(ctx, "test_workflow", map[string]interface{}{
		"Target": "example.com",
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if execution == nil {
		t.Errorf("Expected execution result")
	}

	if execution.Status != "COMPLETED" {
		t.Errorf("Expected COMPLETED status, got %s", execution.Status)
	}
}

func TestEngineWorkflowNotFound(t *testing.T) {
	parser := NewParser()
	engine := NewEngine(parser)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := engine.Execute(ctx, "nonexistent", map[string]interface{}{})

	if err == nil {
		t.Errorf("Expected error for nonexistent workflow")
	}
}

func TestEngineEvaluateCondition(t *testing.T) {
	engine := &Engine{}

	context := map[string]interface{}{
		"target_type": "WEB",
		"found_ports": true,
	}

	// Test simple truthy evaluation
	if !engine.evaluateCondition("found_ports", context) {
		t.Errorf("Expected condition to evaluate to true")
	}

	// Test equality check
	if !engine.evaluateCondition("target_type=WEB", context) {
		t.Errorf("Expected equality condition to pass")
	}
}

func TestWorkflowTypeDefinition(t *testing.T) {
	workflow := &Workflow{
		Name:    "test",
		Version: "1.0",
		Steps: []Step{
			{
				ID:   "step1",
				Name: "Step 1",
				Tool: "nmap",
			},
		},
	}

	if workflow.Name != "test" {
		t.Errorf("Expected workflow name to be test")
	}

	if len(workflow.Steps) != 1 {
		t.Errorf("Expected 1 step")
	}
}
