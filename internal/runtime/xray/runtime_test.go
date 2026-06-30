package xray

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("XRAY_FAKE_PROCESS") == "1" &&
		len(os.Args) >= 4 &&
		os.Args[1] == "run" &&
		os.Args[2] == "-config" &&
		os.Args[3] != "" {
		select {}
	}

	os.Exit(m.Run())
}

func TestStartMissingConfigReturnsClearError(t *testing.T) {
	runtime := NewRuntime("xray", filepath.Join(t.TempDir(), "missing-xray.json"))

	err := runtime.Start(context.Background())
	if err == nil {
		t.Fatal("Start() error = nil, want missing config error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("Start() error = %q, want does not exist", err.Error())
	}

	health := runtime.Health(context.Background())
	if health.Running {
		t.Fatal("health.Running = true, want false")
	}
	if health.LastError == "" {
		t.Fatal("health.LastError is empty, want start error")
	}
}

func TestStartInvalidConfigReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.json")
	if err := os.WriteFile(path, []byte(`{"ok":`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runtime := NewRuntime("xray", path)

	err := runtime.Start(context.Background())
	if err == nil {
		t.Fatal("Start() error = nil, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("Start() error = %q, want not valid JSON", err.Error())
	}

	health := runtime.Health(context.Background())
	if health.Running {
		t.Fatal("health.Running = true, want false")
	}
	if health.LastError == "" {
		t.Fatal("health.LastError is empty, want start error")
	}
}

func TestStopBeforeStartReturnsNil(t *testing.T) {
	runtime := NewRuntime("xray", filepath.Join(t.TempDir(), "xray.json"))

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil before Start", err)
	}

	health := runtime.Health(context.Background())
	if health.Running {
		t.Fatal("health.Running = true, want false")
	}
}

func TestQueryStatsReturnsEmptySlice(t *testing.T) {
	runtime := NewRuntime("xray", filepath.Join(t.TempDir(), "xray.json"))

	traffic, err := runtime.QueryStats(context.Background(), true)
	if err != nil {
		t.Fatalf("QueryStats() error = %v", err)
	}
	if traffic == nil {
		t.Fatal("QueryStats() traffic = nil, want empty slice")
	}
	if len(traffic) != 0 {
		t.Fatalf("len(QueryStats()) = %d, want 0", len(traffic))
	}
}

func TestStartAndStopFakeProcess(t *testing.T) {
	t.Setenv("XRAY_FAKE_PROCESS", "1")

	path := filepath.Join(t.TempDir(), "xray.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runtime := NewRuntime(os.Args[0], path)

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer runtime.Stop(context.Background())

	health := runtime.Health(context.Background())
	if !health.Running {
		t.Fatal("health.Running = false, want true")
	}
	if health.PID <= 0 {
		t.Fatalf("health.PID = %d, want process PID", health.PID)
	}
	if health.LastStartAt == 0 {
		t.Fatal("health.LastStartAt = 0, want start timestamp")
	}
	if runtime.process == nil {
		t.Fatal("runtime.process = nil, want process")
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start(already running) error = %v", err)
	}
	healthAfterSecondStart := runtime.Health(context.Background())
	if healthAfterSecondStart.PID != health.PID {
		t.Fatalf("second Start PID = %d, want same PID %d", healthAfterSecondStart.PID, health.PID)
	}

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	health = runtime.Health(context.Background())
	if health.Running {
		t.Fatal("health.Running = true, want false")
	}
	if health.PID != 0 {
		t.Fatalf("health.PID = %d, want 0 after Stop", health.PID)
	}
}
