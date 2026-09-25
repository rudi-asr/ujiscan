package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Registry manages tool definitions and cache
type Registry struct {
	tools   map[string]*Tool
	cache   map[string]*ToolCache
	mu      sync.RWMutex
	cachDir string
}

// NewRegistry creates a new tool registry
func NewRegistry(toolsConfigPath string) (*Registry, error) {
	r := &Registry{
		tools:   make(map[string]*Tool),
		cache:   make(map[string]*ToolCache),
		cachDir: filepath.Join(os.TempDir(), "ujiscan-tools"),
	}

	// Create cache directory
	if err := os.MkdirAll(r.cachDir, 0755); err != nil {
		return nil, err
	}

	// Load tools from YAML
	if err := r.loadToolsFromFile(toolsConfigPath); err != nil {
		return nil, err
	}

	return r, nil
}

// loadToolsFromFile loads tool definitions from YAML file
func (r *Registry) loadToolsFromFile(path string) error {
	fmt.Printf("[registry] Loading tools from: %s\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tools config: %w", err)
	}
	fmt.Printf("[registry] Read %d bytes from config\n", len(data))

	registry := &ToolRegistry{}
	if err := yaml.Unmarshal(data, registry); err != nil {
		return fmt.Errorf("failed to parse tools config: %w", err)
	}

	fmt.Printf("[registry] Parsed %d tools from YAML\n", len(registry.Tools))
	for name, tool := range registry.Tools {
		fmt.Printf("[registry] Tool '%s': execute_template='%s'\n", name, tool.ExecuteTemplate)
	}

	r.tools = registry.Tools
	return nil
}

// GetTool returns a tool by name
func (r *Registry) GetTool(name string) *Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// ListTools returns all available tools
func (r *Registry) ListTools() map[string]*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Tool)
	for k, v := range r.tools {
		result[k] = v
	}
	return result
}

// GetToolsByCategory returns tools in a specific category
func (r *Registry) GetToolsByCategory(category string) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Tool
	for _, tool := range r.tools {
		if tool.Category == category {
			result = append(result, tool)
		}
	}
	return result
}

// IsInstalled checks if a tool is installed
func (r *Registry) IsInstalled(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cache, exists := r.cache[toolName]
	return exists && cache.Installed
}

// SetInstalled marks a tool as installed
func (r *Registry) SetInstalled(toolName string, version string, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.cache[toolName] = &ToolCache{
		ToolName:    toolName,
		Installed:   true,
		Version:     version,
		InstalledAt: now,
		Path:        path,
	}
}

// SetInstallationError marks installation as failed
func (r *Registry) SetInstallationError(toolName string, err string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[toolName] = &ToolCache{
		ToolName:  toolName,
		Installed: false,
		Error:     err,
	}
}

// GetInstallationCache returns installation cache info
func (r *Registry) GetInstallationCache(toolName string) *ToolCache {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cache, exists := r.cache[toolName]; exists {
		return cache
	}
	return nil
}

// UpdateLastUsed updates the last used timestamp
func (r *Registry) UpdateLastUsed(toolName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cache, exists := r.cache[toolName]; exists {
		now := time.Now()
		cache.LastUsed = &now
	}
}

// ClearCache clears installation cache
func (r *Registry) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*ToolCache)
}
