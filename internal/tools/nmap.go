package tools

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanWithNmap executes nmap with full port scan and version detection
func (e *Executor) ScanWithNmap(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	
	tmpFile, err := ioutil.TempFile("", "nmap_*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	args := []string{
		"-sV",                   // version detection
		"-p-",                   // all ports
		"-oX", tmpFile.Name(),   // output XML to temp file (more reliable)
		cleanTarget,
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}
	
	xmlData, readErr := ioutil.ReadFile(tmpFile.Name())
	if readErr == nil {
		output.Stdout = string(xmlData)
		var parsed interface{}
		// Try to parse XML if needed (optional for now)
		_ = parsed
	}

	return output, nil
}

// QuickNmapScan runs a fast nmap scan (common ports only)
func (e *Executor) QuickNmapScan(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	
	tmpFile, err := ioutil.TempFile("", "nmap_quick_*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	args := []string{
		"-F",                    // fast mode - common ports only
		"-oX", tmpFile.Name(),   // output XML to temp file (more reliable than JSON)
		cleanTarget,
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}
	
	xmlData, readErr := ioutil.ReadFile(tmpFile.Name())
	if readErr == nil {
		output.Stdout = string(xmlData)
	}

	return output, nil
}

// NmapPingDiscovery does ping scan to discover live hosts
func (e *Executor) NmapPingDiscovery(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	
	tmpFile, err := ioutil.TempFile("", "nmap_ping_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	args := []string{
		"-sn",      // ping scan
		cleanTarget,
		"-oJ", tmpFile.Name(), // output JSON to temp file
	}

	output, err := e.RunTool("nmap", args, models.PhaseRecon, target)
	if err != nil {
		return output, err
	}
	
	jsonData, readErr := ioutil.ReadFile(tmpFile.Name())
	if readErr == nil {
		output.Stdout = string(jsonData)
	}

	return output, nil
}
