package localstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestSaveLoadAgentState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "agent.json")
	state := AgentState{
		Version:    1,
		PanelURL:   "https://panel.example.com",
		NodeID:     1001,
		NodeDomain: "node1.example.com",
		State:      "running",
		CreatedAt:  1700000000,
		UpdatedAt:  1700000100,
	}

	if err := SaveAgentState(path, state); err != nil {
		t.Fatalf("SaveAgentState() error = %v", err)
	}

	got, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState() error = %v", err)
	}
	if got != state {
		t.Fatalf("LoadAgentState() = %#v, want %#v", got, state)
	}
}

func TestSaveLoadRuntimeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.json")
	state := RuntimeState{
		CoreVersion:    "xray-test",
		AgentVersion:   "agent-test",
		LastConfigHash: "config-hash",
		LastUsersHash:  "users-hash",
		LastError:      "",
		LastApplyAt:    1700000000,
		UpdatedAt:      1700000100,
	}

	if err := SaveRuntimeState(path, state); err != nil {
		t.Fatalf("SaveRuntimeState() error = %v", err)
	}

	got, err := LoadRuntimeState(path)
	if err != nil {
		t.Fatalf("LoadRuntimeState() error = %v", err)
	}
	if got != state {
		t.Fatalf("LoadRuntimeState() = %#v, want %#v", got, state)
	}
}

func TestSaveLoadUsersCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.cache.json")
	cache := UsersCache{
		Version: 1,
		Users: []nodeapi.UserInfo{
			{
				ID:             1,
				UUID:           "11111111-1111-4111-8111-111111111111",
				Email:          "user@example.com",
				SpeedLimitMbps: 100,
				IPLimit:        2,
				Enabled:        true,
				UpdatedAt:      1700000000,
			},
		},
		UsersHash: "users-hash",
		UsersETag: `W/"users-v1"`,
		UpdatedAt: 1700000100,
	}

	if err := SaveUsersCache(path, cache); err != nil {
		t.Fatalf("SaveUsersCache() error = %v", err)
	}

	got, err := LoadUsersCache(path)
	if err != nil {
		t.Fatalf("LoadUsersCache() error = %v", err)
	}
	if got.Version != cache.Version || got.UsersHash != cache.UsersHash || got.UsersETag != cache.UsersETag || got.UpdatedAt != cache.UpdatedAt {
		t.Fatalf("LoadUsersCache() = %#v, want %#v", got, cache)
	}
	if len(got.Users) != len(cache.Users) || got.Users[0] != cache.Users[0] {
		t.Fatalf("LoadUsersCache().Users = %#v, want %#v", got.Users, cache.Users)
	}
}

func TestLoadInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadAgentState(path); err == nil {
		t.Fatal("LoadAgentState() error = nil, want invalid JSON error")
	}
}

func TestHashUsersStableWhenOrderChanges(t *testing.T) {
	users := []nodeapi.UserInfo{
		{ID: 2, UUID: "22222222-2222-4222-8222-222222222222", Email: "two@example.com", Enabled: true, UpdatedAt: 1700000000},
		{ID: 1, UUID: "11111111-1111-4111-8111-111111111111", Email: "one@example.com", Enabled: true, UpdatedAt: 1700000100},
	}
	reordered := []nodeapi.UserInfo{users[1], users[0]}

	hashA, err := HashUsers(users)
	if err != nil {
		t.Fatalf("HashUsers() error = %v", err)
	}
	hashB, err := HashUsers(reordered)
	if err != nil {
		t.Fatalf("HashUsers(reordered) error = %v", err)
	}
	if hashA == "" {
		t.Fatal("HashUsers() returned empty hash")
	}
	if hashA != hashB {
		t.Fatalf("HashUsers() = %q, HashUsers(reordered) = %q", hashA, hashB)
	}
}

func TestHashNodeConfigReturnsNonEmptyHash(t *testing.T) {
	hash, err := HashNodeConfig(vless.DefaultNodeConfig(1001, "node1.example.com"))
	if err != nil {
		t.Fatalf("HashNodeConfig() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashNodeConfig() returned empty hash")
	}
}
