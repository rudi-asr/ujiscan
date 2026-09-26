package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AgentKnowledgeLoader memuat file .md rahasia dari direktori agents/
// dan menyusunnya menjadi system prompt untuk Full Scan AI.
//
// Filosofi: kunci Full Scan ada di FILE .md (bukan di LLM).
// LLM hanya membaca instruksi dari sini — mudah diaudit & dipatenkan.
type AgentKnowledgeLoader struct {
	agentsDir string
}

// CatalogTool mewakili satu entri tools-catalog.json
type CatalogTool struct {
	ToolsName    string   `json:"tools_name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Local        bool     `json:"local"`
	Phase        []string `json:"phase"`
	Tags         []string `json:"tags"`
	Alternatives []string `json:"alternatives"`
	Command      *struct {
		Base  string   `json:"base"`
		Flags []string `json:"flags,omitempty"`
	} `json:"command,omitempty"`
}

// NewAgentKnowledgeLoader membuat loader untuk direktori agents/
// Relatif ke working dir (biasanya repo root ujiscan).
func NewAgentKnowledgeLoader(agentsDir string) *AgentKnowledgeLoader {
	if agentsDir == "" {
		agentsDir = "agents"
	}
	return &AgentKnowledgeLoader{agentsDir: agentsDir}
}

// LoadMasterInstructions membaca agents/AGENTS.md (workflow master).
func (l *AgentKnowledgeLoader) LoadMasterInstructions() (string, error) {
	return l.readFile(filepath.Join(l.agentsDir, "AGENTS.md"))
}

// LoadRules membaca semua file di agents/rules/*.md.
func (l *AgentKnowledgeLoader) LoadRules() (string, error) {
	dir := filepath.Join(l.agentsDir, "rules")
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("no rules files in %s", dir)
	}
	var sb strings.Builder
	for _, m := range matches {
		content, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		sb.WriteString("\n--- RULES: " + filepath.Base(m) + " ---\n")
		sb.Write(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// LoadSkillFase membaca SKILL.md untuk fase tertentu
// (recon/enumeration/vulnscan/report) dari agents/skills/ujiscan-<fase>/SKILL.md.
func (l *AgentKnowledgeLoader) LoadSkillFase(fase string) (string, error) {
	if fase == "" {
		return "", nil
	}
	p := filepath.Join(l.agentsDir, "skills", "ujiscan-"+fase, "SKILL.md")
	content, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("skill %s tidak ditemukan: %w", fase, err)
	}
	return string(content), nil
}

// LoadToolCatalog membaca agents/tools-catalog.json → daftar tool untuk prompt.
func (l *AgentKnowledgeLoader) LoadToolCatalog() ([]CatalogTool, error) {
	raw, err := os.ReadFile(filepath.Join(l.agentsDir, "tools-catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("tools-catalog.json tidak ditemukan: %w", err)
	}
	var catalog struct {
		Tools []CatalogTool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("parse tools-catalog gagal: %w", err)
	}
	return catalog.Tools, nil
}

// LoadToolCatalogForPhase mengembalikan daftar tools per fase
// (recon/enumeration/vulnscan/verification), untuk prompt pemilihan tool.
func (l *AgentKnowledgeLoader) LoadToolCatalogForPhase(fase string) []string {
	tools, err := l.LoadToolCatalog()
	if err != nil {
		return nil
	}
	var names []string
	for _, t := range tools {
		for _, p := range t.Phase {
			if p == fase {
				names = append(names, t.ToolsName)
				break
			}
		}
	}
	return names
}

// BuildSystemPrompt menyusun system prompt lengkap untuk agent per fase:
// master instructions + rules + skill fase + katalog tools fase.
// Inilah "otak .md" yang di-inject ke LLM.
func (l *AgentKnowledgeLoader) BuildSystemPrompt(fase string, objective string) (string, error) {
	var sb strings.Builder

	master, err := l.LoadMasterInstructions()
	if err == nil {
		sb.WriteString("=== UJISCAN MASTER WORKFLOW ===\n")
		sb.WriteString(master)
		sb.WriteString("\n\n")
	}

	rules, err := l.LoadRules()
	if err == nil {
		sb.WriteString("=== LEGAL & SCOPE RULES (WAJIB PATUH) ===\n")
		sb.WriteString(rules)
		sb.WriteString("\n\n")
	}

	if fase != "" {
		skill, err := l.LoadSkillFase(fase)
		if err == nil {
			sb.WriteString("=== SKILL FASE: " + strings.ToUpper(fase) + " ===\n")
			sb.WriteString(skill)
			sb.WriteString("\n\n")
		}
	}

	if objective != "" {
		sb.WriteString("=== USER OBJECTIVE ===\n")
		sb.WriteString(objective)
		sb.WriteString("\n\n")
	}

	catalog := l.LoadToolCatalogForPhase(fase)
	if len(catalog) > 0 {
		sb.WriteString("=== TOOLS TERSEDIA UNTUK FASE INI ===\n")
		sb.WriteString(strings.Join(catalog, ", "))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// readFile helper
func (l *AgentKnowledgeLoader) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}