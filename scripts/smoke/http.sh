#!/usr/bin/env bash
# Smoke test for mcp-http. Starts a tiny netcat listener as a target.
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BIN=$ROOT_DIR/bin/mcp-http

if [ ! -x "$BIN" ]; then
    (cd "$ROOT_DIR/servers/http" && go build -o "$BIN" .)
fi

# parse_json is a pure operation (no network) — great for smoke.
REQUESTS='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1.0.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"parse_json","arguments":{"input":"{\"hello\":\"world\"}","path":"hello"}}}'

OUTPUT=$(LOG_LEVEL=error \
    /bin/sh -c 'echo "$1" | "$2"' _ "$REQUESTS" "$BIN" 2>/dev/null || true)

if ! echo "$OUTPUT" | grep -q '"name":"http_get"'; then
    echo "FAIL: http_get missing from tools/list" >&2
    echo "$OUTPUT" >&2
    exit 1
fi

if ! echo "$OUTPUT" | grep -q "world"; then
    echo "FAIL: parse_json did not return expected value" >&2
    echo "$OUTPUT" >&2
    exit 1
fi

echo "OK: http smoke passed"
