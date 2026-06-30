# Operations

This repository is currently at Step 17. It can render a local Xray JSON
configuration for VLESS + REALITY + Vision, includes a guarded Xray runtime
process manager skeleton, centralizes the VLESS inbound builder in
`internal/protocol/vless`, and has a cancellable agent loop framework for
heartbeat and sync scheduling. The sync path now uses users ETag/cache metadata
and config/users hashes to skip runtime apply work when nothing changed. Step
14 adds a reporter framework for traffic, online IP, and detect-log payloads,
and Step 15 adds tolerant access log parsing for online IP extraction, but does
not start new reporting loops. Step 16 adds detect-rule validation and routing
render for supported rules. Step 17 adds installer and Docker Compose templates
for later Linux server rollout.

Local Windows verification:

```powershell
go test ./...
go vet ./...
go build -o .\bin\xnode.exe .\cmd\xnode
.\bin\xnode.exe --version
.\bin\xnode.exe --check
```

The Dockerfile, compose template, and install script template are deployment
previews for later Linux server work. They do not require Docker on Windows,
do not start Xray in local `--check`, and are not a completed production
installer yet.

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

## Step 15 access log parser skeleton

Step 15 adds `internal/logparser`, a tolerant parser for Xray access log lines.
It extracts stable generated user email tags such as `user-10001@panel.local`,
maps them back to `user_id`, extracts IPv4 or bracketed IPv6 source addresses,
and builds deduplicated, sorted online IP payloads for the existing reporter
framework.

Invalid or partial log lines are skipped and counted; they are not fatal.
Real long-running file tailing is not implemented yet. The local `--check`
command still performs one sync and exits; it does not start Xray, Docker,
reporter loops, or access log tailing.

## Step 16 detect-rule routing skeleton

Step 16 adds `internal/audit` validation helpers for detect rules and renders
supported rules into the Xray routing block. Supported rule types are
`protocol` and `domain_regex`.

Invalid rules, invalid regular expressions, and unknown rule types are skipped,
not fatal. The renderer still always includes the default bittorrent block rule.

This is only the routing skeleton. Real detect-log matching, traffic
inspection, long-running audit loops, production panel rollout behavior, Docker
behavior, and real Xray startup remain deferred. The local `--check` command
still performs one mock-safe sync and exits.

## Step 17 deployment template preview

Step 17 adds static rendering helpers in `internal/installer`, a Docker Compose
template under `deploy/`, and a Linux install script template under `scripts/`.
The templates target `/opt/xnode` on a future server and mount local `data/`
and `logs/` directories into the container.

After a future Linux deployment, the expected operator commands are:

```sh
cd /opt/xnode
docker compose ps
docker compose logs -f xnode
docker compose restart xnode
cat data/runtime.json
cat logs/xray.log
tail -f logs/access.log
```

Local Windows development still does not require Docker. Real Docker execution
and real panel rollout tests remain deferred to a later Linux server step.
