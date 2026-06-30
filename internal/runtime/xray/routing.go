package xray

import (
	"strings"

	"github.com/makeausername/xnode-agent/internal/audit"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Routing struct {
	Rules []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	Protocol    []string `json:"protocol,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

func BuildRoutingRules(rules []nodeapi.DetectRule) []RoutingRule {
	routingRules := []RoutingRule{blockProtocolRule("bittorrent")}

	validRules, _ := audit.FilterValidRules(rules)
	for _, rule := range validRules {
		pattern := strings.TrimSpace(rule.Pattern)

		switch strings.ToLower(strings.TrimSpace(rule.Type)) {
		case audit.RuleTypeProtocol:
			routingRules = append(routingRules, blockProtocolRule(pattern))
		case audit.RuleTypeDomainRegex:
			routingRules = append(routingRules, RoutingRule{
				Type:        "field",
				Domain:      []string{"regexp:" + pattern},
				OutboundTag: "block",
			})
		}
	}

	return routingRules
}

func blockProtocolRule(protocol string) RoutingRule {
	return RoutingRule{
		Type:        "field",
		Protocol:    []string{protocol},
		OutboundTag: "block",
	}
}
