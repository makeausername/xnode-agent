# API

This repository is currently at Step 16.

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
| POST | `/node/api/v1/detect-log` | Report detect-rule log matches. |
| POST | `/node/api/v1/heartbeat` | Report lightweight node heartbeat state. |

`POST` requests use `Content-Type: application/json`.

## Report endpoints

Traffic, online IP, and detect-log report payloads include a required
`report_id` for idempotency. The Step 14 agent builds deterministic IDs with
this format:

```text
<node_id>-<period_start>-<kind>
```

For example, traffic for node `1001` and period start `1760000000` uses
`1001-1760000000-traffic`.

`POST /node/api/v1/traffic` sends:

```json
{
  "report_id": "1001-1760000000-traffic",
  "node_id": 1001,
  "period_start": 1760000000,
  "period_end": 1760000060,
  "data": [
    { "user_id": 1, "u": 100, "d": 200 }
  ]
}
```

`POST /node/api/v1/online` sends:

```json
{
  "report_id": "1001-1760000000-online",
  "node_id": 1001,
  "period_start": 1760000000,
  "period_end": 1760000060,
  "data": [
    { "user_id": 1, "ip": "203.0.113.10" },
    { "user_id": 2, "ip": "2001:db8::1" }
  ]
}
```

`POST /node/api/v1/detect-log` sends:

```json
{
  "report_id": "1001-1760000000-detect-log",
  "node_id": 1001,
  "period_start": 1760000000,
  "period_end": 1760000060,
  "data": [
    {
      "user_id": 1,
      "rule_id": 99,
      "ip": "203.0.113.10",
      "target": "example.com:443",
      "created_at": 1760000030
    }
  ]
}
```

The current reporter framework can build and send these payloads through the
panel client. Real Xray stats parsing, access log parsing, audit matching, and
production scheduling are intentionally deferred.

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

## Detect rules

Step 16 supports safe validation and Xray routing render for these detect-rule
types:

```json
[
  { "id": 1, "type": "protocol", "pattern": "bittorrent" },
  { "id": 2, "type": "domain_regex", "pattern": "(?i)example" }
]
```

Valid `protocol` rules render as Xray routing protocol block rules. Valid
`domain_regex` rules render as Xray routing domain rules with the `regexp:`
prefix. Invalid or unknown rules are skipped by the local renderer and are not
fatal. Real detect-log matching and traffic inspection remain deferred.

Automated repository verification uses mock panel mode. The optional real panel
stub check is documented in `README.md` and `docs/operations.md`; it still does
not start Xray or run Docker.
