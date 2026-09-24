// Package findings provides finding management service
package findings

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Service manages findings throughout the lifecycle
type Service struct {
	parserRegistry *ParserRegistry
	classifier     *Classifier
	verifier       *Verifier
	findings       []*Finding
	mutex          sync.RWMutex
}

// NewService creates a new finding service
func NewService() *Service {
	parserReg := NewParserRegistry()
	parserReg.InitializeDefaultParsers()

	return &Service{
		parserRegistry: parserReg,
		classifier:     NewClassifier(),
		verifier:       NewVerifier(),
		findings:       make([]*Finding, 0),
	}
}

// ProcessToolOutput extracts, classifies, and verifies findings
func (s *Service) ProcessToolOutput(
	toolName string,
	output string,
	target string,
) []*Finding {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 1. Parse findings from output
	extracted := s.parserRegistry.Parse(toolName, output, target)
	if len(extracted) == 0 {
		return extracted
	}

	log.Printf("📊 %s: Extracted %d findings from %s", toolName, len(extracted), target)

	// 2. Classify findings
	s.classifier.ClassifyBatch(extracted)

	// 3. Verify severity
	for _, finding := range extracted {
		s.verifier.VerifySeverity(finding)
	}

	// 4. Deduplicate
	unique := s.verifier.FilterDuplicates(extracted)
	log.Printf("  After dedup: %d findings")

	// 5. Aggregate
	aggregated := s.verifier.AggregateFindings(unique)

	// 6. Store
	s.findings = append(s.findings, aggregated...)

	// 7. Summary
	counts := CountBySeverity(aggregated)
	log.Printf("  %s", counts.String())

	return aggregated
}

// GetFindings returns all findings, optionally filtered
func (s *Service) GetFindings(filters ...func(*Finding) bool) []*Finding {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(filters) == 0 {
		// Return copy of all findings
		result := make([]*Finding, len(s.findings))
		copy(result, s.findings)
		return result
	}

	// Apply filters
	filtered := make([]*Finding, 0)
	for _, finding := range s.findings {
		match := true
		for _, filter := range filters {
			if !filter(finding) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, finding)
		}
	}
	return filtered
}

// GetFindingsBySeverity returns findings at specific severity level
func (s *Service) GetFindingsBySeverity(severity string) []*Finding {
	return s.GetFindings(func(f *Finding) bool {
		return f.Severity == severity
	})
}

// GetFindingsByType returns findings of specific type
func (s *Service) GetFindingsByType(findingType string) []*Finding {
	return s.GetFindings(func(f *Finding) bool {
		return f.Type == findingType
	})
}

// GetFindingsByTarget returns findings for specific target
func (s *Service) GetFindingsByTarget(target string) []*Finding {
	return s.GetFindings(func(f *Finding) bool {
		return f.Target == target
	})
}

// GetFindingsBySource returns findings from specific tool
func (s *Service) GetFindingsBySource(source string) []*Finding {
	return s.GetFindings(func(f *Finding) bool {
		return f.Source == source
	})
}

// Summary returns findings summary
func (s *Service) Summary() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(s.findings) == 0 {
		return "No findings found"
	}

	counts := CountBySeverity(s.findings)
	grouped := GroupBySeverity(s.findings)

	var sb strings.Builder
	sb.WriteString("=== FINDINGS SUMMARY ===\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", counts.String()))

	// Group by severity and show details
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		findings := grouped[severity]
		if len(findings) == 0 {
			continue
		}

		emoji := "ℹ️"
		switch severity {
		case "critical":
			emoji = "🔴"
		case "high":
			emoji = "🟠"
		case "medium":
			emoji = "🟡"
		case "low":
			emoji = "🟢"
		}

		sb.WriteString(fmt.Sprintf("\n%s %s (%d findings):\n", emoji, strings.ToUpper(severity), len(findings)))
		for i, f := range findings {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, f.Source, f.Title))
		}
	}

	return sb.String()
}

// Clear removes all findings
func (s *Service) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.findings = make([]*Finding, 0)
}

// Export exports findings in various formats
type FindingExporter interface {
	Export(findings []*Finding) (string, error)
	Format() string
}

// JSONExporter exports findings as JSON array
type JSONExporter struct{}

func (e *JSONExporter) Format() string {
	return "json"
}

func (e *JSONExporter) Export(findings []*Finding) (string, error) {
	// Simplified - would use json.Marshal in real implementation
	return fmt.Sprintf("[%d findings]", len(findings)), nil
}

// MarkdownExporter exports findings as markdown
type MarkdownExporter struct{}

func (e *MarkdownExporter) Format() string {
	return "markdown"
}

func (e *MarkdownExporter) Export(findings []*Finding) (string, error) {
	var sb strings.Builder
	sb.WriteString("# Security Findings Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	counts := CountBySeverity(findings)
	sb.WriteString(fmt.Sprintf("**Total Findings: %d**\n\n", counts.Total))
	sb.WriteString(fmt.Sprintf("- 🔴 Critical: %d\n", counts.Critical))
	sb.WriteString(fmt.Sprintf("- 🟠 High: %d\n", counts.High))
	sb.WriteString(fmt.Sprintf("- 🟡 Medium: %d\n", counts.Medium))
	sb.WriteString(fmt.Sprintf("- 🟢 Low: %d\n", counts.Low))
	sb.WriteString(fmt.Sprintf("- ℹ️ Info: %d\n\n", counts.Info))

	grouped := GroupBySeverity(findings)
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		severityFindings := grouped[severity]
		if len(severityFindings) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s Severity\n\n", strings.ToUpper(severity)))
		for i, f := range severityFindings {
			sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, f.Title))
			sb.WriteString(fmt.Sprintf("- **Target**: %s\n", f.Target))
			sb.WriteString(fmt.Sprintf("- **Source**: %s\n", f.Source))
			sb.WriteString(fmt.Sprintf("- **Description**: %s\n", f.Description))
			sb.WriteString(fmt.Sprintf("- **Evidence**: %s\n", f.Evidence))

			if f.CWE != "" {
				sb.WriteString(fmt.Sprintf("- **CWE**: %s\n", f.CWE))
			}
			if f.CVE != "" {
				sb.WriteString(fmt.Sprintf("- **CVE**: %s\n", f.CVE))
			}
			if f.CVSS > 0 {
				sb.WriteString(fmt.Sprintf("- **CVSS**: %.1f\n", f.CVSS))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// GetExporter returns an exporter for the specified format
func GetExporter(format string) FindingExporter {
	switch format {
	case "json":
		return &JSONExporter{}
	case "markdown":
		return &MarkdownExporter{}
	default:
		return &MarkdownExporter{} // Default to markdown
	}
}
