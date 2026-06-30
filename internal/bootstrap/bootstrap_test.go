package bootstrap

import (
	"context"
	"testing"

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

func setMockPanelEnv(t *testing.T) {
	t.Helper()

	t.Setenv("XNODE_MOCK_PANEL", "true")
	t.Setenv("NODE_ID", "1001")
	t.Setenv("NODE_DOMAIN", "node1.example.com")
}
