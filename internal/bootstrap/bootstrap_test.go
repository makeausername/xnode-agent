package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/makeausername/xnode-agent/internal/config"
	"github.com/makeausername/xnode-agent/internal/localstate"
	"github.com/makeausername/xnode-agent/internal/panel/mock"
	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	runtimepkg "github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/internal/state"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestNewAppWithMockPanel(t *testing.T) {
	setMockPanelEnv(t)

	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if app.Config.NodeID != 1001 {
		t.Fatalf("NodeID = %d, want 1001", app.Config.NodeID)
	}
	if app.Config.NodeDomain != "node1.example.com" {
		t.Fatalf("NodeDomain = %q, want node1.example.com", app.Config.NodeDomain)
	}
	if app.Panel == nil {
		t.Fatal("Panel = nil")
	}
}

func TestSyncOnceWithMockPanel(t *testing.T) {
	setMockPanelEnv(t)

	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
}

func TestEnsureNodeTokenMockPanelNoopAndSyncOnceSucceeds(t *testing.T) {
	dataDir, _ := setMockPanelEnv(t)

	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.EnsureNodeToken(context.Background()); err != nil {
		t.Fatalf("EnsureNodeToken() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "token")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(token) error = %v, want missing token file", err)
	}

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "token")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(token) error = %v, want missing token file", err)
	}
}

func TestSyncOnceSetsStateRunning(t *testing.T) {
	setMockPanelEnv(t)

	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if got := app.State.Get(); got != state.Running {
		t.Fatalf("state = %s, want %s", got, state.Running)
	}
}

func TestSyncOnceSkipsApplyWhenConfigAndUsersUnchanged(t *testing.T) {
	app := newMockBootstrapTestApp(t)
	runtime := newCountingRuntime(app.Config.StatePaths().XrayJSON)
	app.Runtime = runtime

	first, err := app.SyncOnceResult(context.Background())
	if err != nil {
		t.Fatalf("first SyncOnceResult() error = %v", err)
	}
	if !first.Applied {
		t.Fatal("first SyncOnceResult().Applied = false, want true")
	}
	if runtime.ApplyCount() != 1 {
		t.Fatalf("ApplyCount after first sync = %d, want 1", runtime.ApplyCount())
	}
	if _, err := os.Stat(app.Config.StatePaths().XrayJSON); err != nil {
		t.Fatalf("Stat(xray.json) error = %v", err)
	}

	firstState, err := localstate.LoadRuntimeState(app.Config.StatePaths().RuntimeJSON)
	if err != nil {
		t.Fatalf("LoadRuntimeState() after first sync error = %v", err)
	}

	second, err := app.SyncOnceResult(context.Background())
	if err != nil {
		t.Fatalf("second SyncOnceResult() error = %v", err)
	}
	if second.Applied {
		t.Fatal("second SyncOnceResult().Applied = true, want false")
	}
	if second.ConfigChanged {
		t.Fatal("second SyncOnceResult().ConfigChanged = true, want false")
	}
	if second.UsersChanged {
		t.Fatal("second SyncOnceResult().UsersChanged = true, want false")
	}
	if runtime.ApplyCount() != 1 {
		t.Fatalf("ApplyCount after second sync = %d, want 1", runtime.ApplyCount())
	}
	if got := app.State.Get(); got != state.Running {
		t.Fatalf("state after no-op sync = %s, want %s", got, state.Running)
	}

	secondState, err := localstate.LoadRuntimeState(app.Config.StatePaths().RuntimeJSON)
	if err != nil {
		t.Fatalf("LoadRuntimeState() after second sync error = %v", err)
	}
	if secondState.LastApplyAt != firstState.LastApplyAt {
		t.Fatalf("LastApplyAt after no-op sync = %d, want %d", secondState.LastApplyAt, firstState.LastApplyAt)
	}

	mockPanel, ok := app.Panel.(*mock.Client)
	if !ok {
		t.Fatalf("Panel type = %T, want *mock.Client", app.Panel)
	}
	report, ok := mockPanel.LastHeartbeatReport()
	if !ok {
		t.Fatal("LastHeartbeatReport() ok = false, want true")
	}
	if report.ConfigHash != second.ConfigHash {
		t.Fatalf("Heartbeat ConfigHash = %q, want %q", report.ConfigHash, second.ConfigHash)
	}
}

