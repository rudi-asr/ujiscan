package tools

import (
	"testing"
)

func TestQuickNmapScan(t *testing.T) {
	executor := NewExecutor()
	executor.InitializeDefaultTools()

	nmap := executor.GetToolConfig("nmap")
	if nmap == nil || !nmap.Available {
		t.Skip("nmap not available - skipping test")
	}

	// Scan localhost (safe)
	output, err := executor.QuickNmapScan("127.0.0.1")
	if err != nil {
		t.Fatalf("QuickNmapScan failed: %v", err)
	}

	if output.ExitCode != 0 {
		t.Logf("Exit code: %d, Stderr: %s", output.ExitCode, output.Stderr)
	}

	t.Logf("Scan completed in %dms", output.Duration)
	t.Logf("Stdout length: %d bytes", len(output.Stdout))
}
