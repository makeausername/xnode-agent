package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
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

	syncMu sync.Mutex
	mu     sync.RWMutex

	lastNodeID       int64
	lastConfigHash   string
	lastReportConfig nodeapi.ReportConfig
}

type SyncResult struct {
	ConfigHash    string
	UsersHash     string
	UsersChanged  bool
	ConfigChanged bool
	Applied       bool
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

	a.logInfo("xnode-agent started", "version", a.Version, "state", a.State.Get(), "component", "bootstrap")

	heartbeatInterval, configSyncInterval, _ := a.loopIntervals()
	loopCtx, cancelLoops := context.WithCancel(ctx)
	defer cancelLoops()

	var wg sync.WaitGroup
	startLoop := func(run func(context.Context, time.Duration), interval time.Duration) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(loopCtx, interval)
		}()
	}

	startLoop(a.RunHeartbeatLoop, heartbeatInterval)
	startLoop(a.RunConfigSyncLoop, configSyncInterval)

	<-ctx.Done()
	a.State.Set(state.Stopping)
	cancelLoops()
	wg.Wait()
	a.State.Set(state.Stopping)
	a.logInfo("xnode-agent stopped", "state", a.State.Get(), "component", "bootstrap")

	return nil
}

func (a *App) SyncOnce(ctx context.Context) error {
	_, err := a.SyncOnceResult(ctx)
	return err
}

func (a *App) SyncOnceResult(ctx context.Context) (SyncResult, error) {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()

	result := SyncResult{}
	a.State.Set(state.Configured)

	if err := a.EnsureNodeToken(ctx); err != nil {
		return result, a.degrade("ensure node token", err)
	}

	paths := a.Config.StatePaths()
	previousRuntime, hasRuntimeState, err := loadPreviousRuntimeState(paths.RuntimeJSON)
	if err != nil {
		return result, a.degrade("load previous runtime state", err)
	}

	previousUsersCache, hasUsersCache, err := loadPreviousUsersCache(paths.UsersCacheJSON)
	if err != nil {
		return result, a.degrade("load previous users cache", err)
	}

	nodeConfig, err := a.Panel.GetConfig(ctx)
	if err != nil {
		return result, a.degrade("get panel config", err)
	}

	configHash, err := localstate.HashNodeConfig(nodeConfig)
	if err != nil {
		return result, a.degrade("hash node config", err)
	}
	result.ConfigHash = configHash
	result.ConfigChanged = !hasRuntimeState || previousRuntime.LastConfigHash != configHash

	previousUsersETag := ""
	if hasUsersCache {
		previousUsersETag = previousUsersCache.UsersETag
	}
	users, usersETag, err := a.Panel.GetUsers(ctx, previousUsersETag)
	if err != nil {
		return result, a.degrade("get panel users", err)
	}
	if usersETag == "" {
		usersETag = previousUsersETag
	}

	usersCacheChanged := !hasUsersCache
	usersHash := ""
	if users == nil {
		if !hasUsersCache {
			return result, a.degrade("get panel users", errors.New("panel returned not modified without users cache"))
		}
		users = append([]nodeapi.UserInfo(nil), previousUsersCache.Users...)
		usersHash = previousUsersCache.UsersHash
		if usersHash == "" {
			usersHash, err = localstate.HashUsers(users)
			if err != nil {
				return result, a.degrade("hash cached users", err)
			}
		}
		result.UsersChanged = false
	} else {
		users = append([]nodeapi.UserInfo(nil), users...)
		usersHash, err = localstate.HashUsers(users)
		if err != nil {
			return result, a.degrade("hash users", err)
		}
		result.UsersChanged = !hasRuntimeState || previousRuntime.LastUsersHash != usersHash
		usersCacheChanged = usersCacheChanged ||
			result.UsersChanged ||
			(usersETag != "" && usersETag != previousUsersCache.UsersETag)
	}
	result.UsersHash = usersHash

	rules, _, err := a.Panel.GetDetectRules(ctx, "")
	if err != nil {
		return result, a.degrade("get detect rules", err)
	}

	realitySecret, _, err := secrets.EnsureRealitySecret(a.Secrets)
	if err != nil {
		return result, a.degrade("ensure reality secret", err)
	}

	xrayExists, err := fileExists(paths.XrayJSON)
	if err != nil {
		return result, a.degrade("check xray config", err)
	}

	shouldApply := a.Runtime != nil && (result.ConfigChanged || result.UsersChanged || !xrayExists)
	if shouldApply {
		plan := runtime.RuntimePlan{
			NodeConfig: nodeConfig,
			Users:      users,
			Rules:      rules,
			Secrets:    realitySecret,
			Hash:       configHash,
		}
		if err := a.Runtime.ApplyPlan(ctx, plan); err != nil {
			return result, a.degrade("apply runtime plan", err)
		}
		result.Applied = true
	}

	a.State.Set(state.Running)
	a.setLastSyncSnapshot(nodeConfig.NodeID, configHash, nodeConfig.Report)

	if err := a.saveLocalState(ctx, nodeConfig, users, usersETag, usersCacheChanged, configHash, usersHash, previousRuntime, result.Applied); err != nil {
		return result, a.degrade("save local state", err)
	}

	if err := a.Panel.ReportRuntime(ctx, a.runtimeReport(nodeConfig.NodeID, configHash, realitySecret)); err != nil {
		return result, a.degrade("report runtime", err)
	}

	if err := a.ReportHeartbeat(ctx); err != nil {
		return result, a.degrade("report heartbeat", err)
	}

	a.logInfo(
		"sync completed",
		"node_id", nodeConfig.NodeID,
		"domain", nodeConfig.Domain,
		"user_count", len(users),
		"rule_count", len(rules),
		"profile_name", nodeConfig.Profile.Name,
		"state", a.State.Get(),
		"config_changed", result.ConfigChanged,
		"users_changed", result.UsersChanged,
		"applied", result.Applied,
		"component", "bootstrap",
	)

	return result, nil
}

