package vless

import (
	"strings"
	"testing"

	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestDefaultNodeConfigReturnsVLESSRealityVisionProfile(t *testing.T) {
	config := DefaultNodeConfig(1001, "node1.example.com")

	if config.Profile.Name != DefaultProfileName {
		t.Fatalf("profile name = %q, want %q", config.Profile.Name, DefaultProfileName)
	}
	if config.Profile.Protocol != DefaultProtocol {
		t.Fatalf("profile protocol = %q, want %q", config.Profile.Protocol, DefaultProtocol)
	}
	if config.Profile.Network != DefaultNetwork {
		t.Fatalf("profile network = %q, want %q", config.Profile.Network, DefaultNetwork)
	}
	if config.Profile.Security != DefaultSecurity {
		t.Fatalf("profile security = %q, want %q", config.Profile.Security, DefaultSecurity)
	}
	if config.Profile.Flow != DefaultFlow {
		t.Fatalf("profile flow = %q, want %q", config.Profile.Flow, DefaultFlow)
	}
	if config.Profile.Listen != DefaultListen {
		t.Fatalf("profile listen = %q, want %q", config.Profile.Listen, DefaultListen)
	}
	if config.Profile.Port != DefaultPort {
		t.Fatalf("profile port = %d, want %d", config.Profile.Port, DefaultPort)
	}
}

func TestStableUserEmail(t *testing.T) {
	if got := StableUserEmail(10001); got != "user-10001@panel.local" {
		t.Fatalf("StableUserEmail(10001) = %q, want user-10001@panel.local", got)
	}
}

func TestBuildInboundMapsEnabledUsersAndRealitySettings(t *testing.T) {
	inbound, err := BuildInbound(testNodeConfig(), testUsers(), testSecret())
	if err != nil {
		t.Fatalf("BuildInbound() error = %v", err)
	}

	if inbound.Tag != "in-vless-reality-1001" {
		t.Fatalf("inbound tag = %q, want node_id tag", inbound.Tag)
	}
	if inbound.Protocol != DefaultProtocol {
		t.Fatalf("protocol = %q, want %q", inbound.Protocol, DefaultProtocol)
	}
	if inbound.Settings.Decryption != "none" {
		t.Fatalf("decryption = %q, want none", inbound.Settings.Decryption)
	}

	clients := inbound.Settings.Clients
	if len(clients) != 1 {
		t.Fatalf("len(clients) = %d, want only enabled users", len(clients))
	}
	if clients[0].ID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("client id = %q", clients[0].ID)
	}
	if clients[0].Level != 0 {
		t.Fatalf("client level = %d, want 0", clients[0].Level)
	}
	if clients[0].Email != "user-10001@panel.local" {
		t.Fatalf("client email = %q, want stable panel-local email", clients[0].Email)
	}
	if clients[0].Email == "do-not-use@example.com" {
		t.Fatalf("client email used user.Email = %q", clients[0].Email)
	}
	if clients[0].Flow != DefaultFlow {
		t.Fatalf("client flow = %q, want %q", clients[0].Flow, DefaultFlow)
	}

	if inbound.StreamSettings.Network != DefaultNetwork {
		t.Fatalf("stream network = %q, want %q", inbound.StreamSettings.Network, DefaultNetwork)
	}
	if inbound.StreamSettings.Security != DefaultSecurity {
		t.Fatalf("stream security = %q, want %q", inbound.StreamSettings.Security, DefaultSecurity)
	}

	reality := inbound.StreamSettings.RealitySettings
	if reality.Show {
		t.Fatal("reality show = true, want false")
	}
	if reality.Target != DefaultTarget {
		t.Fatalf("reality target = %q, want %q", reality.Target, DefaultTarget)
	}
	if len(reality.ServerNames) != 1 || reality.ServerNames[0] != DefaultServerName {
		t.Fatalf("reality serverNames = %#v, want default server name", reality.ServerNames)
	}
	if reality.PrivateKey != "local-private-key" {
		t.Fatalf("reality privateKey = %q, want local private key", reality.PrivateKey)
	}
	if len(reality.ShortIDs) != 1 || reality.ShortIDs[0] != "abcdef0123456789" {
		t.Fatalf("reality shortIds = %#v, want local short ids", reality.ShortIDs)
	}

	if !inbound.Sniffing.Enabled {
		t.Fatal("sniffing enabled = false, want true")
	}
	wantDestOverride := []string{"http", "tls", "quic"}
	if len(inbound.Sniffing.DestOverride) != len(wantDestOverride) {
		t.Fatalf("sniffing destOverride = %#v, want %#v", inbound.Sniffing.DestOverride, wantDestOverride)
	}
	for i := range wantDestOverride {
		if inbound.Sniffing.DestOverride[i] != wantDestOverride[i] {
			t.Fatalf("sniffing destOverride = %#v, want %#v", inbound.Sniffing.DestOverride, wantDestOverride)
		}
	}
}

func TestBuildInboundValidatesMissingPrivateKey(t *testing.T) {
	secret := testSecret()
	secret.PrivateKey = ""

	_, err := BuildInbound(testNodeConfig(), testUsers(), secret)
	if err == nil {
		t.Fatal("BuildInbound() error = nil, want missing private_key error")
	}
	if !strings.Contains(err.Error(), "private_key") {
		t.Fatalf("BuildInbound() error = %q, want private_key", err.Error())
	}
}

func TestBuildInboundValidatesEmptyUUIDForEnabledUser(t *testing.T) {
	users := testUsers()
	users[0].UUID = ""

	_, err := BuildInbound(testNodeConfig(), users, testSecret())
	if err == nil {
		t.Fatal("BuildInbound() error = nil, want empty uuid error")
	}
	if !strings.Contains(err.Error(), "uuid") {
		t.Fatalf("BuildInbound() error = %q, want uuid", err.Error())
	}
}

func testNodeConfig() nodeapi.NodeConfig {
	return DefaultNodeConfig(1001, "node1.example.com")
}

func testUsers() []nodeapi.UserInfo {
	return []nodeapi.UserInfo{
		{
			ID:      10001,
			UUID:    "11111111-1111-4111-8111-111111111111",
			Email:   "do-not-use@example.com",
			Enabled: true,
		},
		{
			ID:      10002,
			UUID:    "22222222-2222-4222-8222-222222222222",
			Email:   "disabled@example.com",
			Enabled: false,
		},
	}
}

func testSecret() secrets.RealitySecret {
	return secrets.RealitySecret{
		PrivateKey: "local-private-key",
		PublicKey:  "public-key",
		ShortIDs:   []string{"abcdef0123456789"},
	}
}
