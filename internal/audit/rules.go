package audit

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	RuleTypeProtocol    = "protocol"
	RuleTypeDomainRegex = "domain_regex"
)

// ValidateRule checks whether a detect rule is supported and safe to use.
func ValidateRule(rule nodeapi.DetectRule) error {
	ruleType := normalizeRuleType(rule.Type)
	pattern := strings.TrimSpace(rule.Pattern)

	switch ruleType {
	case RuleTypeProtocol:
		if pattern == "" {
			return fmt.Errorf("detect rule %d protocol pattern is required", rule.ID)
		}
		return nil
	case RuleTypeDomainRegex:
		if pattern == "" {
			return fmt.Errorf("detect rule %d domain_regex pattern is required", rule.ID)
		}
		if _, err := compileDomainPattern(pattern); err != nil {
			return fmt.Errorf("detect rule %d domain_regex pattern: %w", rule.ID, err)
		}
		return nil
	case "":
		return fmt.Errorf("detect rule %d type is required", rule.ID)
	default:
		return fmt.Errorf("detect rule %d unsupported type %q", rule.ID, rule.Type)
	}
}

func FilterValidRules(rules []nodeapi.DetectRule) ([]nodeapi.DetectRule, []error) {
	valid := make([]nodeapi.DetectRule, 0, len(rules))
	var skipped []error

	for _, rule := range rules {
		if err := ValidateRule(rule); err != nil {
			skipped = append(skipped, err)
			continue
		}
		valid = append(valid, rule)
	}

	return valid, skipped
}

func MatchProtocol(rule nodeapi.DetectRule, protocol string) bool {
	if normalizeRuleType(rule.Type) != RuleTypeProtocol {
		return false
	}
	if err := ValidateRule(rule); err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(rule.Pattern), strings.TrimSpace(protocol))
}

func MatchDomain(rule nodeapi.DetectRule, domain string) bool {
	if normalizeRuleType(rule.Type) != RuleTypeDomainRegex {
		return false
	}
	if err := ValidateRule(rule); err != nil {
		return false
	}

	re, err := compileDomainPattern(strings.TrimSpace(rule.Pattern))
	if err != nil {
		return false
	}
	return re.MatchString(strings.TrimSpace(domain))
}

func normalizeRuleType(ruleType string) string {
	return strings.ToLower(strings.TrimSpace(ruleType))
}

func compileDomainPattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i:" + pattern + ")")
}
