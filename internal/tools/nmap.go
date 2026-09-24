package tools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanWithNmap executes nmap with full port scan and version detection
func (e *Executor) ScanWithNmap(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	
	tmpFile, err := ioutil.TempFile("", "nmap_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	args := []string{
		"-sV",           // version detection
		"-p-",           // all ports
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
		var parsed interface{}
		if parseErr := json.Unmarshal(jsonData, &parsed); parseErr == nil {
			output.Parsed = parsed
		}
	}

	return output, nil
}

// QuickNmapScan runs a fast nmap scan (common ports only)
func (e *Executor) QuickNmapScan(target string) (*models.ToolOutput, error) {
	cleanTarget := CleanTarget(target)
	
	tmpFile, err := ioutil.TempFile("", "nmap_quick_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	args := []string{
		"-F",       // fast mode - common ports only
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
