# Design: tg-alerts CLI

Date: 2026-05-29
Status: approved for implementation planning
Owner: user00

## Summary

`tg-alerts` is a local command-line tool for querying Telegram alert channels from an operator's machine. It is designed primarily for LLMs and automation agents, so it emits stable machine-readable JSON by default and optionally emits NDJSON.

The first version is read-only. It does not send, edit, delete, acknowledge, or modify Telegram messages. It does not change Alertmanager or Telegraf. It does not maintain a local database, cache, indexer, daemon, or server.

The CLI authenticates as the operator's Telegram user account and reads only channels that account can already access. This is intentional because the goal is querying existing channel history, which is better suited to Telegram client APIs over MTProto than to the Bot API.

## Goals

- Provide a Go CLI that can be distributed as a single local tool for operators.
- Authenticate locally as the operator's Telegram account.
- Search an accessible Telegram channel for alert messages.
- Support filters for text, period, host or instance, alert name, severity, and status.
- Return stable JSON by default for LLMs, scripts, and coding agents.
- Preserve the raw Telegram message text in every result.
- Extract structured alert fields where the message template allows it.
- Keep all session and configuration data local to the operator's machine.
- Avoid storing secrets in normal config files or printing them in logs.

## Non-goals for v1

- MCP server support.
- Web UI.
- Local cache, database, index, or background daemon.
- Centralized multi-user service.
- Centralized audit trail.
- Direct integration with Alertmanager or Telegraf.
- Sending, editing, deleting, acknowledging, or replying to Telegram messages.
- Built-in LLM summarization inside the CLI.
- Perfect parsing of every possible alert template.

## Technical direction

Use Go and the `gotd/td` Telegram MTProto client library.

Reasons:

- The project needs a Go CLI.
- A Go-native MTProto client avoids shipping TDLib native binaries in v1.
- MTProto user authentication is appropriate for reading historical channel messages accessible to the operator.
- Telegram methods such as `messages.search` and `messages.getHistory` match the required read-only search behavior.

References:

- `gotd/td`: https://github.com/gotd/td
- `messages.search`: https://core.telegram.org/method/messages.search
- `messages.getHistory`: https://core.telegram.org/method/messages.getHistory
- Telegram APIs overview: https://core.telegram.org/

## Architecture

Logical flow:

```text
Operator or LLM/agent
        |
        v
tg-alerts CLI
        |
        v
Local config + local Telegram session
        |
        v
Telegram MTProto via gotd/td
        |
        v
Alert channel
        |
        v
Tolerant alert parser
        |
        v
JSON or NDJSON on stdout
```

Suggested package boundaries:

- `cmd`: command definitions, flag parsing, validation, and process exit behavior.
- `config`: config file loading, environment variable resolution, defaults, and path discovery.
- `auth`: login, auth status, logout, and session lifecycle.
- `telegram`: MTProto client setup, channel resolution, history lookup, search calls, and retry/rate-limit handling.
- `search`: search request model, pagination model, max-scan behavior, and local filter application.
- `parser`: tolerant extraction of alert fields from raw Telegram messages.
- `output`: JSON and NDJSON serialization, stable error payloads, and stdout/stderr separation.

Each package should expose a narrow interface. The parser should not know about Telegram transport. The output layer should not know how Telegram is queried. The command layer should orchestrate rather than implement business logic.

## Commands

### `auth login`

Authenticates the local operator account.

```bash
tg-alerts auth login
```

Behavior:

- Prompts for phone number, Telegram code, and 2FA password if needed.
- Prompts must go to stderr, not stdout.
- Writes the local session file with restrictive permissions where supported.
- Never prints the phone code, 2FA password, `api_hash`, or raw session data.

### `auth status`

Reports whether the CLI has a usable local Telegram session.

```bash
tg-alerts auth status --format json
```

Example:

```json
{
  "ok": true,
  "authenticated": true,
  "account": {
    "phone_redacted": "+55******1234",
    "username": "operador"
  },
  "session": {
    "exists": true,
    "path": "~/.local/share/tg-alerts/session.json"
  }
}
```

### `auth logout`

Removes the local CLI session.

```bash
tg-alerts auth logout
```

In v1, this only removes the local session used by the CLI. It does not need to revoke all sessions for the Telegram account.

### `search`

Searches a Telegram channel and returns alert results.

```bash
tg-alerts search \
  --channel "@canal-alertas" \
  --query "cpu" \
  --since 24h \
  --limit 50
```

Structured filters:

```bash
tg-alerts search \
  --channel "@canal-alertas" \
  --severity critical \
  --status firing \
  --host srv-01 \
  --alertname HighCPUUsage
```

Flags:

- `--channel`: required unless configured as default.
- `--query`: optional text query.
- `--since`: optional lower time bound, for example `24h`, `7d`, or an RFC3339 timestamp.
- `--until`: optional upper time bound.
- `--host`: optional structured host filter.
- `--instance`: optional structured instance filter.
- `--alertname`: optional structured alert name filter.
- `--severity`: optional structured severity filter.
- `--status`: optional structured status filter.
- `--limit`: max number of returned results; default 50.
- `--page-size`: Telegram page size; default 100.
- `--max-scan`: max number of messages inspected locally; default 1000.
- `--timeout`: operation timeout; default 30s.
- `--format`: `json` by default; `ndjson` optional.
- `--page-token`: opaque continuation token from a previous response.

