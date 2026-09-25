package validator

import (
	"testing"
)

func TestClassifierWebURL(t *testing.T) {
	c := NewClassifier()
	result := c.Classify("https://example.com")
	if result.Type != TargetTypeWeb {
		t.Errorf("Expected TargetTypeWeb, got %v", result.Type)
	}
	if result.Confidence < 0.9 {
		t.Errorf("Expected high confidence, got %f", result.Confidence)
	}
}

func TestClassifierNetworkIP(t *testing.T) {
	c := NewClassifier()
	result := c.Classify("192.168.1.1")
	if result.Type != TargetTypeNetwork {
		t.Errorf("Expected TargetTypeNetwork, got %v", result.Type)
	}
}

func TestClassifierNetworkCIDR(t *testing.T) {
	c := NewClassifier()
	result := c.Classify("10.0.0.0/8")
	if result.Type != TargetTypeNetwork {
		t.Errorf("Expected TargetTypeNetwork, got %v", result.Type)
	}
}

func TestClassifierDomain(t *testing.T) {
	c := NewClassifier()
	result := c.Classify("example.com")
	if result.Type != TargetTypeWeb {
		t.Errorf("Expected TargetTypeWeb, got %v", result.Type)
	}
}

func TestClassifierWildcardDomain(t *testing.T) {
	c := NewClassifier()
	result := c.Classify("*.example.com")
	if result.Type != TargetTypeCloud {
		t.Errorf("Expected TargetTypeCloud, got %v", result.Type)
	}
}

func TestValidateLocalhostRejected(t *testing.T) {
	sv := NewScopeValidator()
	profile, err := sv.Validate("localhost", "", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if profile.IsValid {
		t.Errorf("Expected localhost to be rejected")
	}
	if len(profile.Errors) == 0 {
		t.Errorf("Expected errors for localhost")
	}
}

func TestValidate127Rejected(t *testing.T) {
	sv := NewScopeValidator()
	profile, err := sv.Validate("127.0.0.1", "", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if profile.IsValid {
		t.Errorf("Expected 127.0.0.1 to be rejected")
	}
}

func TestValidatePublicURL(t *testing.T) {
	sv := NewScopeValidator()
	profile, err := sv.Validate("https://example.com", "", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !profile.IsValid {
		t.Errorf("Expected valid profile, got errors: %v", profile.Errors)
	}
	if profile.Type != TargetTypeWeb {
		t.Errorf("Expected TargetTypeWeb, got %v", profile.Type)
	}
}

func TestValidateWithScope(t *testing.T) {
	sv := NewScopeValidator()
	profile, err := sv.Validate("https://api.example.com", "*.example.com", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !profile.IsValid {
		t.Errorf("Expected valid profile within scope, got errors: %v", profile.Errors)
	}
}

func TestValidateOutOfScope(t *testing.T) {
	sv := NewScopeValidator()
	profile, err := sv.Validate("https://other.com", "*.example.com", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if profile.IsValid {
		t.Errorf("Expected out-of-scope target to be rejected")
	}
}

func TestValidateEmptyTarget(t *testing.T) {
	sv := NewScopeValidator()
	_, err := sv.Validate("", "", "")
	if err == nil {
		t.Errorf("Expected error for empty target")
	}
}
