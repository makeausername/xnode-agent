# Configuration

This repository is currently at Step 11 development stage. Local configuration,
state paths, mock panel mode, local Secret Vault persistence, Reality key
generation, Xray JSON rendering, local users cache, and runtime metadata are
implemented with SSPanel client enrollment and node token persistence.

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

- `/var/lib/xnode/agent.json` stores local enrollment and config identity
  state, including the panel URL, node ID, node domain, lifecycle state, and
  local timestamps.
- `/var/lib/xnode/token` stores the node token.
- `/var/lib/xnode/reality.json` stores the local Reality key material.
- `/var/lib/xnode/xray.json` stores the rendered Xray runtime config.
- `/var/lib/xnode/users.cache.json` stores the latest usable users and users
  hash for restart recovery.
- `/var/lib/xnode/runtime.json` stores runtime metadata, including the last
  config hash, last users hash, last apply timestamp, and last error.

None of these files should be committed. The repository ignores the local
`.xnode/` runtime directory used by Windows mock checks.

Local log files:

- `/var/log/xnode/xray.log`
- `/var/log/xnode/access.log`

## Secret Vault

The local Secret Vault is file-backed under `DATA_DIR`.

- `token` contains `node_token` and must not be committed.
- `ENROLL_TOKEN` is only needed for first enrollment when `token` is missing.
- After enrollment, the panel-issued `node_token` is saved to `token` and is
  used for config, user, rule, runtime, traffic, online, and heartbeat API
  requests.
- Deleting `token` forces re-enrollment and requires a fresh valid
  `ENROLL_TOKEN`.
- `reality.json` is generated automatically when the agent first syncs.
- `reality.json` contains Reality `private_key`, `public_key`, `short_ids`, and
  `created_at`.
- `private_key` must never leave the node and must never be uploaded to the
  panel.
- `public_key` and `short_ids` may be reported to the panel.
- Deleting `reality.json` causes future regeneration and will invalidate
  existing subscription parameters.

Mock panel mode does not require `ENROLL_TOKEN` and does not create or use a
node token.
