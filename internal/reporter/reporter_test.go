package reporter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/makeausername/xnode-agent/internal/logparser"
	"github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func TestBuildReportIDDeterministic(t *testing.T) {
	got := BuildReportID(1001, 1760000000, "traffic")
	if got != "1001-1760000000-traffic" {
		t.Fatalf("BuildReportID() = %q, want %q", got, "1001-1760000000-traffic")
	}
	if gotAgain := BuildReportID(1001, 1760000000, "traffic"); gotAgain != got {
		t.Fatalf("BuildReportID() changed from %q to %q", got, gotAgain)
	}
}

func TestReportTrafficSendsTrafficReportWithReportID(t *testing.T) {
	panel := &fakePanel{}
	runtime := &fakeRuntime{
		stats: []nodeapi.UserTraffic{
			{UserID: 7, Upload: 100, Download: 200},
		},
	}
	manager := NewManager(1001, panel, runtime)

	if err := manager.ReportTraffic(context.Background(), 1760000000, 1760000060); err != nil {
		t.Fatalf("ReportTraffic() error = %v", err)
	}
	if runtime.queryStatsCalls != 1 {
		t.Fatalf("QueryStats calls = %d, want 1", runtime.queryStatsCalls)
	}
	if !runtime.lastReset {
		t.Fatal("QueryStats reset = false, want true")
	}
	if panel.trafficCalls != 1 {
		t.Fatalf("ReportTraffic calls = %d, want 1", panel.trafficCalls)
	}

	want := nodeapi.TrafficReport{
		ReportID:    "1001-1760000000-traffic",
		NodeID:      1001,
		PeriodStart: 1760000000,
		PeriodEnd:   1760000060,
		Data: []nodeapi.UserTraffic{
			{UserID: 7, Upload: 100, Download: 200},
		},
	}
	if !reflect.DeepEqual(panel.trafficReport, want) {
		t.Fatalf("TrafficReport = %#v, want %#v", panel.trafficReport, want)
	}
}

func TestReportTrafficWorksWithEmptyStats(t *testing.T) {
	panel := &fakePanel{}
	manager := NewManager(1001, panel, &fakeRuntime{})

	if err := manager.ReportTraffic(context.Background(), 1760000000, 1760000060); err != nil {
		t.Fatalf("ReportTraffic() error = %v", err)
	}
	if panel.trafficCalls != 1 {
		t.Fatalf("ReportTraffic calls = %d, want 1", panel.trafficCalls)
	}
	if panel.trafficReport.ReportID != "1001-1760000000-traffic" {
		t.Fatalf("ReportID = %q, want deterministic traffic ID", panel.trafficReport.ReportID)
	}
	if panel.trafficReport.Data == nil {
		t.Fatal("TrafficReport.Data = nil, want empty slice")
	}
	if len(panel.trafficReport.Data) != 0 {
		t.Fatalf("len(TrafficReport.Data) = %d, want 0", len(panel.trafficReport.Data))
	}
}

func TestReportOnlineSendsIPv4AndIPv6Examples(t *testing.T) {
	panel := &fakePanel{}
	manager := NewManager(1001, panel, &fakeRuntime{})
	online := []nodeapi.OnlineIP{
		{UserID: 1, IP: "203.0.113.10"},
		{UserID: 2, IP: "2001:db8::1"},
	}

	if err := manager.ReportOnline(context.Background(), 1760000000, 1760000060, online); err != nil {
		t.Fatalf("ReportOnline() error = %v", err)
	}
	if panel.onlineCalls != 1 {
		t.Fatalf("ReportOnline calls = %d, want 1", panel.onlineCalls)
	}

	want := nodeapi.OnlineReport{
		ReportID:    "1001-1760000000-online",
		NodeID:      1001,
		PeriodStart: 1760000000,
		PeriodEnd:   1760000060,
		Data:        online,
	}
	if !reflect.DeepEqual(panel.onlineReport, want) {
		t.Fatalf("OnlineReport = %#v, want %#v", panel.onlineReport, want)
	}
}

func TestReportOnlineFromEntriesBuildsDeduplicatedSortedReport(t *testing.T) {
	panel := &fakePanel{}
	manager := NewManager(1001, panel, &fakeRuntime{})
	entries := []logparser.AccessEntry{
		{UserID: 2, SourceIP: "203.0.113.10"},
		{UserID: 1, SourceIP: "203.0.113.20"},
		{UserID: 1, SourceIP: "198.51.100.1"},
		{UserID: 1, SourceIP: "198.51.100.1"},
	}

	if err := manager.ReportOnlineFromEntries(context.Background(), 1760000000, 1760000060, entries); err != nil {
		t.Fatalf("ReportOnlineFromEntries() error = %v", err)
	}
	if panel.onlineCalls != 1 {
		t.Fatalf("ReportOnline calls = %d, want 1", panel.onlineCalls)
	}

	want := nodeapi.OnlineReport{
		ReportID:    "1001-1760000000-online",
		NodeID:      1001,
		PeriodStart: 1760000000,
		PeriodEnd:   1760000060,
		Data: []nodeapi.OnlineIP{
			{UserID: 1, IP: "198.51.100.1"},
			{UserID: 1, IP: "203.0.113.20"},
			{UserID: 2, IP: "203.0.113.10"},
		},
	}
	if !reflect.DeepEqual(panel.onlineReport, want) {
		t.Fatalf("OnlineReport = %#v, want %#v", panel.onlineReport, want)
	}
}

