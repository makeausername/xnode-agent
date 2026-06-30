package xray

import (
	"context"

	"github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Runtime struct {
	BinPath    string
	ConfigPath string
	health     runtime.Health
}

func NewRuntime(binPath string, configPath string) *Runtime {
	return &Runtime{
		BinPath:    binPath,
		ConfigPath: configPath,
		health: runtime.Health{
			Running: false,
		},
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	r.health.Running = true
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.health.Running = false
	return nil
}

func (r *Runtime) Health(ctx context.Context) runtime.Health {
	return r.health
}

func (r *Runtime) ApplyPlan(ctx context.Context, plan runtime.RuntimePlan) error {
	data, err := RenderConfig(plan)
	if err != nil {
		return err
	}
	if err := WriteConfigAtomic(r.ConfigPath, data); err != nil {
		return err
	}

	r.health.ConfigHash = plan.Hash
	return nil
}

func (r *Runtime) QueryStats(ctx context.Context, reset bool) ([]nodeapi.UserTraffic, error) {
	return []nodeapi.UserTraffic{}, nil
}
