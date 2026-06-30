package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultDataDir = "/var/lib/xnode"
	DefaultLogDir  = "/var/log/xnode"
	DefaultXrayBin = "/usr/local/bin/xray"
)

type StatePaths struct {
	AgentJSON      string
	Token          string
	RealityJSON    string
	XrayJSON       string
	UsersCacheJSON string
	RuntimeJSON    string
	XrayLog        string
	AccessLog      string
}

type LocalConfig struct {
	PanelURL    string
	NodeID      int64
	NodeDomain  string
	DataDir     string
	LogDir      string
	XrayBinPath string
	EnrollToken string
	MockPanel   bool
}

func DefaultConfig() LocalConfig {
	return LocalConfig{
		DataDir:     DefaultDataDir,
		LogDir:      DefaultLogDir,
		XrayBinPath: DefaultXrayBin,
	}
}

func LoadFromEnv() (LocalConfig, error) {
	cfg := DefaultConfig()

	cfg.PanelURL = envString("PANEL_URL", cfg.PanelURL)
	cfg.NodeDomain = envString("NODE_DOMAIN", cfg.NodeDomain)
	cfg.DataDir = envString("DATA_DIR", cfg.DataDir)
	cfg.LogDir = envString("LOG_DIR", cfg.LogDir)
	cfg.XrayBinPath = envString("XRAY_BIN", cfg.XrayBinPath)
	cfg.EnrollToken = envString("ENROLL_TOKEN", cfg.EnrollToken)

	nodeID := strings.TrimSpace(os.Getenv("NODE_ID"))
	if nodeID != "" {
		parsed, err := strconv.ParseInt(nodeID, 10, 64)
		if err != nil {
			return LocalConfig{}, fmt.Errorf("invalid NODE_ID %q: %w", nodeID, err)
		}
		if parsed <= 0 {
			return LocalConfig{}, fmt.Errorf("invalid NODE_ID %q: must be positive", nodeID)
		}
		cfg.NodeID = parsed
	}

	mockPanel, err := parseMockPanel(os.Getenv("XNODE_MOCK_PANEL"))
	if err != nil {
		return LocalConfig{}, err
	}
	cfg.MockPanel = mockPanel

	if err := cfg.Validate(); err != nil {
		return LocalConfig{}, err
	}

	return cfg, nil
}

func (c LocalConfig) StatePaths() StatePaths {
	return StatePaths{
		AgentJSON:      statePath(c.DataDir, "agent.json"),
		Token:          statePath(c.DataDir, "token"),
		RealityJSON:    statePath(c.DataDir, "reality.json"),
		XrayJSON:       statePath(c.DataDir, "xray.json"),
		UsersCacheJSON: statePath(c.DataDir, "users.cache.json"),
		RuntimeJSON:    statePath(c.DataDir, "runtime.json"),
		XrayLog:        statePath(c.LogDir, "xray.log"),
		AccessLog:      statePath(c.LogDir, "access.log"),
	}
}

func (c LocalConfig) Validate() error {
	if strings.TrimSpace(c.DataDir) == "" {
		return errors.New("DATA_DIR is required")
	}
	if strings.TrimSpace(c.LogDir) == "" {
		return errors.New("LOG_DIR is required")
	}
	if strings.TrimSpace(c.XrayBinPath) == "" {
		return errors.New("XRAY_BIN is required")
	}
	if c.NodeID < 0 {
		return errors.New("NODE_ID must not be negative")
	}
	if !c.MockPanel {
		if strings.TrimSpace(c.PanelURL) == "" {
			return errors.New("PANEL_URL is required when XNODE_MOCK_PANEL is false")
		}
		if err := validatePanelURL(c.PanelURL); err != nil {
			return err
		}
		if c.NodeID == 0 {
			return errors.New("NODE_ID is required and must be positive when XNODE_MOCK_PANEL is false")
		}
		if strings.TrimSpace(c.NodeDomain) == "" {
			return errors.New("NODE_DOMAIN is required when XNODE_MOCK_PANEL is false")
		}
	}

	return nil
}

func validatePanelURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("PANEL_URL must be an absolute http or https URL when XNODE_MOCK_PANEL is false")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return errors.New("PANEL_URL must be an absolute http or https URL when XNODE_MOCK_PANEL is false")
	}
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseMockPanel(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, nil
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid XNODE_MOCK_PANEL %q: expected true, false, 1, or 0", value)
	}
}

func statePath(dir string, file string) string {
	return filepath.ToSlash(filepath.Join(dir, file))
}
