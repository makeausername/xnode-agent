package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/makeausername/xnode-agent/internal/panel/mock"
	"github.com/makeausername/xnode-agent/internal/state"
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
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("LOG_DIR", logDir)

	return dataDir, logDir
}
