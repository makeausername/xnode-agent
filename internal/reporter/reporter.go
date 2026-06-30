package reporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/makeausername/xnode-agent/internal/logparser"
	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	reportKindTraffic   = "traffic"
	reportKindOnline    = "online"
	reportKindDetectLog = "detect-log"
)

type Manager struct {
	NodeID  int64
	Panel   panel.Client
	Runtime runtime.Runtime
	Logger  *slog.Logger
}

func NewManager(nodeID int64, panelClient panel.Client, runtimeClient runtime.Runtime) *Manager {
	return &Manager{
		NodeID:  nodeID,
		Panel:   panelClient,
		Runtime: runtimeClient,
	}
}

func (m *Manager) ReportTraffic(ctx context.Context, periodStart, periodEnd int64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("report traffic: manager is nil")
	}
	if m.Runtime == nil {
		return errors.New("report traffic: runtime is nil")
	}
	if m.Panel == nil {
		return errors.New("report traffic: panel is nil")
	}

	data, err := m.Runtime.QueryStats(ctx, true)
	if err != nil {
		return fmt.Errorf("query runtime traffic stats: %w", err)
	}

	report := nodeapi.TrafficReport{
		ReportID:    BuildReportID(m.NodeID, periodStart, reportKindTraffic),
		NodeID:      m.NodeID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Data:        cloneTraffic(data),
	}
	if err := m.Panel.ReportTraffic(ctx, report); err != nil {
		return fmt.Errorf("panel report traffic: %w", err)
	}

	return nil
}

func (m *Manager) ReportOnline(ctx context.Context, periodStart, periodEnd int64, online []nodeapi.OnlineIP) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("report online: manager is nil")
	}
	if m.Panel == nil {
		return errors.New("report online: panel is nil")
	}

	report := nodeapi.OnlineReport{
		ReportID:    BuildReportID(m.NodeID, periodStart, reportKindOnline),
		NodeID:      m.NodeID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Data:        cloneOnline(online),
	}
	if err := m.Panel.ReportOnline(ctx, report); err != nil {
		return fmt.Errorf("panel report online: %w", err)
	}

	return nil
}

func (m *Manager) ReportOnlineFromEntries(ctx context.Context, periodStart, periodEnd int64, entries []logparser.AccessEntry) error {
	return m.ReportOnline(ctx, periodStart, periodEnd, logparser.BuildOnlineIPs(entries))
}

func (m *Manager) ReportDetectLog(ctx context.Context, periodStart, periodEnd int64, items []nodeapi.DetectLogItem) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("report detect-log: manager is nil")
	}
	if m.Panel == nil {
		return errors.New("report detect-log: panel is nil")
	}

	report := nodeapi.DetectLogReport{
		ReportID:    BuildReportID(m.NodeID, periodStart, reportKindDetectLog),
		NodeID:      m.NodeID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Data:        cloneDetectLogItems(items),
	}
	if err := m.Panel.ReportDetectLog(ctx, report); err != nil {
		return fmt.Errorf("panel report detect-log: %w", err)
	}

	return nil
}

func cloneTraffic(data []nodeapi.UserTraffic) []nodeapi.UserTraffic {
	if data == nil {
		return []nodeapi.UserTraffic{}
	}
	return append([]nodeapi.UserTraffic(nil), data...)
}

func cloneOnline(data []nodeapi.OnlineIP) []nodeapi.OnlineIP {
	if data == nil {
		return []nodeapi.OnlineIP{}
	}
	return append([]nodeapi.OnlineIP(nil), data...)
}

func cloneDetectLogItems(data []nodeapi.DetectLogItem) []nodeapi.DetectLogItem {
	if data == nil {
		return []nodeapi.DetectLogItem{}
	}
	return append([]nodeapi.DetectLogItem(nil), data...)
}
