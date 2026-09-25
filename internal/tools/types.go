package tools

import "time"

// Tool represents a security tool in the registry
type Tool struct {
	// Tool name (nmap, nuclei, etc)
	Name string `json:"name"`
	// Description
	Description string `json:"description"`
	// Category (recon, scanning, exploitation, etc)
	Category string `json:"category"`
	// Installation command
	InstallCommand string `json:"install_command"`
	// Command to verify installation
	VerifyCommand string `json:"verify_command"`
	// How to execute the tool (with {{ params }})
	ExecuteTemplate string `json:"execute_template"`
	// Supported platforms (macOS, linux, windows)
	Platforms []string `json:"platforms"`
	// Version required
	MinVersion string `json:"min_version"`
	// Parser strategy (nmap_xml, nuclei_json, etc)
	ParserStrategy string `json:"parser_strategy"`
	// Tool-specific config
	Config map[string]interface{} `json:"config,omitempty"`
}

// ToolCache represents cached tool installation state
type ToolCache struct {
	// Tool name
	ToolName string `json:"tool_name"`
	// Is currently installed
	Installed bool `json:"installed"`
	// Installed version
	Version string `json:"version"`
	// Installation timestamp
	InstalledAt time.Time `json:"installed_at"`
	// Last used timestamp
	LastUsed *time.Time `json:"last_used,omitempty"`
	// Error message if installation failed
	Error string `json:"error,omitempty"`
	// Installation path
	Path string `json:"path,omitempty"`
}

// ToolRegistry holds all tool definitions
type ToolRegistry struct {
	Version string           `json:"version"`
	Tools   map[string]*Tool `json:"tools"`
}

// ToolExecutionResult represents result of tool execution
type ToolExecutionResult struct {
	// Tool name
	ToolName string `json:"tool_name"`
	// Success status
	Success bool `json:"success"`
	// Exit code
	ExitCode int `json:"exit_code"`
	// Standard output
	Stdout string `json:"stdout,omitempty"`
	// Standard error
	Stderr string `json:"stderr,omitempty"`
	// Parsed findings (converted to Finding objects)
	Findings []interface{} `json:"findings,omitempty"`
	// Execution duration (seconds)
	Duration int `json:"duration"`
	// Error message
	Error string `json:"error,omitempty"`
}
