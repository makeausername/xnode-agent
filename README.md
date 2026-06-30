# xnode-agent

Step 1 skeleton for `github.com/makeausername/xnode-agent`.

The current stage only provides the project structure, initial command entrypoint, DTO placeholders, state/bootstrap stubs, documentation, CI, and deployment templates. It does not implement real panel API logic or real Xray runtime logic.

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

Docker templates live under `deploy/` for later use. Do not treat them as a completed runtime deployment in Step 1.