### `get`

Fetches one Telegram message by ID and emits the same result schema used by `search`.

```bash
tg-alerts get --channel "@canal-alertas" --message-id 12345
```

## Search behavior

When `--query` is present, use Telegram's search capability for the channel. The remote search should narrow by channel, text, time range, and pagination where possible. Structured filters are still applied locally after parsing because Telegram does not know alert-specific fields such as `severity`, `host`, or `alertname`.

When `--query` is absent but structured filters are present, read recent channel history and apply local parsing and filtering. This allows commands such as:

```bash
tg-alerts search --channel "@canal-alertas" --severity critical --since 24h
```

The CLI must distinguish between returned result count and scanned message count.

- `--limit` controls how many matching results are returned.
- `--max-scan` controls how many Telegram messages may be inspected to find those matches.

If `--max-scan` is reached before the result set is complete, return `ok: true` with a warning and pagination metadata.

## Pagination

JSON responses include explicit pagination metadata.

Example:

```json
{
  "ok": true,
  "results": [],
  "pagination": {
    "has_more": true,
    "next": {
      "page_token": "opaque-token-v1"
    }
  }
}
```

The page token should be opaque to callers. Internally it may encode Telegram offsets, channel identity, query text, time bounds, page size, and local filter state. The implementation should validate that a page token matches the current request shape or reject it with `QUERY_INVALID`.

## Output contract

Stdout is reserved for JSON or NDJSON. Stderr is for prompts, logs, warnings intended for humans, and authentication interaction.

Default JSON success shape:

```json
{
  "ok": true,
  "query": {
    "channel": "@canal-alertas",
    "text": "cpu",
    "since": "24h",
    "until": null,
    "filters": {
      "host": "srv-01",
      "instance": null,
      "alertname": "HighCPUUsage",
      "severity": "critical",
      "status": "firing"
    },
    "limit": 50
  },
  "results": [
    {
      "source": "telegram",
      "channel": "@canal-alertas",
      "message_id": 12345,
      "timestamp": "2026-05-29T13:20:00Z",
      "text": "HighCPUUsage firing critical srv-01...",
      "fields": {
        "host": "srv-01",
        "instance": "srv-01:9100",
        "alertname": "HighCPUUsage",
        "severity": "critical",
        "status": "firing",
        "summary": "CPU usage above threshold"
      },
      "parse": {
        "confidence": "medium",
        "matched_patterns": ["severity_label", "alertname_label", "host_candidate"],
        "missing_fields": []
      }
    }
  ],
  "pagination": {
    "has_more": false,
    "next": null
  }
}
```

Default JSON error shape:

```json
{
  "ok": false,
  "error": {
    "code": "CHANNEL_ACCESS_DENIED",
    "message": "The authenticated Telegram account cannot access this channel.",
    "hint": "Join the channel with this Telegram account and try again."
  }
}
```

NDJSON mode emits one JSON object per result. If warnings or pagination are needed in NDJSON mode, emit a final metadata object with a distinct `type`, for example:

```json
{"type":"result","source":"telegram","channel":"@canal-alertas","message_id":12345}
{"type":"meta","ok":true,"pagination":{"has_more":false,"next":null}}
```

## Parser design

The parser must preserve the raw Telegram message text in every result. It then attempts tolerant extraction of known alert fields.

Fields to extract in v1:

- `alertname`
- `status`
- `severity`
- `host`
- `instance`
- `summary`
- `description`
- `job`
- `environment`
- `namespace`

Supported style examples:

```text
alertname: HighCPUUsage
Alert: HighCPUUsage
alert_name = HighCPUUsage
status: firing
Status: RESOLVED
severity="critical"
host: srv-01
instance: srv-01:9100
```

Prometheus/Alertmanager-style blocks:

```text
Labels:
  alertname = HighCPUUsage
  severity = critical
  instance = srv-01:9100

Annotations:
  summary = CPU usage above threshold
```

Example parsed block:

```json
{
  "fields": {
    "alertname": "HighCPUUsage",
    "status": "firing",
    "severity": "critical",
    "host": "srv-01",
    "instance": "srv-01:9100",
    "summary": "CPU usage above threshold"
  },
  "parse": {
    "confidence": "medium",
    "matched_patterns": [
      "alertname_label",
      "severity_label",
      "instance_label",
      "status_keyword"
    ],
    "missing_fields": []
  }
}
```

Confidence levels:

- `high`: found `alertname`, `status`, and at least one target identifier such as `host` or `instance`.
- `medium`: found some main fields but missed at least one important field.
- `low`: inferred little; raw text remains the main source of truth.

Normalization:

- `severity`: compare case-insensitively and output lowercase where known, for example `critical` or `warning`.
- `status`: compare case-insensitively and output lowercase where known, for example `firing` or `resolved`.
- `host` and `instance`: preserve original output, compare case-insensitively.
- `alertname`: preserve original output, compare case-insensitively in v1.

