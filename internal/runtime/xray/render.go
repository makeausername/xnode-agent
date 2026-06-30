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
	Log       xrayLog         `json:"log"`
	Stats     struct{}        `json:"stats"`
	Policy    xrayPolicy      `json:"policy"`
	Inbounds  []vless.Inbound `json:"inbounds"`
	Outbounds []xrayOutbound  `json:"outbounds"`
	Routing   Routing         `json:"routing"`
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

type xrayOutbound struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
}

// RenderConfig renders the local Xray JSON config for VLESS + REALITY + Vision.
func RenderConfig(plan runtime.RuntimePlan) ([]byte, error) {
	inbound, err := vless.BuildInbound(plan.NodeConfig, plan.Users, plan.Secrets)
	if err != nil {
		return nil, err
	}

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
		Inbounds: []vless.Inbound{inbound},
		Outbounds: []xrayOutbound{
			{Protocol: "freedom", Tag: "direct"},
			{Protocol: "blackhole", Tag: "block"},
		},
		Routing: Routing{
			Rules: BuildRoutingRules(plan.Rules),
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
