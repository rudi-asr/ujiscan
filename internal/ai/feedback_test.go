package ai

import (
	"testing"
	"time"
)

func TestUpdateToolFeedback(t *testing.T) {
	c := NewClient()

	c.UpdateToolFeedback("nmap", true, 2*time.Second)
	c.UpdateToolFeedback("nmap", true, 4*time.Second)

	fb := c.GetToolFeedback("nmap")
	if fb == nil {
		t.Fatal("expected feedback for nmap")
	}
	if fb.SuccessCount != 2 || fb.FailureCount != 0 {
		t.Errorf("expected 2/0, got %d/%d", fb.SuccessCount, fb.FailureCount)
	}
	if fb.AvgTime != 3*time.Second {
		t.Errorf("expected avg 3s, got %v", fb.AvgTime)
	}

	c.UpdateToolFeedback("nmap", false, time.Second)
	fb = c.GetToolFeedback("nmap")
	if fb.SuccessCount != 2 || fb.FailureCount != 1 {
		t.Errorf("expected 2/1 after failure, got %d/%d", fb.SuccessCount, fb.FailureCount)
	}
	if fb.LastUsed.IsZero() {
		t.Error("LastUsed should be set")
	}
}

func TestGetPreferredTools(t *testing.T) {
	c := NewClient()

	// nmap: high success rate, nuclei: poor history (0/2), default: unknown
	c.UpdateToolFeedback("nmap", true, time.Second)
	c.UpdateToolFeedback("nmap", true, time.Second)
	c.UpdateToolFeedback("nuclei", false, time.Second)
	c.UpdateToolFeedback("nuclei", false, time.Second)

	ordered := c.GetPreferredTools([]string{"nuclei", "default", "nmap"})
	if len(ordered) != 3 {
		t.Fatalf("expected 3 tools, got %d: %v", len(ordered), ordered)
	}
	// nmap (1.0) first, unknown (0.5 neutral) middle, nuclei (0.0) last
	if ordered[0] != "nmap" {
		t.Errorf("expected nmap first, got %v", ordered)
	}
	if ordered[2] != "nuclei" {
		t.Errorf("expected nuclei last (worst record), got %v", ordered)
	}
}

func TestGetToolFeedbackUnknown(t *testing.T) {
	c := NewClient()
	if fb := c.GetToolFeedback("never-used"); fb != nil {
		t.Errorf("expected nil for unknown tool, got %+v", fb)
	}
}
