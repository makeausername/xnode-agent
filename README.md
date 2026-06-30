# xnode-agent

Agent for `github.com/makeausername/xnode-agent`.

## Current development stage

- Step 1 completed: project skeleton
- Step 3 completed: bootstrap wiring for local config and mock panel
- Step 4 completed: local Secret Vault file persistence
- Step 5 completed: Reality key and shortId generation
- Step 6 completed: Xray JSON renderer for VLESS + REALITY + Vision
- Step 7 completed: local agent state, users cache, and runtime metadata files

The current stage provides the project structure, initial command entrypoint,
DTO placeholders, state/bootstrap stubs, documentation, CI, deployment
templates, local configuration defaults, state path helpers, mock panel mode,
local Secret Vault file persistence, Reality key pair and shortId generation,
an Xray JSON config renderer, local agent state files, a users cache, runtime
metadata, and a one-shot local sync check. It does not implement real panel API
logic, start Xray, or implement real Docker installer logic.

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
$env:DATA_DIR=".xnode\data"
$env:LOG_DIR=".xnode\logs"
.\bin\xnode.exe --check
```

The local mock check now creates:

```text
.xnode\data\agent.json
.xnode\data\runtime.json
.xnode\data\users.cache.json
.xnode\data\reality.json
.xnode\data\xray.json
```

The `.xnode` runtime directory is ignored by git and must not be committed.

Docker templates live under `deploy/` for later use. Do not treat them as a completed runtime deployment in Step 7.
