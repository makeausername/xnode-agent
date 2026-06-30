# xnode-agent

Agent for `github.com/makeausername/xnode-agent`.

## Current development stage

- Step 1 completed: project skeleton
- Step 3 completed: bootstrap wiring for local config and mock panel
- Step 4 completed: local Secret Vault file persistence
- Step 5 completed: Reality key and shortId generation
- Step 6 completed: Xray JSON renderer for VLESS + REALITY + Vision
- Step 7 completed: local agent state, users cache, and runtime metadata files
- Step 8 completed: Xray Runtime process manager skeleton
- Step 9 completed: VLESS + REALITY + Vision protocol builder centralized in `internal/protocol/vless`
- Step 10 completed: SSPanel Node API v1 HTTP client skeleton
- Step 11 completed: agent enrollment flow and node_token persistence
- Step 12 completed: agent loop framework, heartbeat scheduler, and context-aware graceful shutdown
- Step 13 completed: ETag/hash-based user sync optimization and no-op runtime apply
- Step 14 completed: reporter framework for traffic, online IP, and detect-log reports

The current stage provides the project structure, initial command entrypoint,
DTO placeholders, state/bootstrap stubs, documentation, CI, deployment
templates, local configuration defaults, state path helpers, mock panel mode,
local Secret Vault file persistence, Reality key pair and shortId generation,
an Xray JSON config renderer, local agent state files, a users cache, runtime
metadata, a process manager skeleton for an external Xray process, and a
centralized VLESS + REALITY + Vision inbound builder, a SSPanel Node API v1 HTTP
client layer, a one-shot local sync check, real-mode enrollment, local
`node_token` persistence, and a reporter framework for traffic, online IP, and
detect-log payloads. Step 12 adds cancellable loop helpers for config sync, user
sync, and heartbeat reporting; `Run` performs an initial sync, starts a
heartbeat scheduler and one conservative sync scheduler, and exits cleanly on
context cancellation. Step 13 adds ETag/hash-based users sync, persists the
users ETag in `users.cache.json`, and skips `Runtime.ApplyPlan` when the node
config, users hash, and existing `xray.json` are unchanged. Step 14 adds
deterministic `report_id` generation and safe report builders, but no real
traffic collection yet. Real panel calls are implemented at the client layer and
tested with `httptest`, but the local check flow still uses the mock panel. It
does not start Xray from the local check flow or implement real Docker installer
logic.

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

One-shot sync without the check label is also available:

```powershell
.\bin\xnode.exe --once
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

Docker templates live under `deploy/` for later use. Do not treat them as a completed runtime deployment in Step 10.

Local Windows development remains lightweight: `--check` renders
`.xnode\data\xray.json` only, uses the mock panel, and still does not require a
real panel request, a real Xray binary, or Docker.

Step 11 wires real-panel enrollment into bootstrap. In real panel mode the agent
loads `.xnode\data\token` or enrolls once with `ENROLL_TOKEN`, saves the returned
`node_token`, and uses that token for later Node API calls. Mock mode remains
token-free and does not require `ENROLL_TOKEN`.

Step 12 is a safe loop framework only. Heartbeats report current runtime health
metadata without requiring Xray to be running. Config and user loop helpers still
reuse `SyncOnce` until their production responsibilities are split.

Step 13 optimizes that shared `SyncOnce` path. The agent sends the cached users
ETag when available, falls back to `users.cache.json` on a not-modified users
response, compares users/config hashes against `runtime.json`, and avoids
rewriting `xray.json` when nothing changed. Real Xray stats, traffic reporting,
online IP parsing, and production panel rollout behavior are deferred to later
steps.

Step 14 adds the reporter framework for traffic, online IP, and detect-log
reports. It can build deterministic idempotency IDs such as
`1001-1760000000-traffic` and send mock-safe payloads through the panel client.
It does not implement real Xray stats parsing, real access log parsing, or real
traffic collection yet.
