package sspanel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestEnrollSendsPostAndAuthorizationHeader(t *testing.T) {
	const token = "secret-node-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != enrollPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, enrollPath)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		writeAPIResponse(t, w, http.StatusOK, nodeapi.EnrollResponse{
			NodeToken:         "issued-token",
			PanelURL:          "https://panel.example.com",
			NodeID:            1001,
			Domain:            "node1.example.com",
			ReportIntervalSec: 60,
			ConfigIntervalSec: 300,
		})
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, token, server.Client())
	got, err := client.Enroll(context.Background(), nodeapi.EnrollRequest{
		NodeID:       1001,
		Domain:       "node1.example.com",
		AgentVersion: "test",
	})
	if err != nil {
		t.Fatalf("Enroll() error = %v", err)
	}
	if got.NodeToken != "issued-token" {
		t.Fatalf("NodeToken = %q, want issued-token", got.NodeToken)
	}
	if got.NodeID != 1001 {
		t.Fatalf("NodeID = %d, want 1001", got.NodeID)
	}
}

func TestGetConfigDecodesNodeConfig(t *testing.T) {
	want := nodeapi.NodeConfig{
		SchemaVersion: 1,
		NodeID:        1001,
		Domain:        "node1.example.com",
		Profile: nodeapi.NodeProfile{
			Name:     "node1",
			Protocol: "vless",
			Network:  "tcp",
			Security: "reality",
			Flow:     "xtls-rprx-vision",
			Listen:   "0.0.0.0",
			Port:     443,
		},
		Reality: nodeapi.RealityConfig{
			Target:      "www.example.com:443",
			ServerNames: []string{"www.example.com"},
			Fingerprint: "chrome",
			PublicKey:   "public-key",
			ShortIDs:    []string{"abc123"},
		},
		Report: nodeapi.ReportConfig{
			UserSyncIntervalSec:  300,
			TrafficIntervalSec:   60,
			OnlineIntervalSec:    60,
			HeartbeatIntervalSec: 30,
		},
		ConfigHash: "hash-1",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != configPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, configPath)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}
		writeAPIResponse(t, w, http.StatusOK, want)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	got, err := client.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if got.NodeID != want.NodeID {
		t.Fatalf("NodeID = %d, want %d", got.NodeID, want.NodeID)
	}
	if got.Domain != want.Domain {
		t.Fatalf("Domain = %q, want %q", got.Domain, want.Domain)
	}
	if got.ConfigHash != want.ConfigHash {
		t.Fatalf("ConfigHash = %q, want %q", got.ConfigHash, want.ConfigHash)
	}
}

func TestGetUsersSendsIfNoneMatchAndReturnsETag(t *testing.T) {
	users := []nodeapi.UserInfo{
		{
			ID:             1,
			UUID:           "11111111-1111-4111-8111-111111111111",
			Email:          "user@example.com",
			SpeedLimitMbps: 100,
			IPLimit:        2,
			Enabled:        true,
			UpdatedAt:      1700000000,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usersPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, usersPath)
		}
		if got := r.Header.Get("If-None-Match"); got != "users-v1" {
			t.Fatalf("If-None-Match = %q, want users-v1", got)
		}
		w.Header().Set("ETag", "users-v2")
		writeAPIResponse(t, w, http.StatusOK, users)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	gotUsers, gotETag, err := client.GetUsers(context.Background(), "users-v1")
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}
	if gotETag != "users-v2" {
		t.Fatalf("etag = %q, want users-v2", gotETag)
	}
	if len(gotUsers) != 1 || gotUsers[0].Email != users[0].Email {
		t.Fatalf("users = %#v, want %#v", gotUsers, users)
	}
}

func TestGetUsersHandlesNotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usersPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, usersPath)
		}
		w.Header().Set("ETag", "users-v1")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	users, etag, err := client.GetUsers(context.Background(), "users-v1")
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}
	if users != nil {
		t.Fatalf("users = %#v, want nil", users)
	}
	if etag != "users-v1" {
		t.Fatalf("etag = %q, want users-v1", etag)
	}
}

func TestGetDetectRulesHandlesNotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != detectRulesPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, detectRulesPath)
		}
		w.Header().Set("ETag", "rules-v1")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	rules, etag, err := client.GetDetectRules(context.Background(), "rules-v1")
	if err != nil {
		t.Fatalf("GetDetectRules() error = %v", err)
	}
	if rules != nil {
		t.Fatalf("rules = %#v, want nil", rules)
	}
	if etag != "rules-v1" {
		t.Fatalf("etag = %q, want rules-v1", etag)
	}
}

func TestReportRuntimeSendsPublicRealityFieldsOnly(t *testing.T) {
	var rawBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != runtimePath {
			t.Fatalf("path = %q, want %q", r.URL.Path, runtimePath)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode request body error = %v", err)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Marshal payload error = %v", err)
		}
		rawBody = string(encoded)

		if payload["public_key"] != "public-key" {
			t.Fatalf("public_key = %#v, want public-key", payload["public_key"])
		}
		shortIDs, ok := payload["short_ids"].([]any)
		if !ok || len(shortIDs) != 2 || shortIDs[0] != "abc123" || shortIDs[1] != "def456" {
			t.Fatalf("short_ids = %#v, want two short IDs", payload["short_ids"])
		}

		writeAPIResponse(t, w, http.StatusOK, json.RawMessage(`{}`))
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	err := client.ReportRuntime(context.Background(), nodeapi.RuntimeReport{
		NodeID:       1001,
		AgentVersion: "test",
		State:        "running",
		PublicKey:    "public-key",
		ShortIDs:     []string{"abc123", "def456"},
		ConfigHash:   "config-hash",
	})
	if err != nil {
		t.Fatalf("ReportRuntime() error = %v", err)
	}
	if strings.Contains(rawBody, "private_key") {
		t.Fatalf("runtime report body contains private_key: %s", rawBody)
	}
}

