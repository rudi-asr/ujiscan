package tools

import (
	"testing"

	"github.com/rudi-asr/ujiscan/internal/models"
)

func TestExecutorInitialize(t *testing.T) {
	executor := NewExecutor()
	executor.InitializeDefaultTools()

	tools := executor.ListTools()
	if len(tools) == 0 {
		t.Fatal("No tools registered after initialization")
	}

	// Check nmap is available
	nmap := executor.GetToolConfig("nmap")
	if nmap == nil {
		t.Fatal("nmap tool not found in registry")
	}

	if !nmap.Available {
		t.Skip("nmap not installed on system - skipping tool test")
	}
}

func TestNmapVersion(t *testing.T) {
	executor := NewExecutor()
	executor.InitializeDefaultTools()

	nmap := executor.GetToolConfig("nmap")
	if nmap == nil || !nmap.Available {
		t.Skip("nmap not available - skipping test")
	}

	// Run nmap --version as a sanity check
	output, err := executor.RunTool("nmap", []string{"--version"}, models.PhaseRecon, "")
	if err != nil {
		t.Fatalf("Failed to run nmap --version: %v", err)
	}

	if !output.Success {
		t.Fatalf("nmap --version returned non-zero exit code: %d", output.ExitCode)
	}

	if len(output.Stdout) == 0 {
		t.Fatal("nmap --version returned empty output")
	}

	t.Logf("nmap output: %s", output.Stdout[:100])
}

func TestNucleiVersion(t *testing.T) {
	executor := NewExecutor()
	executor.InitializeDefaultTools()

	nuclei := executor.GetToolConfig("nuclei")
	if nuclei == nil || !nuclei.Available {
		t.Skip("nuclei not available - skipping test")
	}

	// Run nuclei -version as a sanity check
	output, err := executor.RunTool("nuclei", []string{"-version"}, models.PhaseEnum, "")
	if err != nil {
		t.Fatalf("Failed to run nuclei -version: %v", err)
	}

	if !output.Success {
		t.Logf("nuclei -version returned exit code: %d", output.ExitCode)
	}

	t.Logf("nuclei output: %s", output.Stdout)
}