Filter fallback rule:

If a structured filter is provided and the parser cannot extract the corresponding field from a message, that message does not match the filter. This avoids false positives for agents.

## Local config and session storage

Use OS-appropriate user directories. Linux examples:

```text
~/.config/tg-alerts/config.yaml
~/.local/share/tg-alerts/session.json
```

The implementation should use platform-aware path discovery rather than hard-coded Linux-only paths.

Example config:

```yaml
telegram:
  api_id: 123456
  api_hash_env: TG_ALERTS_API_HASH

defaults:
  channel: "@canal-alertas"
  format: "json"
  limit: 50
```

Rules:

- `api_hash` should come from environment variable in v1.
- Config files should not contain `api_hash` in plaintext by default.
- Session files should use restrictive file permissions where supported.
- Debug output must still redact secrets.
- `auth status` may show redacted phone and username, but not secrets.

## Error codes

Stable v1 error codes:

- `AUTH_REQUIRED`: session missing, invalid, or expired.
- `CHANNEL_NOT_FOUND`: channel username or identifier could not be resolved.
- `CHANNEL_ACCESS_DENIED`: authenticated account cannot access the channel.
- `QUERY_INVALID`: invalid flags, invalid page token, invalid time range, or ambiguous input.
- `TELEGRAM_RATE_LIMITED`: Telegram requested a wait before retrying.
- `TELEGRAM_UNAVAILABLE`: temporary Telegram or network failure.
- `TIMEOUT`: operation exceeded the configured timeout.
- `CONFIG_INVALID`: configuration file or required environment variable is invalid.

Warnings in successful responses:

- `MAX_SCAN_REACHED`: local scan limit reached before exhausting potential results.
- `PARSE_LOW_CONFIDENCE`: one or more results have low parse confidence.
- `PARTIAL_RESULTS`: some messages could not be processed but the command returned usable results.

## Security considerations

The session file grants access equivalent to the authenticated operator session for this CLI. Treat it as sensitive.

Security rules:

- Do not print raw session data.
- Do not print `api_hash`.
- Do not print login code or 2FA password.
- Do not write secrets to normal config files.
- Store session with restrictive local permissions where supported.
- Keep stdout parseable; do not mix human login prompts into stdout.
- Do not attempt to bypass Telegram channel permissions.
- Read only from channels the operator account can already access.

## Testing strategy

### Parser unit tests

Cover representative alert template variants.

Cases:

```text
alertname: HighCPUUsage
status: firing
severity: critical
host: srv-01
instance: srv-01:9100
summary: CPU above threshold
```

```text
Labels:
  alertname = DiskFull
  severity = warning
  instance = srv-02:9100

Annotations:
  summary = Disk usage above 85%
```

```text
🚨 FIRING
Alert: MemoryPressure
Host: srv-03
Severity: Critical
```

Assertions:

- Known fields are extracted.
- Raw text is preserved.
- Confidence is assigned consistently.
- Case normalization works.
- Missing fields are represented consistently.

### Output contract tests

Assertions:

- stdout always contains valid JSON or valid NDJSON.
- Operational errors return `ok: false` in JSON mode.
- Successful partial results return `ok: true` with `warnings`.
- Result objects always include `message_id`, `timestamp`, `channel`, and raw text.
- Secrets are not present in output, including debug-like error cases.

### Telegram integration validation

Use a private development channel with representative alert messages.

Manual validation commands:

```bash
tg-alerts auth login
tg-alerts auth status
tg-alerts search --channel "@canal-dev-alertas" --query "cpu" --limit 5
tg-alerts search --channel "@canal-dev-alertas" --severity critical --since 24h
tg-alerts get --channel "@canal-dev-alertas" --message-id <id>
tg-alerts auth logout
```

## Acceptance criteria

1. An operator can authenticate a Telegram account locally.
2. The CLI can query a channel accessible to that account.
3. `search` supports text, period, host or instance, alert name, severity, and status filters.
4. JSON grouped output is the default.
5. NDJSON is available as an optional format.
6. Every result includes raw text and extracted fields.
7. Structured filters are applied locally after parser extraction.
8. No local database, cache, indexer, server, or daemon exists in v1.
9. Session and config remain on the operator machine.
10. Secrets are never printed to stdout or stderr.
11. Errors use stable structured codes.
12. The CLI remains read-only against Telegram.

## Open decisions resolved

- Language: Go.
- Telegram access model: MTProto user client, not Bot API.
- Runtime location: operator machine only.
- Cache/index: no cache in v1.
- Main consumer: LLMs and agents via stdout.
- Default output: grouped JSON.
- Optional output: NDJSON.
- Parser strategy: tolerant extraction plus raw text preservation.

## Spec self-review

- Placeholder scan: no unresolved placeholder markers remain.
- Internal consistency: architecture, commands, parser, output, security, and tests all match a local read-only CLI.
- Scope check: v1 is narrow enough for a single implementation plan.
- Ambiguity check: the important behavior around stdout, local filters, missing parsed fields, pagination, and session storage is explicit.
