package vless

import (
	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	DefaultProfileName = "vless-reality-vision"
	DefaultProtocol    = "vless"
	DefaultNetwork     = "raw"
	DefaultSecurity    = "reality"
	DefaultFlow        = "xtls-rprx-vision"
	DefaultListen      = "0.0.0.0"
	DefaultPort        = 443
	DefaultTarget      = "www.microsoft.com:443"
	DefaultServerName  = "www.microsoft.com"
	DefaultFingerprint = "chrome"
)

func DefaultNodeConfig(nodeID int64, domain string) nodeapi.NodeConfig {
	return nodeapi.NodeConfig{
		SchemaVersion: 1,
		NodeID:        nodeID,
		Domain:        domain,
		Profile: nodeapi.NodeProfile{
			Name:     DefaultProfileName,
			Protocol: DefaultProtocol,
			Network:  DefaultNetwork,
			Security: DefaultSecurity,
			Flow:     DefaultFlow,
			Listen:   DefaultListen,
			Port:     DefaultPort,
		},
		Reality: nodeapi.RealityConfig{
			Target:      DefaultTarget,
			ServerNames: []string{DefaultServerName},
			Fingerprint: DefaultFingerprint,
		},
		Report: nodeapi.ReportConfig{
			UserSyncIntervalSec:  60,
			TrafficIntervalSec:   60,
			OnlineIntervalSec:    60,
			HeartbeatIntervalSec: 30,
		},
	}
}

type InboundClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow"`
}

type Inbound struct {
	Tag      string          `json:"tag"`
	Listen   string          `json:"listen"`
	Port     int             `json:"port"`
	Protocol string          `json:"protocol"`
	Clients  []InboundClient `json:"clients"`
}

func BuildClients(users []nodeapi.UserInfo, flow string) []InboundClient {
	clients := make([]InboundClient, 0, len(users))

	for _, user := range users {
		if !user.Enabled {
			continue
		}

		clients = append(clients, InboundClient{
			ID:    user.UUID,
			Email: user.Email,
			Flow:  flow,
		})
	}

	return clients
}

func BuildInbound(config nodeapi.NodeConfig, users []nodeapi.UserInfo, secret secrets.RealitySecret) Inbound {
	return Inbound{
		Tag:      "in-vless-reality",
		Listen:   config.Profile.Listen,
		Port:     config.Profile.Port,
		Protocol: config.Profile.Protocol,
		Clients:  BuildClients(users, config.Profile.Flow),
	}
}
