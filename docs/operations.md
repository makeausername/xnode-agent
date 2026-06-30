# Operations

This repository is currently at Step 6. It can render a local Xray JSON
configuration for VLESS + REALITY + Vision, but it does not start or restart
Xray yet.

Local Windows verification:

```powershell
go test ./...
go vet ./...
go build -o .\bin\xnode.exe .\cmd\xnode
.\bin\xnode.exe --version
```

The Dockerfile and compose file under `deploy/` are templates for later deployment work. They do not require an Xray binary yet, and Docker is not required for Step 1 local development.

The placeholder smoke test runs the agent version command:

```sh
sh scripts/smoke.sh
```

After a local mock sync, inspect the rendered config with:

```sh
cat data/xray.json
```

Step 6 only renders the config file. It does not call the Xray binary, start
the Xray process, or run Docker.