func TestSyncOnceRebuildsMissingUsersCacheWithoutApply(t *testing.T) {
	app := newMockBootstrapTestApp(t)
	runtime := newCountingRuntime(app.Config.StatePaths().XrayJSON)
	app.Runtime = runtime

	if _, err := app.SyncOnceResult(context.Background()); err != nil {
		t.Fatalf("first SyncOnceResult() error = %v", err)
	}
	if err := os.Remove(app.Config.StatePaths().UsersCacheJSON); err != nil {
		t.Fatalf("Remove(users.cache.json) error = %v", err)
	}

	result, err := app.SyncOnceResult(context.Background())
	if err != nil {
		t.Fatalf("second SyncOnceResult() error = %v", err)
	}
	if result.Applied {
		t.Fatal("SyncOnceResult().Applied = true, want false")
	}
	if runtime.ApplyCount() != 1 {
		t.Fatalf("ApplyCount = %d, want 1", runtime.ApplyCount())
	}

	cache, err := localstate.LoadUsersCache(app.Config.StatePaths().UsersCacheJSON)
	if err != nil {
		t.Fatalf("LoadUsersCache() error = %v", err)
	}
	if len(cache.Users) == 0 {
		t.Fatal("rebuilt users.cache.json users is empty")
	}
	if cache.UsersHash != result.UsersHash {
		t.Fatalf("rebuilt users hash = %q, want %q", cache.UsersHash, result.UsersHash)
	}
	if cache.UsersETag == "" {
		t.Fatal("rebuilt users.cache.json users_etag is empty")
	}
}

func TestSyncOnceAppliesWhenXrayConfigMissingEvenWhenHashesMatch(t *testing.T) {
	app := newMockBootstrapTestApp(t)
	runtime := newCountingRuntime(app.Config.StatePaths().XrayJSON)
	app.Runtime = runtime

	if _, err := app.SyncOnceResult(context.Background()); err != nil {
		t.Fatalf("first SyncOnceResult() error = %v", err)
	}
	if err := os.Remove(app.Config.StatePaths().XrayJSON); err != nil {
		t.Fatalf("Remove(xray.json) error = %v", err)
	}

	result, err := app.SyncOnceResult(context.Background())
	if err != nil {
		t.Fatalf("second SyncOnceResult() error = %v", err)
	}
	if !result.Applied {
		t.Fatal("SyncOnceResult().Applied = false, want true when xray.json is missing")
	}
	if result.ConfigChanged {
		t.Fatal("SyncOnceResult().ConfigChanged = true, want false")
	}
	if result.UsersChanged {
		t.Fatal("SyncOnceResult().UsersChanged = true, want false")
	}
	if runtime.ApplyCount() != 2 {
		t.Fatalf("ApplyCount = %d, want 2", runtime.ApplyCount())
	}
	if _, err := os.Stat(app.Config.StatePaths().XrayJSON); err != nil {
		t.Fatalf("Stat(xray.json) after reapply error = %v", err)
	}
}

func TestEnsureNodeTokenRealPanelUsesExistingToken(t *testing.T) {
	dataDir := t.TempDir()
	store := secrets.NewFileStore(dataDir)
	if err := store.SaveToken("stored-node-token"); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	panel := &fakeEnrollPanel{
		enrollResponse: nodeapi.EnrollResponse{NodeToken: "issued-node-token"},
	}
	app := newRealTestApp(t, dataDir, panel)

	if err := app.EnsureNodeToken(context.Background()); err != nil {
		t.Fatalf("EnsureNodeToken() error = %v", err)
	}
	if panel.enrollCalls != 0 {
		t.Fatalf("Enroll calls = %d, want 0", panel.enrollCalls)
	}
	if panel.token != "stored-node-token" {
		t.Fatalf("panel token = %q, want stored token", panel.token)
	}
}

func TestEnsureNodeTokenRealPanelMissingTokenRequiresEnrollToken(t *testing.T) {
	panel := &fakeEnrollPanel{}
	app := newRealTestApp(t, t.TempDir(), panel)
	app.Config.EnrollToken = ""

	err := app.EnsureNodeToken(context.Background())
	if err == nil {
		t.Fatal("EnsureNodeToken() error = nil, want ENROLL_TOKEN error")
	}
	if !strings.Contains(err.Error(), "ENROLL_TOKEN is required") {
		t.Fatalf("EnsureNodeToken() error = %q, want ENROLL_TOKEN context", err.Error())
	}
	if panel.enrollCalls != 0 {
		t.Fatalf("Enroll calls = %d, want 0", panel.enrollCalls)
	}
}

