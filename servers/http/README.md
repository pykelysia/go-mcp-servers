# `mcp-http` — HTTP client MCP server (Go)

Make outbound HTTP requests, with an optional host allowlist, default Authorization header, timeout, and response size cap. Comes with helpers to parse JSON paths and strip HTML to text.

## Tools

| Tool | Description |
|---|---|
| `http_get` | GET with optional headers. |
| `http_post` | POST with body + headers. |
| `http_put` | PUT with body + headers. |
| `http_delete` | DELETE with optional headers. |
| `http_request` | Generic — specify any method. |
| `parse_json` | Parse JSON, optionally extract a dotted path (`a.b.0.c`). |
| `parse_html` | Strip tags and return visible text. Skips `<script>` and `<style>`. |

Responses are returned as JSON:

```json
{
  "status": 200,
  "headers": { "Content-Type": "application/json" },
  "body": "{...}",
  "body_len": 1234,
  "truncated": false
}
```

## Environment

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `HTTP_DEFAULT_TIMEOUT` | no | `30` | Per-request timeout seconds. |
| `HTTP_MAX_RESPONSE_SIZE` | no | `5242880` (5MB) | Max bytes of body returned. Excess sets `truncated: true`. |
| `HTTP_AUTH_HEADER` | no | – | Default value for `Authorization`. Overridden per request via `headers`. |
| `HTTP_USER_AGENT` | no | `go-mcp-servers/http/1.0` | Default User-Agent. |
| `HTTP_ALLOWED_HOSTS` | no | – (no restriction) | Comma-separated hostnames. Hostname comparison is case-insensitive. |
| `LOG_LEVEL` | no | `info` | `debug` for verbose. |

## Claude Desktop config

```json
{
  "mcpServers": {
    "http": {
      "command": "/usr/local/bin/mcp-http",
      "env": {
        "HTTP_ALLOWED_HOSTS": "api.github.com,httpbin.org",
        "HTTP_DEFAULT_TIMEOUT": "15"
      }
    }
  }
}
```

## Security

- **SSRF defense**: when `HTTP_ALLOWED_HOSTS` is set, any URL with a host outside the list is rejected before the request is built. Be explicit — defaults to "allow all".
- **Response size cap**: bodies above `HTTP_MAX_RESPONSE_SIZE` are truncated, with `truncated: true` in the response so callers detect it.
- **Header injection**: per-request `headers` JSON is parsed via `encoding/json` and applied via `req.Header.Set`, which handles CRLF defensively.
- **Timeouts**: a context deadline kicks in even if the server stalls mid-stream.
