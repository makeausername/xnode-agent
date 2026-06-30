package sspanel

import (
	"context"
	"errors"

	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Client struct {
	PanelURL string
	Token    string
}

var _ panel.Client = (*Client)(nil)

func NewClient(panelURL string, token string) *Client {
	return &Client{
		PanelURL: panelURL,
		Token:    token,
	}
}

func (c *Client) Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error) {
	return nodeapi.EnrollResponse{}, notImplemented()
}

func (c *Client) GetConfig(ctx context.Context) (nodeapi.NodeConfig, error) {
	return nodeapi.NodeConfig{}, notImplemented()
}

func (c *Client) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	return nil, "", notImplemented()
}

func (c *Client) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	return nil, "", notImplemented()
}

func (c *Client) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	return notImplemented()
}

func (c *Client) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	return notImplemented()
}

func (c *Client) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	return notImplemented()
}

func (c *Client) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	return notImplemented()
}

func notImplemented() error {
	return errors.New("sspanel client is a placeholder: real panel calls are not implemented")
}
