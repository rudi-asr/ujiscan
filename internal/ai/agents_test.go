package ai

import (
	"os"
	"testing"
)

// TestAgentKnowledgeLoader verifies bahwa file .md rahasia agents/
// termuat dan membentuk system prompt yang benar.
func TestAgentKnowledgeLoader(t *testing.T) {
	wd, _ := os.Getwd()
	// Pastikan run dari repo root
	os.Chdir("../..")

	l := NewAgentKnowledgeLoader("agents")

	master, err := l.LoadMasterInstructions()
	if err != nil {
		t.Fatalf("master gagal dimuat: %v", err)
	}
	if len(master) < 500 {
		t.Fatalf("master terlalu pendek: %d bytes", len(master))
	}

	rules, err := l.LoadRules()
	if err != nil {
		t.Fatalf("rules gagal dimuat: %v", err)
	}
	if len(rules) < 100 {
		t.Fatalf("rules terlalu pendek: %d bytes", len(rules))
	}

	skill, err := l.LoadSkillFase("recon")
	if err != nil {
		t.Fatalf("skill recon gagal: %v", err)
	}
	if len(skill) < 200 {
		t.Fatalf("skill recon terlalu pendek: %d bytes", len(skill))
	}

	tools := l.LoadToolCatalogForPhase("recon")
	if len(tools) < 3 {
		t.Fatalf("tools recon kurang: %v", tools)
	}
	t.Logf("recon tools: %v", tools)

	vulnTools := l.LoadToolCatalogForPhase("vulnscan")
	if len(vulnTools) < 2 {
		t.Fatalf("tools vulnscan kurang: %v", vulnTools)
	}

	sys, err := l.BuildSystemPrompt("recon", "comprehensive")
	if err != nil {
		t.Fatalf("system prompt gagal: %v", err)
	}
	if len(sys) < 1000 {
		t.Fatalf("system prompt terlalu pendek: %d bytes", len(sys))
	}
	t.Logf("System prompt: %d bytes", len(sys))

	catalog, err := l.LoadToolCatalog()
	if err != nil {
		t.Fatalf("catalog gagal: %v", err)
	}
	t.Logf("Total tools di katalog: %d", len(catalog))

	os.Chdir(wd)
}