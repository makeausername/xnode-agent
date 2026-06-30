package panel

import (
	"context"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Client interface {
	Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error)
	GetConfig(ctx context.Context) (nodeapi.NodeConfig, error)
	GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error)
	GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error)
	ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error
	ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error
	ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error
	ReportDetectLog(ctx context.Context, report nodeapi.DetectLogReport) error
	ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error
}
