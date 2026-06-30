package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type panelTokenSetter interface {
	SetToken(token string)
}

func (a *App) EnsureNodeToken(ctx context.Context) error {
	if a.Config.MockPanel {
		return nil
	}

	token, err := a.Secrets.LoadToken()
	if err == nil && strings.TrimSpace(token) != "" {
		return a.setPanelToken(strings.TrimSpace(token))
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load node token: %w", err)
	}

	enrollToken := strings.TrimSpace(a.Config.EnrollToken)
	if enrollToken == "" {
		return errors.New("node token is missing and ENROLL_TOKEN is required for enrollment")
	}
	if err := a.setPanelToken(enrollToken); err != nil {
		return err
	}

	resp, err := a.Panel.Enroll(ctx, a.enrollRequest())
	if err != nil {
		return fmt.Errorf("enroll node: %w", err)
	}

	nodeToken := strings.TrimSpace(resp.NodeToken)
	if nodeToken == "" {
		return errors.New("enroll node: panel returned empty node_token")
	}

	if err := a.Secrets.SaveToken(nodeToken); err != nil {
		return fmt.Errorf("save node token: %w", err)
	}

	return a.setPanelToken(nodeToken)
}

func (a *App) setPanelToken(token string) error {
	setter, ok := a.Panel.(panelTokenSetter)
	if !ok {
		return fmt.Errorf("panel client %T cannot update node token", a.Panel)
	}
	setter.SetToken(token)
	return nil
}

func (a *App) enrollRequest() nodeapi.EnrollRequest {
	hostname, _ := os.Hostname()

	return nodeapi.EnrollRequest{
		NodeID:             a.Config.NodeID,
		Domain:             a.Config.NodeDomain,
		AgentVersion:       a.Version,
		InstallFingerprint: hostname,
		Host: nodeapi.HostInfo{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
	}
}
