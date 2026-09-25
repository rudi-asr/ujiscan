package orchestrator

import (
	"sort"
	"sync"
	"time"
)

// ResultCollector collects and aggregates findings from all steps
type ResultCollector struct {
	findings []Finding
	mu       sync.RWMutex
}

// NewResultCollector creates a new result collector
func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		findings: make([]Finding, 0),
	}
}

// AddFinding adds a single finding
func (rc *ResultCollector) AddFinding(finding Finding) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if finding.ID == "" {
		finding.ID = generateFindingID()
	}
	if finding.DiscoveredAt.IsZero() {
		finding.DiscoveredAt = time.Now()
	}

	rc.findings = append(rc.findings, finding)
}

// AddFindings adds multiple findings
func (rc *ResultCollector) AddFindings(findings []Finding) {
	for _, f := range findings {
		rc.AddFinding(f)
	}
}

// GetFindings returns all collected findings
func (rc *ResultCollector) GetFindings() []Finding {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Return copy
	result := make([]Finding, len(rc.findings))
	copy(result, rc.findings)
	return result
}

// DeduplicateFindings removes duplicate findings (>80% similarity)
func (rc *ResultCollector) DeduplicateFindings() []Finding {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if len(rc.findings) <= 1 {
		return rc.findings
	}

	// Simple deduplication: group by Type + Description
	seen := make(map[string]bool)
	var unique []Finding

	for _, f := range rc.findings {
		key := f.Type + ":" + f.Description[:min(len(f.Description), 50)]
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}

	rc.findings = unique
	return rc.findings
}

// RankFindings ranks findings by severity
func (rc *ResultCollector) RankFindings() []Finding {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Create copy and sort
	findings := make([]Finding, len(rc.findings))
	copy(findings, rc.findings)

	sort.Slice(findings, func(i, j int) bool {
		// Sort by severity descending
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity > findings[j].Severity
		}
		// Then by discovery time
		return findings[i].DiscoveredAt.After(findings[j].DiscoveredAt)
	})

	return findings
}

// FilterBySeverity returns findings with minimum severity
func (rc *ResultCollector) FilterBySeverity(minSeverity int) []Finding {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var filtered []Finding
	for _, f := range rc.findings {
		if f.Severity >= minSeverity {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// GetFindingsBySource returns findings from specific source
func (rc *ResultCollector) GetFindingsBySource(source string) []Finding {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var bySource []Finding
	for _, f := range rc.findings {
		if f.Source == source {
			bySource = append(bySource, f)
		}
	}
	return bySource
}

// GetFindingsSummary returns a summary of findings
func (rc *ResultCollector) GetFindingsSummary() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	summary := map[string]interface{}{
		"total": len(rc.findings),
	}

	// Count by severity
	bySeverity := make(map[int]int)
	byType := make(map[string]int)
	bySource := make(map[string]int)

	for _, f := range rc.findings {
		bySeverity[f.Severity]++
		byType[f.Type]++
		bySource[f.Source]++
	}

	summary["by_severity"] = bySeverity
	summary["by_type"] = byType
	summary["by_source"] = bySource

	return summary
}

// ClearFindings clears all findings
func (rc *ResultCollector) ClearFindings() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.findings = make([]Finding, 0)
}

// Helper functions
func generateFindingID() string {
	return "finding-" + string(rune(time.Now().UnixNano()))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
