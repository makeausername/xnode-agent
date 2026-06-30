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
- Step 15 completed: access log parser for online IP extraction
- Step 16 completed: detect rules and audit routing skeleton
- Step 17 completed: installer and Docker Compose deployment templates

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
traffic collection yet. Step 15 adds a tolerant access log parser that maps
stable generated user email tags to `user_id` and builds deduplicated online IP
payloads from extracted source addresses. Step 16 adds a safe detect-rule
framework and renders supported `protocol` and `domain_regex` rules into Xray
routing block rules, skipping invalid rules instead of failing local config
render. Real panel calls are implemented at the client layer and tested with
`httptest`; the local check flow can use either mock mode or a reachable
SSPanel-compatible `/node/api/v1` stub depending on `XNODE_MOCK_PANEL`. It does
not start Xray from the local check flow. Step 17 adds installer and Docker
Compose templates for future Linux server deployment, but real Docker execution
remains deferred.

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

## Real panel stub check

Use this flow when a reachable SSPanel-compatible `/node/api/v1` stub is
available and you want to exercise real HTTP enrollment and sync without
starting Xray:

```powershell
$env:XNODE_MOCK_PANEL="false"
$env:PANEL_URL="https://panel.example.com"
$env:NODE_ID="1001"
$env:NODE_DOMAIN="node1.example.com"
$env:ENROLL_TOKEN="xne_xxx"
$env:DATA_DIR=".xnode-real\data"
$env:LOG_DIR=".xnode-real\logs"
.\bin\xnode.exe --check
```

`ENROLL_TOKEN` is used only once when `DATA_DIR\token` is missing. The panel
returns a `node_token`, and the agent saves it to `DATA_DIR\token` for later
Node API calls. `reality.json` and `xray.json` are generated locally under
`DATA_DIR`. The Reality `private_key` stays local and is never sent to the
panel; runtime reports send only the public Reality fields. `--check` performs
one sync and render pass, then exits without starting Xray.

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

Docker templates live under `deploy/`, and the Linux install script template
lives under `scripts/install.sh.tmpl`. They are static deployment skeletons for
later use.

Local Windows development remains lightweight: mock `--check` renders
`.xnode\data\xray.json` without a real panel request, a real Xray binary, or
Docker. Real panel stub `--check` makes Node API HTTP requests but still does
not start Xray or Docker.

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

Step 15 adds `internal/logparser` for online IP extraction from Xray access log
lines. Invalid log lines are skipped, not fatal, and real file tailing is still
deferred. Online IP report plumbing remains safe and test-only; `--check` does
not start Xray, tail logs, run Docker, or call a real panel in mock mode.

Step 16 adds `internal/audit` for detect-rule validation and matching helpers.
Supported rule types are `protocol` and `domain_regex`. The Xray renderer always
keeps the default bittorrent block rule and appends valid detect rules to
`routing.rules`; invalid detect rules are skipped and real detect-log matching
is still deferred.

Step 17 adds `internal/installer` rendering helpers plus Docker Compose and
install script templates. Local Windows development still does not require
Docker, and real Docker testing will happen later on a Linux server.