func TestEnsureNodeTokenRealPanelEnrollsAndSavesToken(t *testing.T) {
	const enrollToken = "bootstrap-enroll-secret"
	const nodeToken = "issued-node-token"

	dataDir := t.TempDir()
	panel := &fakeEnrollPanel{
		enrollResponse: nodeapi.EnrollResponse{NodeToken: nodeToken},
	}
	app := newRealTestApp(t, dataDir, panel)
	app.Config.EnrollToken = enrollToken

	if err := app.EnsureNodeToken(context.Background()); err != nil {
		t.Fatalf("EnsureNodeToken() error = %v", err)
	}
	if panel.enrollCalls != 1 {
		t.Fatalf("Enroll calls = %d, want 1", panel.enrollCalls)
	}
	if panel.enrollAuthToken != enrollToken {
		t.Fatalf("Enroll used token = %q, want ENROLL_TOKEN", panel.enrollAuthToken)
	}
	if panel.token != nodeToken {
		t.Fatalf("panel token = %q, want issued node token", panel.token)
	}

	stored, err := secrets.NewFileStore(dataDir).LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if stored != nodeToken {
		t.Fatalf("stored token = %q, want issued node token", stored)
	}
}

func TestEnsureNodeTokenRealPanelRejectsEmptyEnrollResponse(t *testing.T) {
	const enrollToken = "bootstrap-enroll-secret"

	panel := &fakeEnrollPanel{
		enrollResponse: nodeapi.EnrollResponse{NodeToken: "  "},
	}
	app := newRealTestApp(t, t.TempDir(), panel)
	app.Config.EnrollToken = enrollToken

	err := app.EnsureNodeToken(context.Background())
	if err == nil {
		t.Fatal("EnsureNodeToken() error = nil, want empty node_token error")
	}
	if !strings.Contains(err.Error(), "empty node_token") {
		t.Fatalf("EnsureNodeToken() error = %q, want empty node_token context", err.Error())
	}
	if strings.Contains(err.Error(), enrollToken) {
		t.Fatalf("EnsureNodeToken() error leaked ENROLL_TOKEN: %q", err.Error())
	}
}

func TestSyncOnceWithRealPanelEnrollsBeforeGetConfig(t *testing.T) {
	const nodeToken = "issued-node-token"

	dataDir := t.TempDir()
	panel := &fakeEnrollPanel{
		enrollResponse:        nodeapi.EnrollResponse{NodeToken: nodeToken},
		requireTokenForConfig: nodeToken,
	}
	app := newRealTestApp(t, dataDir, panel)
	app.Config.EnrollToken = "bootstrap-enroll-secret"

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if panel.enrollCalls != 1 {
		t.Fatalf("Enroll calls = %d, want 1", panel.enrollCalls)
	}
	if panel.configCalls != 1 {
		t.Fatalf("GetConfig calls = %d, want 1", panel.configCalls)
	}
	if got := app.State.Get(); got != state.Running {
		t.Fatalf("state = %s, want %s", got, state.Running)
	}
}

