package runtime

import (
	"context"

	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Health struct {
	Running     bool   `json:"running"`
	PID         int    `json:"pid"`
	CoreVersion string `json:"core_version"`
	LastStartAt int64  `json:"last_start_at"`
	LastError   string `json:"last_error"`
	ConfigHash  string `json:"config_hash"`
}

type RuntimePlan struct {
	NodeConfig nodeapi.NodeConfig
	Users      []nodeapi.UserInfo
	Rules      []nodeapi.DetectRule
	Secrets    secrets.RealitySecret
	Hash       string
}

type Runtime interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) Health
	ApplyPlan(ctx context.Context, plan RuntimePlan) error
	QueryStats(ctx context.Context, reset bool) ([]nodeapi.UserTraffic, error)
}