func (a *App) saveLocalState(ctx context.Context, nodeConfig nodeapi.NodeConfig, users []nodeapi.UserInfo, usersETag string, saveUsersCache bool, configHash string, usersHash string, previousRuntime localstate.RuntimeState, applied bool) error {
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

	if saveUsersCache {
		usersCache := localstate.UsersCache{
			Version:   1,
			Users:     append([]nodeapi.UserInfo(nil), users...),
			UsersHash: usersHash,
			UsersETag: usersETag,
			UpdatedAt: now,
		}
		if err := localstate.SaveUsersCache(paths.UsersCacheJSON, usersCache); err != nil {
			return err
		}
	}

	health := runtime.Health{}
	if a.Runtime != nil {
		health = a.Runtime.Health(ctx)
	}
	lastApplyAt := previousRuntime.LastApplyAt
	if applied {
		lastApplyAt = now
	}
	runtimeState := localstate.RuntimeState{
		CoreVersion:    health.CoreVersion,
		AgentVersion:   a.Version,
		LastConfigHash: configHash,
		LastUsersHash:  usersHash,
		LastError:      health.LastError,
		LastApplyAt:    lastApplyAt,
		UpdatedAt:      now,
	}
	if err := localstate.SaveRuntimeState(paths.RuntimeJSON, runtimeState); err != nil {
		return err
	}

	return nil
}

func loadPreviousRuntimeState(path string) (localstate.RuntimeState, bool, error) {
	runtimeState, err := localstate.LoadRuntimeState(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localstate.RuntimeState{}, false, nil
		}
		return localstate.RuntimeState{}, false, err
	}
	return runtimeState, true, nil
}

func loadPreviousUsersCache(path string) (localstate.UsersCache, bool, error) {
	usersCache, err := localstate.LoadUsersCache(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localstate.UsersCache{}, false, nil
		}
		return localstate.UsersCache{}, false, err
	}
	return usersCache, true, nil
}

func fileExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

func (a *App) heartbeatReport(nodeID int64, configHash string, health runtime.Health) nodeapi.HeartbeatReport {
	return nodeapi.HeartbeatReport{
		NodeID:       nodeID,
		AgentVersion: a.Version,
		CoreVersion:  health.CoreVersion,
		State:        string(a.State.Get()),
		LastError:    health.LastError,
		ConfigHash:   configHash,
	}
}

func (a *App) degrade(action string, err error) error {
	a.State.Set(state.Degraded)
	return fmt.Errorf("sync once: %s: %w", action, err)
}
