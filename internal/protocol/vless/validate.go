package vless

import (
	"errors"
	"fmt"
	"strings"

	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

func ValidateConfig(config nodeapi.NodeConfig) error {
	if config.NodeID <= 0 {
		return errors.New("node_id is required and must be > 0")
	}
	if strings.TrimSpace(config.Domain) == "" {
		return errors.New("domain is required")
	}
	if strings.TrimSpace(config.Profile.Protocol) != DefaultProtocol {
		return fmt.Errorf("profile.protocol must be %q", DefaultProtocol)
	}
	if strings.TrimSpace(config.Profile.Network) != DefaultNetwork {
		return fmt.Errorf("profile.network must be %q", DefaultNetwork)
	}
	if strings.TrimSpace(config.Profile.Security) != DefaultSecurity {
		return fmt.Errorf("profile.security must be %q", DefaultSecurity)
	}
	if strings.TrimSpace(config.Profile.Flow) == "" {
		return errors.New("profile.flow is required")
	}
	if strings.TrimSpace(config.Profile.Listen) == "" {
		return errors.New("profile.listen is required")
	}
	if config.Profile.Port <= 0 {
		return errors.New("profile.port is required and must be > 0")
	}
	if strings.TrimSpace(config.Reality.Target) == "" {
		return errors.New("reality.target is required")
	}
	if len(cleanNonEmptyStrings(config.Reality.ServerNames)) == 0 {
		return errors.New("at least one reality server_name is required")
	}

	return nil
}

func ValidateSecret(secret secrets.RealitySecret) error {
	if strings.TrimSpace(secret.PrivateKey) == "" {
		return errors.New("reality private_key is required")
	}
	if len(cleanNonEmptyStrings(secret.ShortIDs)) == 0 {
		return errors.New("at least one reality short_id is required")
	}

	return nil
}
