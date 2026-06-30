package xray

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/makeausername/xnode-agent/internal/runtime"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Runtime struct {
	BinPath       string
	ConfigPath    string
	process       *exec.Cmd
	processCancel context.CancelFunc
	mu            sync.Mutex
	health        runtime.Health
	lastStartAt   int64
}

func NewRuntime(binPath string, configPath string) *Runtime {
	return &Runtime{
		BinPath:    binPath,
		ConfigPath: configPath,
		health: runtime.Health{
			Running: false,
		},
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.process != nil && r.health.Running {
		return nil
	}

	if strings.TrimSpace(r.BinPath) == "" {
		err := errors.New("xray binary path is required")
		r.health.LastError = err.Error()
		return err
	}
	if err := validateConfigFile(r.ConfigPath); err != nil {
		r.health.LastError = err.Error()
		return err
	}

	processCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(processCtx, r.BinPath, "run", "-config", r.ConfigPath)

	now := time.Now().Unix()
	if err := cmd.Start(); err != nil {
		cancel()
		r.health.Running = false
		r.health.PID = 0
		r.health.LastError = fmt.Sprintf("start xray process %q: %v", r.BinPath, err)
		return fmt.Errorf("start xray process %q: %w", r.BinPath, err)
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	r.process = cmd
	r.processCancel = cancel
	r.lastStartAt = now
	r.health.Running = true
	r.health.PID = pid
	r.health.LastStartAt = now
	r.health.LastError = ""

	go r.waitProcess(cmd, cancel)

	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	cmd := r.process
	cancel := r.processCancel
	if cmd == nil || !r.health.Running {
		r.mu.Unlock()
		return nil
	}

	if cancel != nil {
		cancel()
	}

	process := cmd.Process
	if process == nil {
		r.process = nil
		r.processCancel = nil
		r.health.Running = false
		r.health.PID = 0
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		r.mu.Lock()
		r.health.LastError = fmt.Sprintf("stop xray process: %v", err)
		r.mu.Unlock()
		return fmt.Errorf("stop xray process: %w", err)
	}

	r.mu.Lock()
	if r.process == cmd {
		r.process = nil
		r.processCancel = nil
	}
	r.health.Running = false
	r.health.PID = 0
	r.mu.Unlock()

	return nil
}

func (r *Runtime) Health(ctx context.Context) runtime.Health {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.health
}

func (r *Runtime) ApplyPlan(ctx context.Context, plan runtime.RuntimePlan) error {
	data, err := RenderConfig(plan)
	if err != nil {
		return err
	}
	if err := WriteConfigAtomic(r.ConfigPath, data); err != nil {
		return err
	}

	r.mu.Lock()
	r.health.ConfigHash = plan.Hash
	r.mu.Unlock()

	return nil
}

func (r *Runtime) QueryStats(ctx context.Context, reset bool) ([]nodeapi.UserTraffic, error) {
	// TODO: Query Xray stats API once the process/runtime API integration is added.
	return []nodeapi.UserTraffic{}, nil
}

func validateConfigFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("xray config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("xray config path %q does not exist", path)
		}
		return fmt.Errorf("read xray config path %q: %w", path, err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("xray config path %q is not valid JSON", path)
	}

	return nil
}

func (r *Runtime) waitProcess(cmd *exec.Cmd, cancel context.CancelFunc) {
	err := cmd.Wait()
	cancel()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.process != cmd {
		return
	}

	r.process = nil
	r.processCancel = nil
	r.health.Running = false
	r.health.PID = 0
	if err != nil {
		r.health.LastError = fmt.Sprintf("xray process exited: %v", err)
		return
	}
	r.health.LastError = ""
}
