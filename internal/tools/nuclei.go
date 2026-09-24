package tools

import (
	"encoding/json"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanWithNuclei executes nuclei vulnerability scanner
func (e *Executor) ScanWithNuclei(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	args := []string{
		"-u", cleanTarget,
		"-jsonl", // JSONL output (nuclei v3+)
		// Removed: -json, -stats (not available/needed in v3.11+)
	}

	output, err := e.RunTool("nuclei", args, models.PhaseEnum, target)
	if err != nil {
		return output, err
	}

	// Parse JSONL output (one JSON object per line)
	if output.Stdout != "" {
		parsed, parseErr := parseNucleiJSONL(output.Stdout)
		if parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}

// ScanWithNucleiCustom executes nuclei with custom templates
func (e *Executor) ScanWithNucleiCustom(target, templates string) (*models.ToolOutput, error) {
	args := []string{
		"-u", target,
		"-t", templates, // custom template path
		"-json",
		"-stats", "false",
	}

	output, err := e.RunTool("nuclei", args, models.PhaseEnum, target)
	if err != nil {
		return output, err
	}

	if output.Stdout != "" {
		parsed, parseErr := parseNucleiJSON(output.Stdout)
		if parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}

// parseNucleiJSON parses nuclei JSON output (each line is a JSON object)
func parseNucleiJSON(jsonOutput string) ([]models.NucleiResult, error) {
	results := make([]models.NucleiResult, 0)
	
	// nuclei outputs one JSON object per line
	lines := strings.Split(strings.TrimSpace(jsonOutput), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var result models.NucleiResult
		err := json.Unmarshal([]byte(line), &result)
		if err != nil {
			// Skip lines that can't be parsed
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// parseNucleiJSONL parses nuclei JSONL output (one JSON object per line)
func parseNucleiJSONL(jsonlOutput string) (interface{}, error) {
	var results []interface{}
	
	lines := strings.Split(strings.TrimSpace(jsonlOutput), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var obj interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// Skip malformed lines, continue parsing
			continue
		}
		results = append(results, obj)
	}
	
	return results, nil
}

// CountVulnerabilitiesBySeverity counts nuclei results by severity
func CountVulnerabilitiesBySeverity(results []models.NucleiResult) map[string]int {
	counts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}

	for _, result := range results {
		severity := strings.ToLower(result.Severity)
		if _, ok := counts[severity]; ok {
			counts[severity]++
		}
	}

	return counts
}

// FilterNucleiByTag filters nuclei results by tag
func FilterNucleiByTag(results []models.NucleiResult, tag string) []models.NucleiResult {
	filtered := make([]models.NucleiResult, 0)
	for _, result := range results {
		if strings.Contains(strings.ToLower(result.Tags), strings.ToLower(tag)) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}
