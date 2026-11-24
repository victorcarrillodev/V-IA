#!/bin/sh
# Comprueba con el servicio real que el MCP remoto de Context7 responde al handshake.
# Solo envía un `initialize` anónimo (sin datos del proyecto ni del usuario). Requiere red.
set -eu
URL=${CONTEXT7_URL:-https://mcp.context7.com/mcp}
out=$(curl -sS -m 20 -X POST "$URL" -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"v-ia-check","version":"0"}}}')
echo "$out" | grep -q '"serverInfo"' && echo "✓ Context7 responde al handshake MCP" || { echo "✗ respuesta inesperada de $URL"; exit 1; }
