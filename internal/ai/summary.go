package ai

import (
	"context"
	"fmt"
	"strings"
)

// GenerateExecutiveSummary produces an AI-driven executive summary (C.5).
// Without an API key it falls back to a deterministic board-friendly
// template so report generation always works.
func (c *Client) GenerateExecutiveSummary(ctx context.Context, target string, counts map[string]int) (string, error) {
	fallback := BuildFallbackSummary(target, counts)

	// Mock mode — no API key configured
	if c.apiKey == "" {
		return fallback, nil
	}

	prompt := fmt.Sprintf(
		"You are a senior penetration testing report writer. Write a concise executive summary "+
			"(2-3 sentences, non-technical, board-friendly) for a security assessment of %s with findings: "+
			"critical=%d high=%d medium=%d low=%d info=%d. Focus on overall risk posture and urgency, "+
			"not tool names.",
		target, counts["critical"], counts["high"], counts["medium"], counts["low"], counts["info"],
	)

	resp, err := c.callOpenAIRaw(ctx, prompt)
	if err != nil || strings.TrimSpace(resp) == "" {
		// AI unavailable — keep deterministic fallback
		return fallback, nil
	}
	return strings.TrimSpace(resp), nil
}

// BuildFallbackSummary returns the deterministic executive summary used when
// no AI provider is configured or reachable.
func BuildFallbackSummary(target string, counts map[string]int) string {
	total := counts["critical"] + counts["high"] + counts["medium"] + counts["low"] + counts["info"]
	if total == 0 {
		return fmt.Sprintf(
			"Security assessment of %s completed with no confirmed findings across the executed "+
				"tests. The target surface appears adequately hardened; periodic re-testing is recommended.",
			target,
		)
	}

	critical := counts["critical"]
	high := counts["high"]

	urgency := "routine monitoring is recommended"
	switch {
	case critical > 0:
		urgency = "immediate remediation is required"
	case high > 0:
		urgency = "prioritized remediation is strongly recommended"
	case counts["medium"] > 0:
		urgency = "scheduled remediation is recommended"
	}

	return fmt.Sprintf(
		"Security assessment of %s identified %d finding(s): %d critical, %d high, %d medium, "+
			"%d low, %d informational. Overall posture: %s. Full technical details, evidence, and "+
			"remediation steps are provided in the findings below.",
		target, total, critical, high, counts["medium"], counts["low"], counts["info"], urgency,
	)
}
