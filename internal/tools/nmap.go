package tools

import (
	"encoding/json"
	"fmt"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanWithNmap executes nmap with full port scan and version detection
func (e *Executor) ScanWithNmap(target string) (*models.ToolOutput, error) {
	// Clean target: remove protocol/path, keep only domain/IP
	cleanTarget := CleanTarget(target)
	
	// Construct nmap arguments for JSON output
	args := []string{
		"-sV",      // version detection
		"-p-",      // all ports
		"-oJ", "-", // output JSON to stdout
		cleanTarget,
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}

	// Parse JSON output
	if output.Success && output.Stdout != "" {
		parsed, parseErr := parseNmapJSON(output.Stdout)
		if parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}

// parseNmapJSON parses nmap JSON output
func parseNmapJSON(jsonOutput string) (interface{}, error) {
	// Simple parsing - just unmarshal and return
	var data interface{}
	err := json.Unmarshal([]byte(jsonOutput), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nmap JSON: %w", err)
	}
	return data, nil
}

// QuickNmapScan runs a fast nmap scan (common ports only)
func (e *Executor) QuickNmapScan(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	args := []string{
		"-F",       // fast mode - common ports only
		"-oJ", "-", // output JSON to stdout
		cleanTarget,
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}

	if output.Success && output.Stdout != "" {
		parsed, parseErr := parseNmapJSON(output.Stdout)
		if parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}

// NmapPingDiscovery does ping scan to discover live hosts
func (e *Executor) NmapPingDiscovery(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	args := []string{
		"-sn",      // ping scan
		"-oJ", "-", // output JSON to stdout
		cleanTarget,
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}

	if output.Success && output.Stdout != "" {
		parsed, parseErr := parseNmapJSON(output.Stdout)
		if parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}
