package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	"github.com/makeausername/xnode-agent/internal/runtime"
)

type xrayConfig struct {
	Log       xrayLog        `json:"log"`
	Stats     struct{}       `json:"stats"`
	Policy    xrayPolicy     `json:"policy"`
	Inbounds  []xrayInbound  `json:"inbounds"`
	Outbounds []xrayOutbound `json:"outbounds"`
	Routing   xrayRouting    `json:"routing"`
}

type xrayLog struct {
	LogLevel string `json:"loglevel"`
	Access   string `json:"access"`
	Error    string `json:"error"`
}

type xrayPolicy struct {
	Levels map[string]xrayLevelPolicy `json:"levels"`
	System xraySystemPolicy           `json:"system"`
}

type xrayLevelPolicy struct {
	StatsUserUplink   bool `json:"statsUserUplink"`
	StatsUserDownlink bool `json:"statsUserDownlink"`
}

type xraySystemPolicy struct {
	StatsInboundUplink   bool `json:"statsInboundUplink"`
	StatsInboundDownlink bool `json:"statsInboundDownlink"`
}

type xrayInbound struct {
	Tag            string             `json:"tag"`
	Listen         string             `json:"listen"`
	Port           int                `json:"port"`
	Protocol       string             `json:"protocol"`
	Settings       xrayInboundSetting `json:"settings"`
	StreamSettings xrayStreamSettings `json:"streamSettings"`
	Sniffing       xraySniffing       `json:"sniffing"`
}

type xrayInboundSetting struct {
	Clients    []xrayClient `json:"clients"`
	Decryption string       `json:"decryption"`
}

type xrayClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow"`
}

type xrayStreamSettings struct {
	Network         string              `json:"network"`
	Security        string              `json:"security"`
	RealitySettings xrayRealitySettings `json:"realitySettings"`
}

type xrayRealitySettings struct {
	Show        bool     `json:"show"`
	Target      string   `json:"target"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIDs    []string `json:"shortIds"`
}

type xraySniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type xrayOutbound struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
}

type xrayRouting struct {
	Rules []xrayRoutingRule `json:"rules"`
}

type xrayRoutingRule struct {
	Type        string   `json:"type"`
	Protocol    []string `json:"protocol"`
	OutboundTag string   `json:"outboundTag"`
}

// RenderConfig renders the local Xray JSON config for VLESS + REALITY + Vision.
func RenderConfig(plan runtime.RuntimePlan) ([]byte, error) {
	if err := validatePlan(plan); err != nil {
		return nil, err
	}

	nodeConfig := plan.NodeConfig
	serverNames := cleanNonEmptyStrings(nodeConfig.Reality.ServerNames)
	shortIDs := cleanNonEmptyStrings(plan.Secrets.ShortIDs)

	config := xrayConfig{
		Log: xrayLog{
			LogLevel: "warning",
			Access:   "/var/log/xnode/access.log",
			Error:    "/var/log/xnode/xray.log",
		},
		Policy: xrayPolicy{
			Levels: map[string]xrayLevelPolicy{
				"0": {
					StatsUserUplink:   true,
					StatsUserDownlink: true,
				},
			},
			System: xraySystemPolicy{
				StatsInboundUplink:   true,
				StatsInboundDownlink: true,
			},
		},
		Inbounds: []xrayInbound{
			{
				Tag:      fmt.Sprintf("in-vless-reality-%d", nodeConfig.NodeID),
				Listen:   nodeConfig.Profile.Listen,
				Port:     nodeConfig.Profile.Port,
				Protocol: vless.DefaultProtocol,
				Settings: xrayInboundSetting{
					Clients:    buildClients(plan),
					Decryption: "none",
				},
				StreamSettings: xrayStreamSettings{
					Network:  vless.DefaultNetwork,
					Security: vless.DefaultSecurity,
					RealitySettings: xrayRealitySettings{
						Show:        false,
						Target:      strings.TrimSpace(nodeConfig.Reality.Target),
						ServerNames: serverNames,
						PrivateKey:  strings.TrimSpace(plan.Secrets.PrivateKey),
						ShortIDs:    shortIDs,
					},
				},
				Sniffing: xraySniffing{
					Enabled:      true,
					DestOverride: []string{"http", "tls", "quic"},
				},
			},
		},
		Outbounds: []xrayOutbound{
			{Protocol: "freedom", Tag: "direct"},
			{Protocol: "blackhole", Tag: "block"},
		},
		Routing: xrayRouting{
			Rules: []xrayRoutingRule{
				{Type: "field", Protocol: []string{"bittorrent"}, OutboundTag: "block"},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal xray config: %w", err)
	}

	return append(data, '\n'), nil
}

func WriteConfigAtomic(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("xray config path is required")
	}
	if !json.Valid(data) {
		return errors.New("xray config data is not valid JSON")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create xray config directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary xray config file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary xray config mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary xray config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary xray config file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace xray config file %q: %w", path, err)
	}

	return nil
}

func validatePlan(plan runtime.RuntimePlan) error {
	nodeConfig := plan.NodeConfig

	if nodeConfig.NodeID <= 0 {
		return errors.New("node_id is required and must be > 0")
	}
	if strings.TrimSpace(nodeConfig.Domain) == "" {
		return errors.New("domain is required")
	}
	if nodeConfig.Profile.Protocol != vless.DefaultProtocol {
		return fmt.Errorf("profile.protocol must be %q", vless.DefaultProtocol)
	}
	if nodeConfig.Profile.Security != vless.DefaultSecurity {
		return fmt.Errorf("profile.security must be %q", vless.DefaultSecurity)
	}
	if nodeConfig.Profile.Network != vless.DefaultNetwork {
		return fmt.Errorf("profile.network must be %q", vless.DefaultNetwork)
	}
	if nodeConfig.Profile.Port <= 0 {
		return errors.New("profile.port is required and must be > 0")
	}
	if strings.TrimSpace(nodeConfig.Reality.Target) == "" {
		return errors.New("reality.target is required")
	}
	if len(cleanNonEmptyStrings(nodeConfig.Reality.ServerNames)) == 0 {
		return errors.New("at least one reality server_name is required")
	}
	if strings.TrimSpace(plan.Secrets.PrivateKey) == "" {
		return errors.New("reality private_key is required")
	}
	if len(cleanNonEmptyStrings(plan.Secrets.ShortIDs)) == 0 {
		return errors.New("at least one reality short_id is required")
	}

	return nil
}

func buildClients(plan runtime.RuntimePlan) []xrayClient {
	clients := make([]xrayClient, 0, len(plan.Users))
	for _, user := range plan.Users {
		if !user.Enabled {
			continue
		}
		clients = append(clients, xrayClient{
			ID:    user.UUID,
			Email: fmt.Sprintf("user-%d@panel.local", user.ID),
			Flow:  plan.NodeConfig.Profile.Flow,
		})
	}
	return clients
}

func cleanNonEmptyStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}
