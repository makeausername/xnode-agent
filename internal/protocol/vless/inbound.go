package vless

import (
	"fmt"
	"strings"

	"github.com/makeausername/xnode-agent/internal/secrets"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

type Inbound struct {
	Tag            string          `json:"tag"`
	Listen         string          `json:"listen"`
	Port           int             `json:"port"`
	Protocol       string          `json:"protocol"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings StreamSettings  `json:"streamSettings"`
	Sniffing       Sniffing        `json:"sniffing"`
}

type InboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

type Client struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow"`
}

type StreamSettings struct {
	Network         string          `json:"network"`
	Security        string          `json:"security"`
	RealitySettings RealitySettings `json:"realitySettings"`
}

type RealitySettings struct {
	Show        bool     `json:"show"`
	Target      string   `json:"target"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIDs    []string `json:"shortIds"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

func BuildInbound(config nodeapi.NodeConfig, users []nodeapi.UserInfo, secret secrets.RealitySecret) (Inbound, error) {
	if err := ValidateConfig(config); err != nil {
		return Inbound{}, err
	}
	if err := ValidateSecret(secret); err != nil {
		return Inbound{}, err
	}

	clients, err := buildClients(config, users)
	if err != nil {
		return Inbound{}, err
	}

	return Inbound{
		Tag:      fmt.Sprintf("in-vless-reality-%d", config.NodeID),
		Listen:   strings.TrimSpace(config.Profile.Listen),
		Port:     config.Profile.Port,
		Protocol: strings.TrimSpace(config.Profile.Protocol),
		Settings: InboundSettings{
			Clients:    clients,
			Decryption: "none",
		},
		StreamSettings: StreamSettings{
			Network:  strings.TrimSpace(config.Profile.Network),
			Security: strings.TrimSpace(config.Profile.Security),
			RealitySettings: RealitySettings{
				Show:        false,
				Target:      strings.TrimSpace(config.Reality.Target),
				ServerNames: cleanNonEmptyStrings(config.Reality.ServerNames),
				PrivateKey:  strings.TrimSpace(secret.PrivateKey),
				ShortIDs:    cleanNonEmptyStrings(secret.ShortIDs),
			},
		},
		Sniffing: Sniffing{
			Enabled:      true,
			DestOverride: []string{"http", "tls", "quic"},
		},
	}, nil
}

func StableUserEmail(userID int64) string {
	return fmt.Sprintf("user-%d@panel.local", userID)
}

func buildClients(config nodeapi.NodeConfig, users []nodeapi.UserInfo) ([]Client, error) {
	clients := make([]Client, 0, len(users))
	flow := strings.TrimSpace(config.Profile.Flow)

	for _, user := range users {
		if !user.Enabled {
			continue
		}
		if strings.TrimSpace(user.UUID) == "" {
			return nil, fmt.Errorf("enabled user %d uuid is required", user.ID)
		}

		clients = append(clients, Client{
			ID:    strings.TrimSpace(user.UUID),
			Email: StableUserEmail(user.ID),
			Flow:  flow,
		})
	}

	return clients, nil
}

func cleanNonEmptyStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}
