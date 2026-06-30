package installer

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
)

const (
	DefaultImage      = "ghcr.io/makeausername/xnode-agent:latest"
	DefaultInstallDir = "/opt/xnode"
	DefaultTimezone   = "Asia/Shanghai"
)

type TemplateData struct {
	PanelURL    string
	Image       string
	EnrollToken string
	InstallDir  string
	Timezone    string
}

func RenderCompose(data TemplateData) (string, error) {
	normalized, err := normalizeTemplateData(data)
	if err != nil {
		return "", err
	}

	return renderCompose(normalized)
}

func RenderInstallScript(data TemplateData) (string, error) {
	normalized, err := normalizeTemplateData(data)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("install.sh.tmpl").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"compose": func() (string, error) {
				return renderCompose(normalized)
			},
			"shellQuote": shellQuote,
		}).
		Parse(installScriptTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, normalized); err != nil {
		return "", err
	}
	return out.String(), nil
}

func ValidateTemplateData(data TemplateData) error {
	_, err := normalizeTemplateData(data)
	return err
}

func renderCompose(data TemplateData) (string, error) {
	tmpl, err := template.New("docker-compose.yml.tmpl").
		Option("missingkey=error").
		Parse(composeTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func normalizeTemplateData(data TemplateData) (TemplateData, error) {
	data.PanelURL = strings.TrimSpace(data.PanelURL)
	data.Image = strings.TrimSpace(data.Image)
	data.EnrollToken = strings.TrimSpace(data.EnrollToken)
	data.InstallDir = strings.TrimSpace(data.InstallDir)
	data.Timezone = strings.TrimSpace(data.Timezone)

	if data.InstallDir == "" {
		data.InstallDir = DefaultInstallDir
	}
	if data.Timezone == "" {
		data.Timezone = DefaultTimezone
	}

	if data.PanelURL == "" {
		return TemplateData{}, errors.New("PanelURL is required")
	}
	if data.Image == "" {
		return TemplateData{}, errors.New("Image is required")
	}

	for name, value := range map[string]string{
		"PanelURL":    data.PanelURL,
		"Image":       data.Image,
		"EnrollToken": data.EnrollToken,
		"InstallDir":  data.InstallDir,
		"Timezone":    data.Timezone,
	} {
		if strings.ContainsAny(value, "\x00\r\n") {
			return TemplateData{}, fmt.Errorf("%s must not contain control characters or line breaks", name)
		}
	}

	return data, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

const composeTemplate = `services:
  xnode:
    image: {{ .Image }}
    container_name: xnode
    network_mode: host
    restart: unless-stopped
    volumes:
      - ./data:/var/lib/xnode
      - ./logs:/var/log/xnode
    environment:
      PANEL_URL: "${PANEL_URL}"
      NODE_ID: "${NODE_ID}"
      NODE_DOMAIN: "${NODE_DOMAIN}"
      ENROLL_TOKEN: "${ENROLL_TOKEN}"
      DATA_DIR: "/var/lib/xnode"
      LOG_DIR: "/var/log/xnode"
      XRAY_BIN: "/usr/local/bin/xray"
      TZ: "{{ .Timezone }}"
`

const installScriptTemplate = `#!/usr/bin/env bash
set -euo pipefail

PANEL_URL={{ shellQuote .PanelURL }}
ENROLL_TOKEN={{ shellQuote .EnrollToken }}
INSTALL_DIR={{ shellQuote .InstallDir }}

if [ "$(id -u)" -ne 0 ]; then
  echo "error: install.sh must be run as root" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is required but was not found. Install Docker first, then rerun this script." >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "error: docker compose is required but was not found. Install Docker Compose v2 first, then rerun this script." >&2
  exit 1
fi

if command -v ss >/dev/null 2>&1; then
  if [ -n "$(ss -ltnH 'sport = :443' 2>/dev/null || true)" ]; then
    echo "error: port 443 is already occupied. Stop the existing service before installing xnode." >&2
    exit 1
  fi
fi

printf '请输入节点 ID: '
read -r NODE_ID
printf '请输入节点域名: '
read -r NODE_DOMAIN

if [ -z "${NODE_ID}" ]; then
  echo "error: node ID is required" >&2
  exit 1
fi

if [ -z "${NODE_DOMAIN}" ]; then
  echo "error: node domain is required" >&2
  exit 1
fi

umask 077
mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/logs"

cat > "${INSTALL_DIR}/.env" <<ENV
PANEL_URL=${PANEL_URL}
NODE_ID=${NODE_ID}
NODE_DOMAIN=${NODE_DOMAIN}
ENROLL_TOKEN=${ENROLL_TOKEN}
DATA_DIR=/var/lib/xnode
LOG_DIR=/var/log/xnode
XRAY_BIN=/usr/local/bin/xray
TZ={{ .Timezone }}
ENV

cat > "${INSTALL_DIR}/docker-compose.yml" <<'COMPOSE'
{{ compose }}COMPOSE

cd "${INSTALL_DIR}"
docker compose pull
docker compose up -d

printf '%s\n' "xnode installed. Useful commands:" "cd {{ .InstallDir }}" "docker compose logs -f xnode"
`
