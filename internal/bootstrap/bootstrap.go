package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/makeausername/xnode-agent/internal/state"
)

type App struct {
	Version string
	State   *state.Manager
	Logger  *slog.Logger
}

func NewApp(version string) (*App, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	return &App{
		Version: version,
		State:   state.NewManager(state.Uninitialized),
		Logger:  logger,
	}, nil
}

func Run(ctx context.Context, version string) error {
	app, err := NewApp(version)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return app.Run(ctx)
}

func (a *App) Run(ctx context.Context) error {
	a.State.Set(state.Running)
	a.Logger.Info("xnode-agent started", "version", a.Version, "state", a.State.Get(), "component", "bootstrap")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.State.Set(state.Stopping)
			a.Logger.Info("xnode-agent stopped", "state", a.State.Get(), "component", "bootstrap")
			return nil
		case <-ticker.C:
			a.Logger.Info("heartbeat tick", "state", a.State.Get(), "component", "bootstrap")
		}
	}
}
