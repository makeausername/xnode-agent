package bootstrap

import (
	"context"
	"fmt"
	"time"

	runtimepkg "github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/internal/state"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	defaultHeartbeatInterval  = 30 * time.Second
	defaultConfigSyncInterval = 60 * time.Second
	defaultUserSyncInterval   = 60 * time.Second
)

func (a *App) runConfigSyncLoop(ctx context.Context) {
	a.RunConfigSyncLoop(ctx, defaultConfigSyncInterval)
}

func (a *App) runUserSyncLoop(ctx context.Context) {
	a.RunUserSyncLoop(ctx, defaultUserSyncInterval)
}

func (a *App) runHeartbeatLoop(ctx context.Context) {
	a.RunHeartbeatLoop(ctx, defaultHeartbeatInterval)
}

func (a *App) RunConfigSyncLoop(ctx context.Context, interval time.Duration) {
	a.runTickerLoop(ctx, "config-sync", interval, defaultConfigSyncInterval, func(ctx context.Context) error {
		return a.SyncOnce(ctx)
	})
}

func (a *App) RunUserSyncLoop(ctx context.Context, interval time.Duration) {
	a.runTickerLoop(ctx, "user-sync", interval, defaultUserSyncInterval, func(ctx context.Context) error {
		return a.SyncOnce(ctx)
	})
}

func (a *App) RunHeartbeatLoop(ctx context.Context, interval time.Duration) {
	a.runTickerLoop(ctx, "heartbeat", interval, defaultHeartbeatInterval, func(ctx context.Context) error {
		return a.ReportHeartbeat(ctx)
	})
}

func (a *App) ReportHeartbeat(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	health := runtimepkg.Health{}
	if a.Runtime != nil {
		health = a.Runtime.Health(ctx)
	}

	nodeID, configHash, _ := a.lastSyncSnapshot()
	if nodeID == 0 {
		nodeID = a.Config.NodeID
	}
	if health.ConfigHash != "" {
		configHash = health.ConfigHash
	}

	if err := a.Panel.ReportHeartbeat(ctx, a.heartbeatReport(nodeID, configHash, health)); err != nil {
		return fmt.Errorf("panel report heartbeat: %w", err)
	}

	return nil
}

func (a *App) runTickerLoop(ctx context.Context, component string, interval time.Duration, fallback time.Duration, tick func(context.Context) error) {
	if ctx == nil {
		ctx = context.Background()
	}

	interval = normalizeLoopInterval(interval, fallback)
	a.logInfo("loop started", "component", component, "interval", interval.String(), "state", a.State.Get())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logInfo("loop stopped", "component", component, "state", a.State.Get())
			return
		case <-ticker.C:
			if err := tick(ctx); err != nil {
				if ctx.Err() != nil {
					a.logInfo("loop stopped", "component", component, "state", a.State.Get())
					return
				}
				a.State.Set(state.Degraded)
				a.logError("loop tick failed", "component", component, "state", a.State.Get())
				continue
			}
			a.logInfo("loop tick completed", "component", component, "state", a.State.Get())
		}
	}
}

func (a *App) loopIntervals() (time.Duration, time.Duration, time.Duration) {
	_, _, reportConfig := a.lastSyncSnapshot()

	return intervalFromSeconds(reportConfig.HeartbeatIntervalSec, defaultHeartbeatInterval),
		intervalFromSeconds(reportConfig.UserSyncIntervalSec, defaultConfigSyncInterval),
		intervalFromSeconds(reportConfig.UserSyncIntervalSec, defaultUserSyncInterval)
}

func intervalFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	interval := normalizeLoopInterval(time.Duration(seconds)*time.Second, fallback)
	if interval < fallback {
		return fallback
	}
	return interval
}

func normalizeLoopInterval(interval time.Duration, fallback time.Duration) time.Duration {
	if interval <= 0 {
		return fallback
	}
	return interval
}

func (a *App) setLastSyncSnapshot(nodeID int64, configHash string, reportConfig nodeapi.ReportConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.lastNodeID = nodeID
	a.lastConfigHash = configHash
	a.lastReportConfig = reportConfig
	if a.Reporter != nil {
		a.Reporter.NodeID = nodeID
	}
}

func (a *App) lastSyncSnapshot() (int64, string, nodeapi.ReportConfig) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.lastNodeID, a.lastConfigHash, a.lastReportConfig
}

func (a *App) logInfo(message string, args ...any) {
	if a.Logger == nil {
		return
	}
	a.Logger.Info(message, args...)
}

func (a *App) logError(message string, args ...any) {
	if a.Logger == nil {
		return
	}
	a.Logger.Error(message, args...)
}
