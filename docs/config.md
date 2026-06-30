# Configuration

This repository is currently at Step 2 development stage. Local configuration,
state paths, and mock panel mode are being frozen before real panel API calls,
real Xray process management, or real Docker installer logic are implemented.

The deployment template passes configuration through environment variables:

- `PANEL_URL`
- `NODE_ID`
- `NODE_DOMAIN`
- `ENROLL_TOKEN`
- `DATA_DIR` (default: `/var/lib/xnode`)
- `LOG_DIR` (default: `/var/log/xnode`)
- `XRAY_BIN` (default: `/usr/local/bin/xray`)
- `XNODE_MOCK_PANEL=true` or `XNODE_MOCK_PANEL=1`
- `TZ=Asia/Shanghai`

Local state files:

- `/var/lib/xnode/agent.json`
- `/var/lib/xnode/token`
- `/var/lib/xnode/reality.json`
- `/var/lib/xnode/xray.json`
- `/var/lib/xnode/users.cache.json`
- `/var/lib/xnode/runtime.json`

Local log files:

- `/var/log/xnode/xray.log`
- `/var/log/xnode/access.log`

Step 2 documents and exposes the local layout only. It does not implement real
persistence, panel synchronization, Xray process management, or Docker installer
logic.
