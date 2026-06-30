package audit

import (
	"strings"
	"testing"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestValidateRuleAcceptsValidProtocolRule(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 1, Type: "protocol", Pattern: "bittorrent"}

	if err := ValidateRule(rule); err != nil {
		t.Fatalf("ValidateRule() error = %v", err)
	}
}

func TestValidateRuleAcceptsValidDomainRegexRule(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 2, Type: "domain_regex", Pattern: `(?i)example`}

	if err := ValidateRule(rule); err != nil {
		t.Fatalf("ValidateRule() error = %v", err)
	}
}

func TestValidateRuleRejectsInvalidRegex(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 3, Type: "domain_regex", Pattern: `[`}

	err := ValidateRule(rule)
	if err == nil {
		t.Fatal("ValidateRule() error = nil, want regex error")
	}
	if !strings.Contains(err.Error(), "domain_regex") {
		t.Fatalf("ValidateRule() error = %q, want domain_regex context", err.Error())
	}
}

func TestValidateRuleRejectsUnknownType(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 4, Type: "unknown", Pattern: "example"}

	err := ValidateRule(rule)
	if err == nil {
		t.Fatal("ValidateRule() error = nil, want unknown type error")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("ValidateRule() error = %q, want unsupported type", err.Error())
	}
}

func TestFilterValidRulesSkipsInvalidRules(t *testing.T) {
	rules := []nodeapi.DetectRule{
		{ID: 1, Type: "protocol", Pattern: "bittorrent"},
		{ID: 2, Type: "domain_regex", Pattern: `[`},
		{ID: 3, Type: "domain_regex", Pattern: `example\.com$`},
		{ID: 4, Type: "unknown", Pattern: "noop"},
	}

	valid, skipped := FilterValidRules(rules)
	if len(valid) != 2 {
		t.Fatalf("len(valid) = %d, want 2", len(valid))
	}
	if valid[0].ID != 1 || valid[1].ID != 3 {
		t.Fatalf("valid rules = %#v, want IDs 1 and 3", valid)
	}
	if len(skipped) != 2 {
		t.Fatalf("len(skipped) = %d, want 2", len(skipped))
	}
}

func TestMatchProtocol(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 1, Type: " protocol ", Pattern: " BitTorrent "}

	if !MatchProtocol(rule, "bittorrent") {
		t.Fatal("MatchProtocol() = false, want true")
	}
	if MatchProtocol(rule, "http") {
		t.Fatal("MatchProtocol() = true, want false for different protocol")
	}
}

func TestMatchDomain(t *testing.T) {
	rule := nodeapi.DetectRule{ID: 1, Type: "domain_regex", Pattern: `example\.com$`}

	if !MatchDomain(rule, "API.EXAMPLE.COM") {
		t.Fatal("MatchDomain() = false, want case-insensitive match")
	}
	if MatchDomain(rule, "example.net") {
		t.Fatal("MatchDomain() = true, want false for non-matching domain")
	}
}

func TestMatchDomainInvalidRegexDoesNotPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("MatchDomain() panicked: %v", recovered)
		}
	}()

	rule := nodeapi.DetectRule{ID: 1, Type: "domain_regex", Pattern: `[`}
	if MatchDomain(rule, "example.com") {
		t.Fatal("MatchDomain() = true, want false for invalid regex")
	}
}