func TestReportHeartbeatSendsStateAndConfigHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != heartbeatPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, heartbeatPath)
		}
		var report nodeapi.HeartbeatReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Fatalf("Decode request body error = %v", err)
		}
		if report.State != "running" {
			t.Fatalf("State = %q, want running", report.State)
		}
		if report.ConfigHash != "config-hash" {
			t.Fatalf("ConfigHash = %q, want config-hash", report.ConfigHash)
		}

		writeAPIResponse(t, w, http.StatusOK, json.RawMessage(`{}`))
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	if err := client.ReportHeartbeat(context.Background(), nodeapi.HeartbeatReport{
		NodeID:       1001,
		AgentVersion: "test",
		State:        "running",
		ConfigHash:   "config-hash",
	}); err != nil {
		t.Fatalf("ReportHeartbeat() error = %v", err)
	}
}

func TestRetZeroResponseReturnsUsefulError(t *testing.T) {
	const token = "secret-node-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodeapi.APIResponse[json.RawMessage]{
			Ret:       0,
			Msg:       "bad token",
			Code:      "AUTH_FAILED",
			RequestID: "req-1",
		}); err != nil {
			t.Fatalf("Encode response error = %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, token, server.Client())
	_, err := client.GetConfig(context.Background())
	if err == nil {
		t.Fatal("GetConfig() error = nil, want error")
	}
	errText := err.Error()
	for _, want := range []string{"AUTH_FAILED", "bad token", "req-1"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want to contain %q", errText, want)
		}
	}
	if strings.Contains(errText, token) {
		t.Fatalf("error leaked token: %q", errText)
	}
}

func TestUnauthorizedResponseReturnsUsefulError(t *testing.T) {
	const token = "secret-node-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(nodeapi.APIResponse[json.RawMessage]{
			Ret:       0,
			Msg:       "unauthorized",
			Code:      "UNAUTHORIZED",
			RequestID: "req-401",
		}); err != nil {
			t.Fatalf("Encode response error = %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, token, server.Client())
	_, err := client.GetConfig(context.Background())
	if err == nil {
		t.Fatal("GetConfig() error = nil, want error")
	}
	errText := err.Error()
	for _, want := range []string{"401", "Unauthorized", "UNAUTHORIZED", "req-401"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want to contain %q", errText, want)
		}
	}
	if strings.Contains(errText, token) {
		t.Fatalf("error leaked token: %q", errText)
	}
}

func TestInvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ret":1,"data":`)); err != nil {
			t.Fatalf("Write response error = %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, "token", server.Client())
	_, err := client.GetConfig(context.Background())
	if err == nil {
		t.Fatal("GetConfig() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("error = %q, want invalid JSON", err.Error())
	}
}

func TestConstructorTrimsTrailingSlashFromPanelURL(t *testing.T) {
	client := NewClient("https://panel.example.com///", "token")
	if client.PanelURL != "https://panel.example.com" {
		t.Fatalf("PanelURL = %q, want https://panel.example.com", client.PanelURL)
	}
	if client.HTTPClient == nil {
		t.Fatal("HTTPClient = nil")
	}
	if client.HTTPClient.Timeout != defaultHTTPTimeout {
		t.Fatalf("HTTPClient.Timeout = %s, want %s", client.HTTPClient.Timeout, defaultHTTPTimeout)
	}
}

func TestSetTokenUpdatesAuthorizationHeader(t *testing.T) {
	requestCount := 0
	wantTokens := []string{"initial-node-token", "updated-node-token"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount >= len(wantTokens) {
			t.Fatalf("unexpected request %d", requestCount+1)
		}
		if r.URL.Path != configPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, configPath)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantTokens[requestCount] {
			t.Fatalf("request %d Authorization = %q, want bearer token", requestCount+1, got)
		}
		requestCount++
		writeAPIResponse(t, w, http.StatusOK, nodeapi.NodeConfig{
			SchemaVersion: 1,
			NodeID:        1001,
			Domain:        "node1.example.com",
		})
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL, wantTokens[0], server.Client())
	if _, err := client.GetConfig(context.Background()); err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	client.SetToken(wantTokens[1])
	if _, err := client.GetConfig(context.Background()); err != nil {
		t.Fatalf("GetConfig() after SetToken error = %v", err)
	}
	if requestCount != len(wantTokens) {
		t.Fatalf("request count = %d, want %d", requestCount, len(wantTokens))
	}
}

func writeAPIResponse[T any](t *testing.T, w http.ResponseWriter, statusCode int, data T) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(nodeapi.APIResponse[T]{
		Ret:  1,
		Data: data,
	}); err != nil {
		t.Fatalf("Encode response error = %v", err)
	}
}
