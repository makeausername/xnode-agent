# Operations

This repository is currently at Step 14. It can render a local Xray JSON
configuration for VLESS + REALITY + Vision, includes a guarded Xray runtime
process manager skeleton, centralizes the VLESS inbound builder in
`internal/protocol/vless`, and has a cancellable agent loop framework for
heartbeat and sync scheduling. The sync path now uses users ETag/cache metadata
and config/users hashes to skip runtime apply work when nothing changed. Step
14 adds a reporter framework for traffic, online IP, and detect-log payloads,
but does not start new reporting loops.

Local Windows verification:

```powershell
go test ./...
go vet ./...
go build -o .\bin\xnode.exe .\cmd\xnode
.\bin\xnode.exe --version
.\bin\xnode.exe --check
```

The Dockerfile and compose file under `deploy/` are templates for later deployment work. They do not require an Xray binary yet, and Docker is not required for local Windows development.

The placeholder smoke test runs the agent version command:

```sh
sh scripts/smoke.sh
```

After a local mock sync, inspect the rendered config with:

```sh
cat data/xray.json
```

## Step 8 Xray runtime skeleton

Step 8 adds the runtime process manager framework for starting and stopping an
external Xray process with `xray run -config <xray.json>`. The process manager
validates that the rendered config file exists and contains valid JSON before it
tries to start anything.

The current local `--check` flow may render config through `Runtime.ApplyPlan`
when local hashes changed or `xray.json` is missing. It does not call
`Runtime.Start`, does not start the Xray process, and does not run Docker. Real
Xray process startup will be tested later on a Linux server with an installed
Xray binary.

## Step 9 protocol builder boundary

The first protocol target is fixed to VLESS + REALITY + Vision + TCP/raw + 443.
The Xray runtime package renders the full Xray config wrapper and manages config
files/process state, but protocol-specific inbound construction belongs in
`internal/protocol/vless`.

Additional protocols should be added later as separate protocol builders instead
of being mixed into `internal/runtime/xray`.

## Step 12 agent loop skeleton

Step 12 adds context-aware loop helpers for config sync, user sync, and
heartbeat reporting. `App.Run` performs one startup `SyncOnce`, starts the
heartbeat scheduler and one conservative sync scheduler, then waits for context
cancellation. On Ctrl+C or SIGTERM, the app sets state to `stopping` and exits
cleanly.

The current loops are safe skeletons. Config and user sync still reuse
`SyncOnce`, heartbeat reporting reads local runtime health metadata, and mock
mode remains token-free. The local `--check` command still calls `SyncOnce` once
and exits; it does not start Xray, Docker, or any long-running scheduler.

Real Xray stats, traffic reporting, online IP parsing, and production panel
rollout behavior come later.

## Step 13 users sync optimization

`SyncOnce` loads `runtime.json` and `users.cache.json` before fetching panel
state. When a cached users ETag exists, it is sent with the users request. A
not-modified users response reuses `users.cache.json`, which is also the restart
fallback for the last usable users list.

Unchanged users and unchanged node config should not trigger
`Runtime.ApplyPlan`, so `xray.json` is not rewritten just because a sync loop
ran. The agent still applies the runtime plan if the config hash changed, the
users hash changed, or `xray.json` is missing.

## Step 14 reporter framework

Step 14 adds `internal/reporter`, deterministic `report_id` generation, and
panel-client support for:

- `POST /node/api/v1/traffic`
- `POST /node/api/v1/online`
- `POST /node/api/v1/detect-log`

The framework can build and send these reports in tests and mock-safe local
flows. Real Xray stats parsing, real access log parsing, detect-rule audit
matching, Docker behavior, production panel rollout, and long-running reporter
schedulers come later. The local `--check` command still performs one sync and
exits; it does not start Xray, Docker, or reporter loops.
