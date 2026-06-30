# API

This repository is currently at Step 11.

The `pkg/nodeapi` package contains DTOs for the SSPanel Node API v1 contract. The
`internal/panel/sspanel` package implements the HTTP client layer for these
endpoints and is tested with `net/http/httptest`.

## Node API v1 endpoints

The client sends `Accept: application/json` on all requests. Enrollment uses
`Authorization: Bearer <ENROLL_TOKEN>` and returns a panel-issued
`node_token`. The agent stores that token locally and uses
`Authorization: Bearer <node_token>` for every other endpoint.

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/node/api/v1/enroll` | Enroll an installed node and receive a panel-issued node token. |
| GET | `/node/api/v1/config` | Fetch node configuration. |
| GET | `/node/api/v1/users` | Fetch enabled users. Supports `If-None-Match` and returns response `ETag`. |
| GET | `/node/api/v1/detect-rules` | Fetch detect rules. Supports `If-None-Match` and returns response `ETag`. |
| POST | `/node/api/v1/runtime` | Report runtime state and public REALITY fields. |
| POST | `/node/api/v1/traffic` | Report user traffic counters. |
| POST | `/node/api/v1/online` | Report online user IP state. |
| POST | `/node/api/v1/heartbeat` | Report lightweight node heartbeat state. |

`POST` requests use `Content-Type: application/json`.

## Enrollment flow

On bootstrap in real panel mode, the agent first tries to load
`DATA_DIR/token`. If a non-empty token exists, it configures the SSPanel client
with that `node_token` before fetching config. If the token file is missing,
the agent requires `ENROLL_TOKEN`, calls `POST /node/api/v1/enroll`, stores the
returned `node_token` in `DATA_DIR/token`, and then switches the client to that
token before continuing sync.

Mock panel mode skips enrollment and does not require either token.

## Response format

Successful responses use a common envelope:

```json
{
  "ret": 1,
  "data": {},
  "msg": "",
  "code": "",
  "request_id": ""
}
```

When `ret` is not `1`, the client returns an error containing `code`, `msg`, and
`request_id`. Authentication tokens and private keys must not be included in
errors or logs.

For user and detect-rule sync, `304 Not Modified` is treated as a cache hit: the
client returns nil data, the response `ETag` if present, and no error.

Repository verification uses mock panel mode. Do not call a real panel, start
Xray, or run Docker for the current local check flow.
