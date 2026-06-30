package mock

import (
	"context"
	"sync"

	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	DefaultNodeID = int64(1)
	DefaultDomain = "mock.xnode.local"

	usersETag       = `W/"mock-users-1"`
	detectRulesETag = "mock-detect-rules-v1"
)

var _ panel.Client = (*Client)(nil)

type Client struct {
	mu               sync.Mutex
	config           nodeapi.NodeConfig
	users            []nodeapi.UserInfo
	rules            []nodeapi.DetectRule
	runtimeReport    nodeapi.RuntimeReport
	heartbeatReport  nodeapi.HeartbeatReport
	hasRuntime       bool
	hasHeartbeat     bool
	runtimeReports   int
	heartbeatReports int
}

func NewClient() *Client {
	return NewClientForNode(DefaultNodeID, DefaultDomain)
}

func NewClientForNode(nodeID int64, domain string) *Client {
	if nodeID == 0 {
		nodeID = DefaultNodeID
	}
	if domain == "" {
		domain = DefaultDomain
	}

	return &Client{
		config: vless.DefaultNodeConfig(nodeID, domain),
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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.config, nil
}

func (c *Client) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if etag == usersETag {
		return nil, usersETag, nil
	}
	users := append([]nodeapi.UserInfo(nil), c.users...)
	return users, usersETag, nil
}

func (c *Client) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rules := append([]nodeapi.DetectRule(nil), c.rules...)
	return rules, detectRulesETag, nil
}

func (c *Client) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	report.ShortIDs = append([]string(nil), report.ShortIDs...)
	report.Capabilities = append([]string(nil), report.Capabilities...)
	c.runtimeReport = report
	c.hasRuntime = true
	c.runtimeReports++

	return nil
}

func (c *Client) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	return nil
}

func (c *Client) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	return nil
}

func (c *Client) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.heartbeatReport = report
	c.hasHeartbeat = true
	c.heartbeatReports++

	return nil
}

func (c *Client) LastRuntimeReport() (nodeapi.RuntimeReport, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	report := c.runtimeReport
	report.ShortIDs = append([]string(nil), report.ShortIDs...)
	report.Capabilities = append([]string(nil), report.Capabilities...)

	return report, c.hasRuntime
}

func (c *Client) LastHeartbeatReport() (nodeapi.HeartbeatReport, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.heartbeatReport, c.hasHeartbeat
}

func (c *Client) RuntimeReportCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.runtimeReports
}

func (c *Client) HeartbeatReportCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.heartbeatReports
}
