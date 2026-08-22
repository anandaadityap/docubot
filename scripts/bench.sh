#!/usr/bin/env bash
# Measure chat latency against a running DocuBot API, including time-to-first-token.
# Usage: BASE_URL=http://localhost:8080 ./scripts/bench.sh
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
MSG="${MSG:-Gimana cara reset password?}"
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
first_ns=""

# Stream SSE; record wall clock when the first "event: token" line arrives.
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$tmp"
  if [[ -z "$first_ns" && "$line" == event:\ token ]]; then
    first_ns=$(date +%s%N)
  fi
done < <(curl -sN -H "Content-Type: application/json" -d "{\"message\":\"$MSG\"}" "$BASE_URL/api/v1/chat" || true)

end_ns=$(date +%s%N)
total_ms=$(( (end_ns - start_ns) / 1000000 ))
ttft_ms=""
if [[ -n "$first_ns" ]]; then
  ttft_ms=$(( (first_ns - start_ns) / 1000000 ))
fi

latency=$(grep -o '"latency_ms":[0-9]*' "$tmp" | tail -n1 | grep -o '[0-9]*' || true)
tokens=$(grep -o '"total_tokens":[0-9]*' "$tmp" | tail -n1 | grep -o '[0-9]*' || true)
latency="${latency:-$total_ms}"
tokens="${tokens:-0}"

cost=$(awk -v t="$tokens" -v o="$OUT_PER_M" 'BEGIN { printf "%.6f", (t/1000000.0)*o }')

echo "== chat =="
echo "total_wall_ms=$total_ms"
echo "ttft_ms=${ttft_ms:-n/a}"
echo "server_latency_ms=$latency"
echo "total_tokens=$tokens"
echo "est_cost_usd=$cost  (using $OUT_PER_M USD / 1M output tokens; IN_PER_M=$IN_PER_M unused when split unknown)"
echo
echo "== first 8 SSE lines =="
head -n 8 "$tmp" || true
rm -f "$tmp"
