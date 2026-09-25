package tools

import (
	"testing"
)

func TestToolsTypesExist(t *testing.T) {
	// Just verify types are defined
	tool := &Tool{
		Name:        "test",
		Category:    "recon",
		Description: "Test tool",
	}

	if tool.Name != "test" {
		t.Errorf("Expected tool name test")
	}
}

func TestToolCacheCreation(t *testing.T) {
	cache := &ToolCache{
		ToolName:  "nmap",
		Installed: true,
	}

	if !cache.Installed {
		t.Errorf("Expected installed to be true")
	}
}

func TestToolExecutionResultStruct(t *testing.T) {
	result := &ToolExecutionResult{
		ToolName: "nmap",
		Success:  true,
		ExitCode: 0,
		Stdout:   "test output",
	}

	if !result.Success {
		t.Errorf("Expected success")
	}
}
