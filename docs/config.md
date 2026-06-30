# Configuration

This repository is currently at Step 1 skeleton stage.

The deployment template passes configuration through environment variables:

- `PANEL_URL`
- `NODE_ID`
- `NODE_DOMAIN`
- `ENROLL_TOKEN`
- `TZ=Asia/Shanghai`

Intended local state files:

- `/var/lib/xnode/agent.json`
- `/var/lib/xnode/token`
- `/var/lib/xnode/reality.json`
- `/var/lib/xnode/xray.json`
- `/var/lib/xnode/users.cache.json`
- `/var/lib/xnode/runtime.json`

The files above document the intended layout only. Step 1 does not implement real persistence, panel synchronization, or Xray runtime management.
