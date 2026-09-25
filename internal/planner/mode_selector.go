package planner

import (
	"strings"

	"github.com/rudi-asr/ujiscan/internal/validator"
)

// ModeSelector detects engagement mode from target profile
type ModeSelector struct {
	modes map[EngagementMode]*ModeConfig
}

// NewModeSelector creates a new mode selector
func NewModeSelector() *ModeSelector {
	ms := &ModeSelector{
		modes: make(map[EngagementMode]*ModeConfig),
	}
	ms.initializeModes()
	return ms
}

// initializeModes sets up default mode configurations
func (ms *ModeSelector) initializeModes() {
	ms.modes[ModeWebStandard] = &ModeConfig{
		Name:        ModeWebStandard,
		Description: "Standard web application penetration test",
		Tools:       []string{"nmap", "nuclei", "ffuf"},
		Phases:      []ExecutionPhase{PhaseRecon, PhaseScanning, PhaseAnalysis},
		Duration:    240, // 4 hours
		Indicators: []string{
			"HTTP/HTTPS URL",
			"Web domain",
			"Web ports (80, 443, 8080)",
		},
	}

	ms.modes[ModeNetworkDeep] = &ModeConfig{
		Name:        ModeNetworkDeep,
		Description: "Deep network reconnaissance and scanning",
		Tools:       []string{"nmap", "masscan", "nuclei"},
		Phases:      []ExecutionPhase{PhaseRecon, PhaseScanning, PhaseAnalysis},
		Duration:    480, // 8 hours
		Indicators: []string{
			"IP address",
			"CIDR range",
			"Network infrastructure",
		},
	}

	ms.modes[ModeCloudInfra] = &ModeConfig{
		Name:        ModeCloudInfra,
		Description: "Cloud infrastructure and cloud services testing",
		Tools:       []string{"nmap", "nuclei", "cloud-scanner"},
		Phases:      []ExecutionPhase{PhaseRecon, PhaseScanning, PhaseAnalysis},
		Duration:    300, // 5 hours
		Indicators: []string{
			"Wildcard domain (*.example.com)",
			"Cloud provider indicators",
			"Multi-tenant indicators",
		},
	}

	ms.modes[ModeAggressiveTest] = &ModeConfig{
		Name:        ModeAggressiveTest,
		Description: "Aggressive red-team style testing (exploitation included)",
		Tools:       []string{"nmap", "nuclei", "metasploit", "ffuf"},
		Phases:      []ExecutionPhase{PhaseRecon, PhaseScanning, PhaseAnalysis},
		Duration:    600, // 10 hours
		Indicators: []string{
			"Red team engagement",
			"Authorized exploitation allowed",
			"Full scope engagement",
		},
	}

	ms.modes[ModeQuickScan] = &ModeConfig{
		Name:        ModeQuickScan,
		Description: "Quick priority finding scan",
		Tools:       []string{"nuclei", "ffuf"},
		Phases:      []ExecutionPhase{PhaseScanning, PhaseAnalysis},
		Duration:    60, // 1 hour
		Indicators: []string{
			"Quick scan request",
			"Limited scope",
			"Priority findings only",
		},
	}
}

// DetectMode analyzes target profile and detects engagement mode
func (ms *ModeSelector) DetectMode(targetType validator.TargetType) *ModeDetectionResult {
	result := &ModeDetectionResult{
		Indicators: []string{},
	}

	switch targetType {
	case validator.TargetTypeWeb:
		result.Mode = ModeWebStandard
		result.Confidence = 0.95
		result.Reason = "Target is web application"
		result.Indicators = []string{"HTTP/HTTPS target detected"}

	case validator.TargetTypeNetwork:
		result.Mode = ModeNetworkDeep
		result.Confidence = 0.9
		result.Reason = "Target is network infrastructure"
		result.Indicators = []string{"IP/CIDR target detected"}

	case validator.TargetTypeCloud:
		result.Mode = ModeCloudInfra
		result.Confidence = 0.85
		result.Reason = "Target is cloud infrastructure"
		result.Indicators = []string{"Cloud/wildcard domain detected"}

	case validator.TargetTypeMobile:
		result.Mode = ModeWebStandard // Fallback to web
		result.Confidence = 0.7
		result.Reason = "Mobile target - using web test mode"

	default:
		result.Mode = ModeQuickScan
		result.Confidence = 0.5
		result.Reason = "Unknown target type - using quick scan mode"
	}

	return result
}

// SelectMode allows manual override of detected mode
func (ms *ModeSelector) SelectMode(mode EngagementMode) *ModeConfig {
	if config, exists := ms.modes[mode]; exists {
		return config
	}
	// Fallback to web standard
	return ms.modes[ModeWebStandard]
}

// ListModes returns all available modes
func (ms *ModeSelector) ListModes() map[EngagementMode]*ModeConfig {
	return ms.modes
}

// GetMode returns specific mode config
func (ms *ModeSelector) GetMode(mode EngagementMode) *ModeConfig {
	if config, exists := ms.modes[mode]; exists {
		return config
	}
	return nil
}

// SuggestModes suggests modes based on target characteristics
func (ms *ModeSelector) SuggestModes(target string) []*ModeDetectionResult {
	var suggestions []*ModeDetectionResult

	// Check for engagement type keywords in target
	lower := strings.ToLower(target)

	if strings.Contains(lower, "api") || strings.Contains(lower, "app") {
		suggestions = append(suggestions, &ModeDetectionResult{
			Mode:       ModeWebStandard,
			Confidence: 0.9,
			Reason:     "Target appears to be web API/application",
		})
	}

	if strings.Contains(lower, "cloud") || strings.Contains(lower, "aws") {
		suggestions = append(suggestions, &ModeDetectionResult{
			Mode:       ModeCloudInfra,
			Confidence: 0.85,
			Reason:     "Target appears to be cloud infrastructure",
		})
	}

	if strings.Contains(lower, "network") || strings.Contains(lower, "infra") {
		suggestions = append(suggestions, &ModeDetectionResult{
			Mode:       ModeNetworkDeep,
			Confidence: 0.8,
			Reason:     "Target appears to be network infrastructure",
		})
	}

	// Always suggest quick scan as fallback
	suggestions = append(suggestions, &ModeDetectionResult{
		Mode:       ModeQuickScan,
		Confidence: 0.5,
		Reason:     "Generic quick scan mode",
	})

	return suggestions
}
