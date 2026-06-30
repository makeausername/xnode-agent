package mock

import (
	"context"

	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	DefaultNodeID = int64(1)
	DefaultDomain = "mock.xnode.local"

	usersETag       = "mock-users-v1"
	detectRulesETag = "mock-detect-rules-v1"
)

var _ panel.Client = (*Client)(nil)

type Client struct {
	config nodeapi.NodeConfig
	users  []nodeapi.UserInfo
	rules  []nodeapi.DetectRule
}

func NewClient() *Client {
	return &Client{
		config: vless.DefaultNodeConfig(DefaultNodeID, DefaultDomain),
		users: []nodeapi.UserInfo{
			{
				ID:             1,
				UUID:           "11111111-1111-4111-8111-111111111111",
				Email:          "mock-user@example.com",
				SpeedLimitMbps: 100,
				IPLimit:        2,
				Enabled:        true,
				UpdatedAt:      1700000000,
			},
		},
		rules: []nodeapi.DetectRule{},
	}
}

func (c *Client) Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error) {
	nodeID := req.NodeID
	if nodeID == 0 {
		nodeID = c.config.NodeID
	}

	domain := req.Domain
	if domain == "" {
		domain = c.config.Domain
	}

	return nodeapi.EnrollResponse{
		NodeToken:         "mock-node-token",
		PanelURL:          "mock://panel",
		NodeID:            nodeID,
		Domain:            domain,
		ReportIntervalSec: c.config.Report.TrafficIntervalSec,
		ConfigIntervalSec: c.config.Report.UserSyncIntervalSec,
	}, nil
}

func (c *Client) GetConfig(ctx context.Context) (nodeapi.NodeConfig, error) {
	return c.config, nil
}

func (c *Client) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	users := append([]nodeapi.UserInfo(nil), c.users...)
	return users, usersETag, nil
}

func (c *Client) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	rules := append([]nodeapi.DetectRule(nil), c.rules...)
	return rules, detectRulesETag, nil
}

func (c *Client) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	return nil
}

func (c *Client) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	return nil
}

func (c *Client) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	return nil
}

func (c *Client) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	return nil
}