func TestSyncOnceWithMockPanelCreatesRealityAndReportsRuntime(t *testing.T) {
	dataDir, _ := setMockPanelEnv(t)

	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}
	if got := app.State.Get(); got != state.Running {
		t.Fatalf("state = %s, want %s", got, state.Running)
	}

	realityPath := filepath.Join(dataDir, "reality.json")
	if _, err := os.Stat(realityPath); err != nil {
		t.Fatalf("Stat(reality.json) error = %v", err)
	}

	xrayPath := filepath.Join(dataDir, "xray.json")
	xrayData, err := os.ReadFile(xrayPath)
	if err != nil {
		t.Fatalf("ReadFile(xray.json) error = %v", err)
	}
	if !json.Valid(xrayData) {
		t.Fatalf("xray.json is not valid JSON: %s", xrayData)
	}

	agentPath := filepath.Join(dataDir, "agent.json")
	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf("Stat(agent.json) error = %v", err)
	}

	runtimePath := filepath.Join(dataDir, "runtime.json")
	runtimeState, err := localstate.LoadRuntimeState(runtimePath)
	if err != nil {
		t.Fatalf("LoadRuntimeState(runtime.json) error = %v", err)
	}
	if runtimeState.LastConfigHash == "" {
		t.Fatal("runtime.json last_config_hash is empty")
	}
	if runtimeState.LastUsersHash == "" {
		t.Fatal("runtime.json last_users_hash is empty")
	}

	usersCachePath := filepath.Join(dataDir, "users.cache.json")
	usersCache, err := localstate.LoadUsersCache(usersCachePath)
	if err != nil {
		t.Fatalf("LoadUsersCache(users.cache.json) error = %v", err)
	}
	if len(usersCache.Users) == 0 {
		t.Fatal("users.cache.json users is empty")
	}

	secret, err := app.Secrets.LoadReality()
	if err != nil {
		t.Fatalf("LoadReality() error = %v", err)
	}
	if secret.PublicKey == "" {
		t.Fatal("reality.json public_key is empty")
	}
	if len(secret.ShortIDs) == 0 {
		t.Fatal("reality.json short_ids is empty")
	}

	mockPanel, ok := app.Panel.(*mock.Client)
	if !ok {
		t.Fatalf("Panel type = %T, want *mock.Client", app.Panel)
	}
	report, ok := mockPanel.LastRuntimeReport()
	if !ok {
		t.Fatal("ReportRuntime was not called")
	}
	if report.NodeID != 1001 {
		t.Fatalf("RuntimeReport.NodeID = %d, want 1001", report.NodeID)
	}
	if report.AgentVersion != "test-version" {
		t.Fatalf("RuntimeReport.AgentVersion = %q, want test-version", report.AgentVersion)
	}
	if report.State != string(state.Running) {
		t.Fatalf("RuntimeReport.State = %q, want %q", report.State, state.Running)
	}
	if report.PublicKey != secret.PublicKey {
		t.Fatalf("RuntimeReport.PublicKey = %q, want reality.json public_key", report.PublicKey)
	}
	if len(report.ShortIDs) != len(secret.ShortIDs) || report.ShortIDs[0] != secret.ShortIDs[0] {
		t.Fatalf("RuntimeReport.ShortIDs = %#v, want %#v", report.ShortIDs, secret.ShortIDs)
	}
	if report.ConfigHash != runtimeState.LastConfigHash {
		t.Fatalf("RuntimeReport.ConfigHash = %q, want %q", report.ConfigHash, runtimeState.LastConfigHash)
	}
	wantCapabilities := []string{"vless", "reality", "vision"}
	if len(report.Capabilities) != len(wantCapabilities) {
		t.Fatalf("RuntimeReport.Capabilities = %#v, want %#v", report.Capabilities, wantCapabilities)
	}
	for i := range wantCapabilities {
		if report.Capabilities[i] != wantCapabilities[i] {
			t.Fatalf("RuntimeReport.Capabilities = %#v, want %#v", report.Capabilities, wantCapabilities)
		}
	}
}

func TestReportHeartbeatSucceedsAfterSyncOnce(t *testing.T) {
	app := newMockBootstrapTestApp(t)

	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}

	mockPanel, ok := app.Panel.(*mock.Client)
	if !ok {
		t.Fatalf("Panel type = %T, want *mock.Client", app.Panel)
	}
	before := mockPanel.HeartbeatReportCount()

	if err := app.ReportHeartbeat(context.Background()); err != nil {
		t.Fatalf("ReportHeartbeat() error = %v", err)
	}
	if got := mockPanel.HeartbeatReportCount(); got != before+1 {
		t.Fatalf("HeartbeatReportCount = %d, want %d", got, before+1)
	}

	report, ok := mockPanel.LastHeartbeatReport()
	if !ok {
		t.Fatal("LastHeartbeatReport() ok = false, want true")
	}
	if report.NodeID != 1001 {
		t.Fatalf("HeartbeatReport.NodeID = %d, want 1001", report.NodeID)
	}
	if report.AgentVersion != "test-version" {
		t.Fatalf("HeartbeatReport.AgentVersion = %q, want test-version", report.AgentVersion)
	}
	if report.State != string(state.Running) {
		t.Fatalf("HeartbeatReport.State = %q, want %q", report.State, state.Running)
	}
	if report.ConfigHash == "" {
		t.Fatal("HeartbeatReport.ConfigHash is empty")
	}
}

func TestRunHeartbeatLoopExitsWhenContextCanceled(t *testing.T) {
	app := newMockBootstrapTestApp(t)
	if err := app.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}

	runLoopUntilCanceled(t, app.RunHeartbeatLoop)
}

func TestRunConfigSyncLoopExitsWhenContextCanceled(t *testing.T) {
	app := newMockBootstrapTestApp(t)

	runLoopUntilCanceled(t, app.RunConfigSyncLoop)
}

func TestRunUserSyncLoopExitsWhenContextCanceled(t *testing.T) {
	app := newMockBootstrapTestApp(t)

	runLoopUntilCanceled(t, app.RunUserSyncLoop)
}

func newMockBootstrapTestApp(t *testing.T) *App {
	t.Helper()

	setMockPanelEnv(t)
	app, err := NewApp("test-version")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	return app
}

