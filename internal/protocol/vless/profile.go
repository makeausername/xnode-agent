package vless

import "github.com/makeausername/xnode-agent/pkg/nodeapi"

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
