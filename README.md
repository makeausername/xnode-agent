# xnode-agent

Agent for `github.com/makeausername/xnode-agent`.

## Current development stage

- Step 1 completed: project skeleton
- Step 3 completed: bootstrap wiring for local config and mock panel
- Step 4 in progress: local Secret Vault file persistence

The current stage provides the project structure, initial command entrypoint,
DTO placeholders, state/bootstrap stubs, documentation, CI, deployment
templates, local configuration defaults, state path helpers, mock panel mode,
local Secret Vault file persistence, and a one-shot local sync check. It does
not implement real panel API logic, real Xray runtime logic, Reality key
generation, or real Docker installer logic.

Target protocol:

```text
VLESS + REALITY + Vision + TCP/raw + 443
```

Local Windows verification:

```powershell
go test ./...
go vet ./...
go build -o .\bin\xnode.exe .\cmd\xnode
.\bin\xnode.exe --version
```

Local mock check:

```powershell
$env:XNODE_MOCK_PANEL="true"
$env:NODE_ID="1001"
$env:NODE_DOMAIN="node1.example.com"
.\bin\xnode.exe --check
```

Docker templates live under `deploy/` for later use. Do not treat them as a completed runtime deployment in Step 3.
