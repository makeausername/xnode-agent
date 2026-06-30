package installer

import (
	"strings"
	"testing"
)

func TestRenderComposeIncludesDeploymentContract(t *testing.T) {
	compose, err := RenderCompose(validTemplateData())
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}

	assertContainsAll(t, compose,
		"image: ghcr.io/makeausername/xnode-agent:latest",
		"container_name: xnode",
		"network_mode: host",
		"restart: unless-stopped",
		"- ./data:/var/lib/xnode",
		"- ./logs:/var/log/xnode",
		`PANEL_URL: "${PANEL_URL}"`,
		`NODE_ID: "${NODE_ID}"`,
		`NODE_DOMAIN: "${NODE_DOMAIN}"`,
		`ENROLL_TOKEN: "${ENROLL_TOKEN}"`,
		`DATA_DIR: "/var/lib/xnode"`,
		`LOG_DIR: "/var/log/xnode"`,
		`XRAY_BIN: "/usr/local/bin/xray"`,
		`TZ: "Asia/Shanghai"`,
	)
}

func TestRenderInstallScriptIncludesPrompts(t *testing.T) {
	script, err := RenderInstallScript(validTemplateData())
	if err != nil {
		t.Fatalf("RenderInstallScript() error = %v", err)
	}

	assertContainsAll(t, script,
		"set -euo pipefail",
		"请输入节点 ID:",
		"read -r NODE_ID",
		"请输入节点域名:",
		"read -r NODE_DOMAIN",
		`mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/logs"`,
		"docker compose pull",
		"docker compose up -d",
		"docker compose logs -f xnode",
	)
}

func TestValidateTemplateDataRejectsEmptyPanelURL(t *testing.T) {
	data := validTemplateData()
	data.PanelURL = ""

	if err := ValidateTemplateData(data); err == nil {
		t.Fatal("ValidateTemplateData() error = nil, want PanelURL error")
	}
}

func TestTemplateRenderingAppliesDefaults(t *testing.T) {
	data := TemplateData{
		PanelURL: "https://panel.example.com",
		Image:    DefaultImage,
	}

	if err := ValidateTemplateData(data); err != nil {
		t.Fatalf("ValidateTemplateData() error = %v", err)
	}

	compose, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}
	assertContainsAll(t, compose, `TZ: "Asia/Shanghai"`)

	script, err := RenderInstallScript(data)
	if err != nil {
		t.Fatalf("RenderInstallScript() error = %v", err)
	}
	assertContainsAll(t, script, "INSTALL_DIR='/opt/xnode'", "cd /opt/xnode")
}

func TestRenderedInstallScriptDoesNotExposePrivateKey(t *testing.T) {
	script, err := RenderInstallScript(validTemplateData())
	if err != nil {
		t.Fatalf("RenderInstallScript() error = %v", err)
	}

	for _, forbidden := range []string{"private" + "_key", "private" + "Key"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("rendered install script contains %q", forbidden)
		}
	}
}

func TestRenderedTemplatesDoNotContainDotXnode(t *testing.T) {
	compose, err := RenderCompose(validTemplateData())
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}
	script, err := RenderInstallScript(validTemplateData())
	if err != nil {
		t.Fatalf("RenderInstallScript() error = %v", err)
	}

	for name, rendered := range map[string]string{
		"compose": compose,
		"script":  script,
	} {
		if strings.Contains(rendered, ".xnode") {
			t.Fatalf("%s template contains .xnode", name)
		}
	}
}

func validTemplateData() TemplateData {
	return TemplateData{
		PanelURL:    "https://panel.example.com",
		Image:       DefaultImage,
		EnrollToken: "test-enroll-token",
	}
}

func assertContainsAll(t *testing.T, value string, substrings ...string) {
	t.Helper()

	for _, substring := range substrings {
		if !strings.Contains(value, substring) {
			t.Fatalf("rendered output does not contain %q\n%s", substring, value)
		}
	}
}
