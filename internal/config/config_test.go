package config

import "testing"

func TestDefaultConfigStatePaths(t *testing.T) {
	cfg := DefaultConfig()
	paths := cfg.StatePaths()

	if cfg.DataDir != DefaultDataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, DefaultDataDir)
	}
	if cfg.LogDir != DefaultLogDir {
		t.Fatalf("LogDir = %q, want %q", cfg.LogDir, DefaultLogDir)
	}
	if cfg.XrayBinPath != DefaultXrayBin {
		t.Fatalf("XrayBinPath = %q, want %q", cfg.XrayBinPath, DefaultXrayBin)
	}

	expected := StatePaths{
		AgentJSON:      "/var/lib/xnode/agent.json",
		Token:          "/var/lib/xnode/token",
		RealityJSON:    "/var/lib/xnode/reality.json",
		XrayJSON:       "/var/lib/xnode/xray.json",
		UsersCacheJSON: "/var/lib/xnode/users.cache.json",
		RuntimeJSON:    "/var/lib/xnode/runtime.json",
		XrayLog:        "/var/log/xnode/xray.log",
		AccessLog:      "/var/log/xnode/access.log",
	}

	if paths != expected {
		t.Fatalf("StatePaths() = %#v, want %#v", paths, expected)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PANEL_URL", "https://panel.example.com")
	t.Setenv("NODE_ID", "42")
	t.Setenv("NODE_DOMAIN", "node.example.com")
	t.Setenv("DATA_DIR", "/tmp/xnode-data")
	t.Setenv("LOG_DIR", "/tmp/xnode-log")
	t.Setenv("XRAY_BIN", "/opt/xray")
	t.Setenv("ENROLL_TOKEN", "enroll-token")
	t.Setenv("XNODE_MOCK_PANEL", "")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.PanelURL != "https://panel.example.com" {
		t.Fatalf("PanelURL = %q", cfg.PanelURL)
	}
	if cfg.NodeID != 42 {
		t.Fatalf("NodeID = %d, want 42", cfg.NodeID)
	}
	if cfg.NodeDomain != "node.example.com" {
		t.Fatalf("NodeDomain = %q", cfg.NodeDomain)
	}
	if cfg.DataDir != "/tmp/xnode-data" {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
	if cfg.LogDir != "/tmp/xnode-log" {
		t.Fatalf("LogDir = %q", cfg.LogDir)
	}
	if cfg.XrayBinPath != "/opt/xray" {
		t.Fatalf("XrayBinPath = %q", cfg.XrayBinPath)
	}
	if cfg.EnrollToken != "enroll-token" {
		t.Fatalf("EnrollToken = %q", cfg.EnrollToken)
	}
	if cfg.MockPanel {
		t.Fatalf("MockPanel = true, want false")
	}
}

func TestLoadFromEnvInvalidNodeID(t *testing.T) {
	t.Setenv("NODE_ID", "not-an-int")
	t.Setenv("XNODE_MOCK_PANEL", "")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want NODE_ID parse error")
	}
}

func TestLoadFromEnvMockPanelMode(t *testing.T) {
	t.Setenv("NODE_ID", "")
	t.Setenv("XNODE_MOCK_PANEL", "true")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if !cfg.MockPanel {
		t.Fatalf("MockPanel = false, want true")
	}

	t.Setenv("XNODE_MOCK_PANEL", "1")

	cfg, err = LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if !cfg.MockPanel {
		t.Fatalf("MockPanel = false, want true")
	}
}
