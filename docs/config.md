# Configuration

This repository is currently at Step 5 development stage. Local configuration,
state paths, mock panel mode, local Secret Vault persistence, and Reality key
generation are being frozen before real panel API calls, real Xray process
management, or real Docker installer logic are implemented.

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

## Secret Vault

The local Secret Vault is file-backed under `DATA_DIR`.

- `token` contains `node_token` and must not be committed.
- `reality.json` is generated automatically when the agent first syncs.
- `reality.json` contains Reality `private_key`, `public_key`, `short_ids`, and
  `created_at`.
- `private_key` must never leave the node and must never be uploaded to the
  panel.
- `public_key` and `short_ids` may be reported to the panel.
- Deleting `reality.json` causes future regeneration and will invalidate
  existing subscription parameters.

Step 5 documents and implements local Reality key and shortId generation only.
It does not implement real panel synchronization, Xray process management, or
Docker installer logic.
