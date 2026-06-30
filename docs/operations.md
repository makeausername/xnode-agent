# Operations

This repository is currently at Step 8. It can render a local Xray JSON
configuration for VLESS + REALITY + Vision and includes a guarded Xray runtime
process manager skeleton.

Local Windows verification:

```powershell
go test ./...
go vet ./...
go build -o .\bin\xnode.exe .\cmd\xnode
.\bin\xnode.exe --version
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

The current local `--check` flow still renders config only through
`Runtime.ApplyPlan`. It does not call `Runtime.Start`, does not start the Xray
process, and does not run Docker. Real Xray process startup will be tested later
on a Linux server with an installed Xray binary.