func runLoopUntilCanceled(t *testing.T, run func(context.Context, time.Duration)) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	panicCh := make(chan any, 1)

	go func() {
		defer close(done)
		defer func() {
			if recovered := recover(); recovered != nil {
				panicCh <- recovered
			}
		}()
		run(ctx, 10*time.Millisecond)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loop did not exit after context cancellation")
	}

	select {
	case recovered := <-panicCh:
		t.Fatalf("loop panicked: %v", recovered)
	default:
	}
}

type countingRuntime struct {
	configPath string
	applyCount int
	health     runtimepkg.Health
}

func newCountingRuntime(configPath string) *countingRuntime {
	return &countingRuntime{
		configPath: configPath,
	}
}

func (r *countingRuntime) Start(ctx context.Context) error {
	return nil
}

func (r *countingRuntime) Stop(ctx context.Context) error {
	return nil
}

func (r *countingRuntime) Health(ctx context.Context) runtimepkg.Health {
	return r.health
}

func (r *countingRuntime) ApplyPlan(ctx context.Context, plan runtimepkg.RuntimePlan) error {
	r.applyCount++
	r.health.ConfigHash = plan.Hash

	if err := os.MkdirAll(filepath.Dir(r.configPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(r.configPath, []byte("{\"applied\":true}\n"), 0600)
}

func (r *countingRuntime) QueryStats(ctx context.Context, reset bool) ([]nodeapi.UserTraffic, error) {
	return []nodeapi.UserTraffic{}, nil
}

func (r *countingRuntime) ApplyCount() int {
	return r.applyCount
}

func setMockPanelEnv(t *testing.T) (string, string) {
	t.Helper()

	dataDir := t.TempDir()
	logDir := t.TempDir()

	t.Setenv("XNODE_MOCK_PANEL", "true")
	t.Setenv("NODE_ID", "1001")
	t.Setenv("NODE_DOMAIN", "node1.example.com")
	t.Setenv("ENROLL_TOKEN", "")
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("LOG_DIR", logDir)

	return dataDir, logDir
}

type fakeEnrollPanel struct {
	token                 string
	enrollAuthToken       string
	enrollCalls           int
	configCalls           int
	requireTokenForConfig string
	enrollResponse        nodeapi.EnrollResponse
	enrollErr             error
}

func (p *fakeEnrollPanel) SetToken(token string) {
	p.token = token
}

func (p *fakeEnrollPanel) Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error) {
	p.enrollCalls++
	p.enrollAuthToken = p.token
	if p.enrollErr != nil {
		return nodeapi.EnrollResponse{}, p.enrollErr
	}
	return p.enrollResponse, nil
}

func (p *fakeEnrollPanel) GetConfig(ctx context.Context) (nodeapi.NodeConfig, error) {
	p.configCalls++
	if p.requireTokenForConfig != "" && p.token != p.requireTokenForConfig {
		return nodeapi.NodeConfig{}, fmt.Errorf("GetConfig called with token %q, want issued node token", p.token)
	}
	return vless.DefaultNodeConfig(1001, "node1.example.com"), nil
}

func (p *fakeEnrollPanel) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	return []nodeapi.UserInfo{
		{
			ID:             1,
			UUID:           "11111111-1111-4111-8111-111111111111",
			Email:          "real-mode-user@example.com",
			SpeedLimitMbps: 100,
			IPLimit:        2,
			Enabled:        true,
			UpdatedAt:      1700000000,
		},
	}, "users-v1", nil
}

func (p *fakeEnrollPanel) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	return []nodeapi.DetectRule{}, "rules-v1", nil
}

func (p *fakeEnrollPanel) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	return nil
}

func (p *fakeEnrollPanel) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	return nil
}

func (p *fakeEnrollPanel) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	return nil
}

func (p *fakeEnrollPanel) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	return nil
}

func newRealTestApp(t *testing.T, dataDir string, panel *fakeEnrollPanel) *App {
	t.Helper()

	return &App{
		Version: "test-version",
		Config: config.LocalConfig{
			PanelURL:    "https://panel.example.com",
			NodeID:      1001,
			NodeDomain:  "node1.example.com",
			DataDir:     dataDir,
			LogDir:      t.TempDir(),
			EnrollToken: "",
			MockPanel:   false,
		},
		State:   state.NewManager(state.Uninitialized),
		Panel:   panel,
		Secrets: secrets.NewFileStore(dataDir),
		Runtime: nil,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})),
	}
}
