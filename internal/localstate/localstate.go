package localstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const fileMode = 0600

type AgentState struct {
	Version    int    `json:"version"`
	PanelURL   string `json:"panel_url"`
	NodeID     int64  `json:"node_id"`
	NodeDomain string `json:"node_domain"`
	State      string `json:"state"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type RuntimeState struct {
	CoreVersion    string `json:"core_version"`
	AgentVersion   string `json:"agent_version"`
	LastConfigHash string `json:"last_config_hash"`
	LastUsersHash  string `json:"last_users_hash"`
	LastError      string `json:"last_error"`
	LastApplyAt    int64  `json:"last_apply_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type UsersCache struct {
	Version   int                `json:"version"`
	Users     []nodeapi.UserInfo `json:"users"`
	UsersHash string             `json:"users_hash"`
	UsersETag string             `json:"users_etag,omitempty"`
	UpdatedAt int64              `json:"updated_at"`
}

func SaveAgentState(path string, state AgentState) error {
	if err := saveJSON(path, state); err != nil {
		return fmt.Errorf("save agent state: %w", err)
	}
	return nil
}

func LoadAgentState(path string) (AgentState, error) {
	var state AgentState
	if err := loadJSON(path, &state); err != nil {
		return AgentState{}, fmt.Errorf("load agent state: %w", err)
	}
	return state, nil
}

func SaveRuntimeState(path string, state RuntimeState) error {
	if err := saveJSON(path, state); err != nil {
		return fmt.Errorf("save runtime state: %w", err)
	}
	return nil
}

func LoadRuntimeState(path string) (RuntimeState, error) {
	var state RuntimeState
	if err := loadJSON(path, &state); err != nil {
		return RuntimeState{}, fmt.Errorf("load runtime state: %w", err)
	}
	return state, nil
}

func SaveUsersCache(path string, cache UsersCache) error {
	if err := saveJSON(path, cache); err != nil {
		return fmt.Errorf("save users cache: %w", err)
	}
	return nil
}

func LoadUsersCache(path string) (UsersCache, error) {
	var cache UsersCache
	if err := loadJSON(path, &cache); err != nil {
		return UsersCache{}, fmt.Errorf("load users cache: %w", err)
	}
	return cache, nil
}

func HashUsers(users []nodeapi.UserInfo) (string, error) {
	normalized := make([]userHash, 0, len(users))
	for _, user := range users {
		normalized = append(normalized, userHash{
			ID:             user.ID,
			UUID:           user.UUID,
			Email:          user.Email,
			SpeedLimitMbps: user.SpeedLimitMbps,
			IPLimit:        user.IPLimit,
			Enabled:        user.Enabled,
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ID != normalized[j].ID {
			return normalized[i].ID < normalized[j].ID
		}
		if normalized[i].UUID != normalized[j].UUID {
			return normalized[i].UUID < normalized[j].UUID
		}
		return normalized[i].Email < normalized[j].Email
	})

	return hashJSON(normalized)
}

func HashNodeConfig(config nodeapi.NodeConfig) (string, error) {
	normalized := nodeConfigHash{
		SchemaVersion: config.SchemaVersion,
		NodeID:        config.NodeID,
		Domain:        config.Domain,
		Profile:       config.Profile,
		Reality: nodeRealityHash{
			Target:      config.Reality.Target,
			ServerNames: append([]string(nil), config.Reality.ServerNames...),
			Fingerprint: config.Reality.Fingerprint,
		},
		Limits: config.Limits,
		Report: config.Report,
	}

	return hashJSON(normalized)
}

type userHash struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	Email          string `json:"email"`
	SpeedLimitMbps int    `json:"speed_limit_mbps"`
	IPLimit        int    `json:"ip_limit"`
	Enabled        bool   `json:"enabled"`
}

type nodeConfigHash struct {
	SchemaVersion int                  `json:"schema_version"`
	NodeID        int64                `json:"node_id"`
	Domain        string               `json:"domain"`
	Profile       nodeapi.NodeProfile  `json:"profile"`
	Reality       nodeRealityHash      `json:"reality"`
	Limits        nodeapi.NodeLimits   `json:"limits"`
	Report        nodeapi.ReportConfig `json:"report"`
}

type nodeRealityHash struct {
	Target      string   `json:"target"`
	ServerNames []string `json:"server_names"`
	Fingerprint string   `json:"fingerprint"`
}

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal hash input: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func saveJSON(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	data = append(data, '\n')
	if !json.Valid(data) {
		return errors.New("marshaled data is not valid JSON")
	}

	if err := writeFileAtomic(path, data); err != nil {
		return err
	}
	return nil
}

func loadJSON(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %q does not exist: %w", path, err)
		}
		return fmt.Errorf("read file %q: %w", path, err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("file %q contains invalid JSON", path)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("unmarshal file %q: %w", path, err)
	}

	return nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create state directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(fileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary state file mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace state file %q: %w", path, err)
	}

	return nil
}
