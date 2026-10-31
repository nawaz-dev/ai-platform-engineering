
#!/usr/bin/env bash
set -euo pipefail

URL="${VLLM_URL:-http://localhost:8000}"
MODEL="${MODEL:-qwen-small}"

mkdir -p /tmp/vllm-benchmark

for USERS in 1 5 10 20; do
  echo "================================"
  echo "Testing $USERS concurrent users"
  echo "================================"

  PAYLOAD=$(jq -n --arg model "$MODEL" '{
    model: $model,
    messages: [
      {
        role: "user",
        content: "Explain Kubernetes in approximately 100 words."
      }
    ],
    max_tokens: 128,
    temperature: 0
  }')

  START=$(date +%s%3N)

  for i in $(seq 1 "$USERS"); do
    (
      curl -sS --max-time 180 \
        -o "/tmp/vllm-benchmark/${USERS}-${i}.json" \
        -w "%{http_code}\n" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" \
        "$URL/v1/chat/completions" \
        > "/tmp/vllm-benchmark/${USERS}-${i}.status"
    ) &
  done

  wait

  END=$(date +%s%3N)
  ELAPSED=$(awk -v ms="$((END - START))" \
    'BEGIN {printf "%.3f", ms / 1000}')

  SUCCESS=0
  TOKENS=0

  for i in $(seq 1 "$USERS"); do
    FILE="/tmp/vllm-benchmark/${USERS}-${i}.json"
    STATUS=$(cat "/tmp/vllm-benchmark/${USERS}-${i}.status")

    if [ "$STATUS" = "200" ] &&
       jq -e '.choices[0].message.content | strings | length > 0' \
         "$FILE" >/dev/null 2>&1; then
      SUCCESS=$((SUCCESS + 1))
      COUNT=$(jq -r '.usage.completion_tokens // 0' "$FILE")
      TOKENS=$((TOKENS + COUNT))
    fi
  done

  TPS=$(awk -v t="$TOKENS" -v s="$ELAPSED" \
    'BEGIN {if (s > 0) printf "%.2f", t/s; else print "0"}')

  echo "Successful requests: $SUCCESS/$USERS"
  echo "Total completion tokens: $TOKENS"
  echo "Elapsed time: ${ELAPSED}s"
  echo "Aggregate output throughput: ${TPS} tokens/sec"
  echo
done