func TestReportDetectLogSendsDetectLogReport(t *testing.T) {
	panel := &fakePanel{}
	manager := NewManager(1001, panel, &fakeRuntime{})
	items := []nodeapi.DetectLogItem{
		{
			UserID:    1,
			RuleID:    99,
			IP:        "203.0.113.10",
			Target:    "example.com:443",
			CreatedAt: 1760000030,
		},
	}

	if err := manager.ReportDetectLog(context.Background(), 1760000000, 1760000060, items); err != nil {
		t.Fatalf("ReportDetectLog() error = %v", err)
	}
	if panel.detectLogCalls != 1 {
		t.Fatalf("ReportDetectLog calls = %d, want 1", panel.detectLogCalls)
	}

	want := nodeapi.DetectLogReport{
		ReportID:    "1001-1760000000-detect-log",
		NodeID:      1001,
		PeriodStart: 1760000000,
		PeriodEnd:   1760000060,
		Data:        items,
	}
	if !reflect.DeepEqual(panel.detectLogReport, want) {
		t.Fatalf("DetectLogReport = %#v, want %#v", panel.detectLogReport, want)
	}
}

func TestReportTrafficReturnsRuntimeQueryStatsErrorClearly(t *testing.T) {
	panel := &fakePanel{}
	manager := NewManager(1001, panel, &fakeRuntime{
		queryStatsErr: errors.New("stats offline"),
	})

	err := manager.ReportTraffic(context.Background(), 1760000000, 1760000060)
	if err == nil {
		t.Fatal("ReportTraffic() error = nil, want error")
	}
	for _, want := range []string{"query runtime traffic stats", "stats offline"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ReportTraffic() error = %q, want to contain %q", err.Error(), want)
		}
	}
	if panel.trafficCalls != 0 {
		t.Fatalf("ReportTraffic calls = %d, want 0 after runtime error", panel.trafficCalls)
	}
}

func TestReportTrafficReturnsPanelReportErrorClearly(t *testing.T) {
	panel := &fakePanel{trafficErr: errors.New("panel down")}
	manager := NewManager(1001, panel, &fakeRuntime{})

	err := manager.ReportTraffic(context.Background(), 1760000000, 1760000060)
	if err == nil {
		t.Fatal("ReportTraffic() error = nil, want error")
	}
	for _, want := range []string{"panel report traffic", "panel down"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ReportTraffic() error = %q, want to contain %q", err.Error(), want)
		}
	}
}

type fakeRuntime struct {
	stats           []nodeapi.UserTraffic
	queryStatsErr   error
	queryStatsCalls int
	lastReset       bool
}

func (r *fakeRuntime) Start(ctx context.Context) error {
	return nil
}

func (r *fakeRuntime) Stop(ctx context.Context) error {
	return nil
}

func (r *fakeRuntime) Health(ctx context.Context) runtime.Health {
	return runtime.Health{}
}

func (r *fakeRuntime) ApplyPlan(ctx context.Context, plan runtime.RuntimePlan) error {
	return nil
}

func (r *fakeRuntime) QueryStats(ctx context.Context, reset bool) ([]nodeapi.UserTraffic, error) {
	r.queryStatsCalls++
	r.lastReset = reset
	if r.queryStatsErr != nil {
		return nil, r.queryStatsErr
	}
	return append([]nodeapi.UserTraffic(nil), r.stats...), nil
}

type fakePanel struct {
	trafficReport   nodeapi.TrafficReport
	onlineReport    nodeapi.OnlineReport
	detectLogReport nodeapi.DetectLogReport

	trafficErr   error
	onlineErr    error
	detectLogErr error

	trafficCalls   int
	onlineCalls    int
	detectLogCalls int
}

func (p *fakePanel) Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error) {
	return nodeapi.EnrollResponse{}, nil
}

func (p *fakePanel) GetConfig(ctx context.Context) (nodeapi.NodeConfig, error) {
	return nodeapi.NodeConfig{}, nil
}

func (p *fakePanel) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	return nil, "", nil
}

func (p *fakePanel) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	return nil, "", nil
}

func (p *fakePanel) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	return nil
}

func (p *fakePanel) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	p.trafficCalls++
	p.trafficReport = report
	return p.trafficErr
}

func (p *fakePanel) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	p.onlineCalls++
	p.onlineReport = report
	return p.onlineErr
}

func (p *fakePanel) ReportDetectLog(ctx context.Context, report nodeapi.DetectLogReport) error {
	p.detectLogCalls++
	p.detectLogReport = report
	return p.detectLogErr
}

func (p *fakePanel) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	return nil
}
