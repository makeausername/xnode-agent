package mock

import (
	"context"
	"testing"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestGetConfigReturnsSchemaVersionOne(t *testing.T) {
	client := NewClient()

	cfg, err := client.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if cfg.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", cfg.SchemaVersion)
	}
}

func TestGetUsersReturnsEnabledUser(t *testing.T) {
	client := NewClient()

	users, _, err := client.GetUsers(context.Background(), "")
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}
	if len(users) == 0 {
		t.Fatal("GetUsers() returned no users")
	}

	for _, user := range users {
		if user.Enabled {
			return
		}
	}

	t.Fatal("GetUsers() returned no enabled users")
}

func TestReportMethodsReturnNil(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	if err := client.ReportRuntime(ctx, nodeapi.RuntimeReport{}); err != nil {
		t.Fatalf("ReportRuntime() error = %v", err)
	}
	if err := client.ReportTraffic(ctx, nodeapi.TrafficReport{}); err != nil {
		t.Fatalf("ReportTraffic() error = %v", err)
	}
	if err := client.ReportOnline(ctx, nodeapi.OnlineReport{}); err != nil {
		t.Fatalf("ReportOnline() error = %v", err)
	}
	if err := client.ReportHeartbeat(ctx, nodeapi.HeartbeatReport{}); err != nil {
		t.Fatalf("ReportHeartbeat() error = %v", err)
	}
}

func TestReportRuntimeAcceptsRealityPublicFields(t *testing.T) {
	client := NewClient()
	report := nodeapi.RuntimeReport{
		NodeID:       1001,
		AgentVersion: "test-version",
		State:        "running",
		PublicKey:    "public-key",
		ShortIDs:     []string{"0123456789abcdef"},
		Capabilities: []string{"vless", "reality", "vision"},
	}

	if err := client.ReportRuntime(context.Background(), report); err != nil {
		t.Fatalf("ReportRuntime() error = %v", err)
	}

	got, ok := client.LastRuntimeReport()
	if !ok {
		t.Fatal("LastRuntimeReport() ok = false, want true")
	}
	if got.PublicKey != report.PublicKey {
		t.Fatalf("PublicKey = %q, want %q", got.PublicKey, report.PublicKey)
	}
	if len(got.ShortIDs) != 1 || got.ShortIDs[0] != report.ShortIDs[0] {
		t.Fatalf("ShortIDs = %#v, want %#v", got.ShortIDs, report.ShortIDs)
	}
}
