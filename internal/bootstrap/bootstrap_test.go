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

	"github.com/makeausername/xnode-agent/internal/config"
	"github.com/makeausername/xnode-agent/internal/localstate"
	"github.com/makeausername/xnode-agent/internal/panel/mock"
	"github.com/makeausername/xnode-agent/internal/protocol/vless"
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
