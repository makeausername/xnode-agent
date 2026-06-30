# Operations

This repository is currently at Step 1 skeleton stage.

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
