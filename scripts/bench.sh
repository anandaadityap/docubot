#!/usr/bin/env bash
# Measure chat latency against a running DocuBot API.
# Usage: BASE_URL=http://localhost:8080 TOKEN=... ./scripts/bench.sh
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
MSG="${MSG:-Gimana cara reset password?}"
# DeepSeek-chat list price (USD / 1M tokens) — update if the provider changes rates.
IN_PER_M="${IN_PER_M:-0.27}"
OUT_PER_M="${OUT_PER_M:-1.10}"

if ! command -v curl >/dev/null; then
  echo "curl is required" >&2
  exit 1
fi

echo "== health =="
curl -fsS "$BASE_URL/healthz"
echo

tmp="$(mktemp)"
start_ns=$(date +%s%N)
first_ms=""

curl -sN -D - -o "$tmp" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"$MSG\"}" \
  "$BASE_URL/api/v1/chat" > "${tmp}.hdr" || true

# Approximate TTFT: time until file is non-empty (first SSE bytes).
# For a tighter number, parse after the request using timestamps in `done`.
end_ns=$(date +%s%N)
total_ms=$(( (end_ns - start_ns) / 1000000 ))

latency=$(grep -o '"latency_ms":[0-9]*' "$tmp" | tail -n1 | grep -o '[0-9]*' || true)
tokens=$(grep -o '"total_tokens":[0-9]*' "$tmp" | tail -n1 | grep -o '[0-9]*' || true)
latency="${latency:-$total_ms}"
tokens="${tokens:-0}"

# Cost estimate assumes all tokens are completion (upper bound) if split is unknown.
cost=$(awk -v t="$tokens" -v o="$OUT_PER_M" 'BEGIN { printf "%.6f", (t/1000000.0)*o }')

echo "== chat =="
echo "total_wall_ms=$total_ms"
echo "server_latency_ms=$latency"
echo "total_tokens=$tokens"
echo "est_cost_usd=$cost  (using $OUT_PER_M USD / 1M output tokens)"
echo
echo "== first 8 SSE lines =="
head -n 8 "$tmp" || true
rm -f "$tmp" "${tmp}.hdr"
