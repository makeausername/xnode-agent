package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/makeausername/xnode-agent/internal/protocol/vless"
	runtimex "github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type renderedConfig struct {
	Inbounds []struct {
		Tag      string `json:"tag"`
		Listen   string `json:"listen"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Settings struct {
			Clients []struct {
				ID    string `json:"id"`
				Level int    `json:"level"`
				Email string `json:"email"`
				Flow  string `json:"flow"`
			} `json:"clients"`
			Decryption string `json:"decryption"`
		} `json:"settings"`
		StreamSettings struct {
			Network         string `json:"network"`
			Security        string `json:"security"`
			RealitySettings struct {
				Target      string   `json:"target"`
				ServerNames []string `json:"serverNames"`
				PrivateKey  string   `json:"privateKey"`
				ShortIDs    []string `json:"shortIds"`
			} `json:"realitySettings"`
		} `json:"streamSettings"`
	} `json:"inbounds"`
	Routing Routing `json:"routing"`
}

func TestRenderConfigReturnsValidJSONAndMapsUsers(t *testing.T) {
	plan := testPlan()

	data, err := RenderConfig(plan)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("RenderConfig() returned invalid JSON: %s", data)
	}
	assertRenderedVLESSSettings(t, data)

	rendered := decodeRenderedConfig(t, data)
	if len(rendered.Inbounds) != 1 {
		t.Fatalf("len(inbounds) = %d, want 1", len(rendered.Inbounds))
	}

	inbound := rendered.Inbounds[0]
	if inbound.Tag != "in-vless-reality-1001" {
		t.Fatalf("inbound tag = %q, want node_id tag", inbound.Tag)
	}
	if inbound.Protocol != "vless" {
		t.Fatalf("inbound protocol = %q, want vless", inbound.Protocol)
	}

	clients := inbound.Settings.Clients
	if len(clients) != 1 {
		t.Fatalf("len(clients) = %d, want 1 enabled user", len(clients))
	}
	if clients[0].ID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("client id = %q", clients[0].ID)
	}
	if clients[0].Level != 0 {
		t.Fatalf("client level = %d, want 0", clients[0].Level)
	}
	if clients[0].Email != "user-1@panel.local" {
		t.Fatalf("client email = %q, want stable panel-local email", clients[0].Email)
	}
	if clients[0].Flow != "xtls-rprx-vision" {
		t.Fatalf("client flow = %q, want xtls-rprx-vision", clients[0].Flow)
	}
	if inbound.Settings.Decryption != "none" {
		t.Fatalf("decryption = %q, want none", inbound.Settings.Decryption)
	}

	reality := inbound.StreamSettings.RealitySettings
	if reality.PrivateKey != "local-private-key" {
		t.Fatalf("reality privateKey = %q, want local private key", reality.PrivateKey)
	}
	if len(reality.ShortIDs) != 1 || reality.ShortIDs[0] != "abcdef0123456789" {
		t.Fatalf("reality shortIds = %#v, want local short ids", reality.ShortIDs)
	}
}

func TestRenderConfigMissingPrivateKeyReturnsError(t *testing.T) {
	plan := testPlan()
	plan.Secrets.PrivateKey = ""

	_, err := RenderConfig(plan)
	if err == nil {
		t.Fatal("RenderConfig() error = nil, want missing private key error")
	}
	if !strings.Contains(err.Error(), "private_key") {
		t.Fatalf("RenderConfig() error = %q, want private_key", err.Error())
	}
}

func TestRenderConfigUsesProtocolBuilderInboundShape(t *testing.T) {
	plan := testPlan()

	data, err := RenderConfig(plan)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	var rendered xrayConfig
	if err := json.Unmarshal(data, &rendered); err != nil {
		t.Fatalf("Unmarshal(rendered config) error = %v", err)
	}
	if len(rendered.Inbounds) != 1 {
		t.Fatalf("len(inbounds) = %d, want 1", len(rendered.Inbounds))
	}

	want, err := vless.BuildInbound(plan.NodeConfig, plan.Users, plan.Secrets)
	if err != nil {
		t.Fatalf("vless.BuildInbound() error = %v", err)
	}
	if !reflect.DeepEqual(rendered.Inbounds[0], want) {
		t.Fatalf("rendered inbound = %#v, want protocol builder inbound %#v", rendered.Inbounds[0], want)
	}
}

func TestBuildRoutingRulesIncludesDefaultBittorrentRule(t *testing.T) {
	rules := BuildRoutingRules(nil)

	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if !reflect.DeepEqual(rules[0], RoutingRule{
		Type:        "field",
		Protocol:    []string{"bittorrent"},
		OutboundTag: "block",
	}) {
		t.Fatalf("default routing rule = %#v", rules[0])
	}
}

func TestRenderConfigRendersProtocolDetectRule(t *testing.T) {
	plan := testPlan()
	plan.Rules = []nodeapi.DetectRule{
		{ID: 10, Type: "protocol", Pattern: "http"},
	}

	data, err := RenderConfig(plan)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	rendered := decodeRenderedConfig(t, data)
	if len(rendered.Routing.Rules) != 2 {
		t.Fatalf("len(routing.rules) = %d, want 2", len(rendered.Routing.Rules))
	}
	got := rendered.Routing.Rules[1]
	if !reflect.DeepEqual(got.Protocol, []string{"http"}) {
		t.Fatalf("protocol rule protocol = %#v, want http", got.Protocol)
	}
	if got.OutboundTag != "block" {
		t.Fatalf("protocol rule outboundTag = %q, want block", got.OutboundTag)
	}
}

func TestRenderConfigRendersDomainRegexDetectRule(t *testing.T) {
	plan := testPlan()
	plan.Rules = []nodeapi.DetectRule{
		{ID: 11, Type: "domain_regex", Pattern: `(?i)example`},
	}

	data, err := RenderConfig(plan)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	rendered := decodeRenderedConfig(t, data)
	if len(rendered.Routing.Rules) != 2 {
		t.Fatalf("len(routing.rules) = %d, want 2", len(rendered.Routing.Rules))
	}
	got := rendered.Routing.Rules[1]
	if !reflect.DeepEqual(got.Domain, []string{"regexp:(?i)example"}) {
		t.Fatalf("domain rule domain = %#v, want regexp-prefixed pattern", got.Domain)
	}
	if len(got.Protocol) != 0 {
		t.Fatalf("domain rule protocol = %#v, want empty", got.Protocol)
	}
}

func TestRenderConfigSkipsInvalidDetectRules(t *testing.T) {
	plan := testPlan()
	plan.Rules = []nodeapi.DetectRule{
		{ID: 12, Type: "domain_regex", Pattern: `[`},
		{ID: 13, Type: "unknown", Pattern: "noop"},
		{ID: 14, Type: "protocol", Pattern: "bittorrent"},
	}

	data, err := RenderConfig(plan)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("RenderConfig() returned invalid JSON: %s", data)
	}

	rendered := decodeRenderedConfig(t, data)
	if len(rendered.Routing.Rules) != 2 {
		t.Fatalf("len(routing.rules) = %d, want default plus one valid rule", len(rendered.Routing.Rules))
	}
	if !reflect.DeepEqual(rendered.Routing.Rules[1].Protocol, []string{"bittorrent"}) {
		t.Fatalf("valid protocol rule = %#v, want bittorrent", rendered.Routing.Rules[1])
	}
}

func TestWriteConfigAtomicWritesValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "xray.json")
	data := []byte(`{"ok":true}`)

	if err := WriteConfigAtomic(path, data); err != nil {
		t.Fatalf("WriteConfigAtomic() error = %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !json.Valid(written) {
		t.Fatalf("written data is invalid JSON: %s", written)
	}
}

func TestWriteConfigAtomicReplacesExistingJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.json")

	if err := WriteConfigAtomic(path, []byte(`{"version":1}`)); err != nil {
		t.Fatalf("WriteConfigAtomic(first) error = %v", err)
	}
	if err := WriteConfigAtomic(path, []byte(`{"version":2}`)); err != nil {
		t.Fatalf("WriteConfigAtomic(second) error = %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(written) != `{"version":2}` {
		t.Fatalf("written data = %q, want second JSON", string(written))
	}
}

func TestWriteConfigAtomicRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.json")

	err := WriteConfigAtomic(path, []byte(`{"ok":`))
	if err == nil {
		t.Fatal("WriteConfigAtomic() error = nil, want invalid JSON error")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("Stat(path) error = %v, want not exist", statErr)
	}
}

func TestApplyPlanWritesConfigAndUpdatesHealthHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.json")
	runtime := NewRuntime("xray", path)
	plan := testPlan()
	plan.Hash = "hash-123"

	if err := runtime.ApplyPlan(context.Background(), plan); err != nil {
		t.Fatalf("ApplyPlan() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(xray.json) error = %v", err)
	}
	_ = decodeRenderedConfig(t, data)

	health := runtime.Health(context.Background())
	if health.ConfigHash != "hash-123" {
		t.Fatalf("health.ConfigHash = %q, want hash-123", health.ConfigHash)
	}
	if health.Running {
		t.Fatal("health.Running = true, want false after ApplyPlan")
	}
	if health.PID != 0 {
		t.Fatalf("health.PID = %d, want 0 after ApplyPlan", health.PID)
	}
	if health.LastStartAt != 0 {
		t.Fatalf("health.LastStartAt = %d, want 0 after ApplyPlan", health.LastStartAt)
	}
	if runtime.process != nil {
		t.Fatal("runtime.process is not nil, ApplyPlan must not start Xray")
	}
}

func decodeRenderedConfig(t *testing.T, data []byte) renderedConfig {
	t.Helper()

	var config renderedConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Unmarshal(rendered config) error = %v", err)
	}
	return config
}

func assertRenderedVLESSSettings(t *testing.T, data []byte) {
	t.Helper()

	if strings.Contains(string(data), `"users"`) {
		t.Fatalf("rendered xray config contains unsupported settings.users: %s", data)
	}

	var raw struct {
		Inbounds []struct {
			Protocol string         `json:"protocol"`
			Settings map[string]any `json:"settings"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal(raw rendered config) error = %v", err)
	}

	for _, inbound := range raw.Inbounds {
		if inbound.Protocol != "vless" {
			continue
		}
		if _, ok := inbound.Settings["users"]; ok {
			t.Fatal("vless settings includes unsupported users key")
		}
		rawClients, ok := inbound.Settings["clients"]
		if !ok {
			t.Fatal("vless settings missing clients key")
		}
		clients, ok := rawClients.([]any)
		if !ok {
			t.Fatalf("vless settings.clients type = %T, want array", rawClients)
		}
		if len(clients) != 1 {
			t.Fatalf("len(settings.clients) = %d, want 1", len(clients))
		}
		client, ok := clients[0].(map[string]any)
		if !ok {
			t.Fatalf("settings.clients[0] type = %T, want object", clients[0])
		}
		assertRawSetting(t, client, "id", "11111111-1111-4111-8111-111111111111")
		assertRawSetting(t, client, "email", "user-1@panel.local")
		assertRawSetting(t, client, "flow", "xtls-rprx-vision")
		assertRawSetting(t, client, "level", float64(0))
		assertRawSetting(t, inbound.Settings, "decryption", "none")
		return
	}

	t.Fatal("rendered config missing vless inbound")
}

func assertRawSetting(t *testing.T, settings map[string]any, key string, want any) {
	t.Helper()

	if got := settings[key]; got != want {
		t.Fatalf("%s = %s, want %s", key, formatRawValue(got), formatRawValue(want))
	}
}

func formatRawValue(value any) string {
	return fmt.Sprintf("%#v", value)
}

func testPlan() runtimex.RuntimePlan {
	nodeConfig := vless.DefaultNodeConfig(1001, "node1.example.com")

	return runtimex.RuntimePlan{
		NodeConfig: nodeConfig,
		Users: []nodeapi.UserInfo{
			{
				ID:      1,
				UUID:    "11111111-1111-4111-8111-111111111111",
				Email:   "do-not-use@example.com",
				Enabled: true,
			},
			{
				ID:      2,
				UUID:    "22222222-2222-4222-8222-222222222222",
				Email:   "disabled@example.com",
				Enabled: false,
			},
		},
		Secrets: secrets.RealitySecret{
			PrivateKey: "local-private-key",
			PublicKey:  "public-key",
			ShortIDs:   []string{"abcdef0123456789"},
		},
		Hash: "mock-config",
	}
}
