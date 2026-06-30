package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/makeausername/xnode-agent/internal/config"
	"github.com/makeausername/xnode-agent/internal/localstate"
	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/internal/panel/mock"
	"github.com/makeausername/xnode-agent/internal/panel/sspanel"
	"github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/internal/runtime/xray"
	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/internal/state"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type App struct {
	Version string
	Config  config.LocalConfig
	State   *state.Manager
	Panel   panel.Client
	Secrets secrets.Store
	Runtime runtime.Runtime
	Logger  *slog.Logger

	lastConfigHash string
}

func NewApp(version string) (*App, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, err
	}

	var panelClient panel.Client
	if cfg.MockPanel {
		if cfg.NodeID == 0 {
			cfg.NodeID = mock.DefaultNodeID
		}
		if cfg.NodeDomain == "" {
			cfg.NodeDomain = mock.DefaultDomain
		}
		panelClient = mock.NewClientForNode(cfg.NodeID, cfg.NodeDomain)
	} else {
		panelClient = sspanel.NewClient(cfg.PanelURL, "")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	return &App{
		Version: version,
		Config:  cfg,
		State:   state.NewManager(state.Uninitialized),
		Panel:   panelClient,
		Secrets: secrets.NewFileStore(cfg.DataDir),
		Runtime: xray.NewRuntime(cfg.XrayBinPath, cfg.StatePaths().XrayJSON),
		Logger:  logger,
	}, nil
}

func Run(ctx context.Context, version string) error {
	app, err := NewApp(version)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return app.Run(ctx)
}

func (a *App) Run(ctx context.Context) error {
	if err := a.SyncOnce(ctx); err != nil {
		return err
	}

	a.Logger.Info("xnode-agent started", "version", a.Version, "state", a.State.Get(), "component", "bootstrap")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.State.Set(state.Stopping)
			a.Logger.Info("xnode-agent stopped", "state", a.State.Get(), "component", "bootstrap")
			return nil
		case <-ticker.C:
			if err := a.Panel.ReportHeartbeat(ctx, a.heartbeatReport(a.Config.NodeID, a.lastConfigHash)); err != nil {
				a.State.Set(state.Degraded)
				return fmt.Errorf("report heartbeat: %w", err)
			}
			a.Logger.Info("heartbeat tick", "state", a.State.Get(), "component", "bootstrap")
		}
	}
}

func (a *App) SyncOnce(ctx context.Context) error {
	a.State.Set(state.Configured)

	if err := a.EnsureNodeToken(ctx); err != nil {
		return a.degrade("ensure node token", err)
	}

	nodeConfig, err := a.Panel.GetConfig(ctx)
	if err != nil {
		return a.degrade("get panel config", err)
	}

	users, _, err := a.Panel.GetUsers(ctx, "")
	if err != nil {
		return a.degrade("get panel users", err)
	}

	rules, _, err := a.Panel.GetDetectRules(ctx, "")
	if err != nil {
		return a.degrade("get detect rules", err)
	}

	configHash, err := localstate.HashNodeConfig(nodeConfig)
	if err != nil {
		return a.degrade("hash node config", err)
	}

	usersHash, err := localstate.HashUsers(users)
	if err != nil {
		return a.degrade("hash users", err)
	}

	realitySecret, _, err := secrets.EnsureRealitySecret(a.Secrets)
	if err != nil {
		return a.degrade("ensure reality secret", err)
	}

	if a.Runtime != nil {
		plan := runtime.RuntimePlan{
			NodeConfig: nodeConfig,
			Users:      users,
			Rules:      rules,
			Secrets:    realitySecret,
			Hash:       configHash,
		}
		if err := a.Runtime.ApplyPlan(ctx, plan); err != nil {
			return a.degrade("apply runtime plan", err)
		}
	}

	a.State.Set(state.Running)
	a.lastConfigHash = configHash

	if err := a.saveLocalState(ctx, nodeConfig, users, configHash, usersHash); err != nil {
		return a.degrade("save local state", err)
	}

	if err := a.Panel.ReportRuntime(ctx, a.runtimeReport(nodeConfig.NodeID, configHash, realitySecret)); err != nil {
		return a.degrade("report runtime", err)
	}

	if err := a.Panel.ReportHeartbeat(ctx, a.heartbeatReport(nodeConfig.NodeID, configHash)); err != nil {
		return a.degrade("report heartbeat", err)
	}

	a.Logger.Info(
		"sync completed",
		"node_id", nodeConfig.NodeID,
		"domain", nodeConfig.Domain,
		"user_count", len(users),
		"rule_count", len(rules),
		"profile_name", nodeConfig.Profile.Name,
		"state", a.State.Get(),
		"component", "bootstrap",
	)

	return nil
}

func (a *App) saveLocalState(ctx context.Context, nodeConfig nodeapi.NodeConfig, users []nodeapi.UserInfo, configHash string, usersHash string) error {
	paths := a.Config.StatePaths()
	now := time.Now().Unix()
	createdAt := now
	if existing, err := localstate.LoadAgentState(paths.AgentJSON); err == nil && existing.CreatedAt > 0 {
		createdAt = existing.CreatedAt
	}

	agentState := localstate.AgentState{
		Version:    1,
		PanelURL:   a.Config.PanelURL,
		NodeID:     nodeConfig.NodeID,
		NodeDomain: nodeConfig.Domain,
		State:      string(a.State.Get()),
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	}
	if err := localstate.SaveAgentState(paths.AgentJSON, agentState); err != nil {
		return err
	}

	usersCache := localstate.UsersCache{
		Version:   1,
		Users:     append([]nodeapi.UserInfo(nil), users...),
		UsersHash: usersHash,
		UpdatedAt: now,
	}
	if err := localstate.SaveUsersCache(paths.UsersCacheJSON, usersCache); err != nil {
		return err
	}

	health := runtime.Health{}
	if a.Runtime != nil {
		health = a.Runtime.Health(ctx)
	}
	runtimeState := localstate.RuntimeState{
		CoreVersion:    health.CoreVersion,
		AgentVersion:   a.Version,
		LastConfigHash: configHash,
		LastUsersHash:  usersHash,
		LastError:      health.LastError,
		LastApplyAt:    now,
		UpdatedAt:      now,
	}
	if err := localstate.SaveRuntimeState(paths.RuntimeJSON, runtimeState); err != nil {
		return err
	}

	return nil
}

func (a *App) runtimeReport(nodeID int64, configHash string, realitySecret secrets.RealitySecret) nodeapi.RuntimeReport {
	return nodeapi.RuntimeReport{
		NodeID:       nodeID,
		AgentVersion: a.Version,
		State:        string(a.State.Get()),
		PublicKey:    realitySecret.PublicKey,
		ShortIDs:     append([]string(nil), realitySecret.ShortIDs...),
		Capabilities: []string{"vless", "reality", "vision"},
		ConfigHash:   configHash,
	}
}

func (a *App) heartbeatReport(nodeID int64, configHash string) nodeapi.HeartbeatReport {
	return nodeapi.HeartbeatReport{
		NodeID:       nodeID,
		AgentVersion: a.Version,
		State:        string(a.State.Get()),
		ConfigHash:   configHash,
	}
}

func (a *App) degrade(action string, err error) error {
	a.State.Set(state.Degraded)
	return fmt.Errorf("sync once: %s: %w", action, err)
}
