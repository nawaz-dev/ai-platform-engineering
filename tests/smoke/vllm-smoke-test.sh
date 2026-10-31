
#!/usr/bin/env bash
set -euo pipefail

VLLM_URL="${VLLM_URL:-http://localhost:8000}"
MODEL="${MODEL:-qwen-small}"

echo "=================================="
echo "vLLM Post-Deployment Smoke Test"
echo "=================================="

echo "[1/3] Checking vLLM health..."

curl --fail --silent --show-error \
  --max-time 15 \
  "${VLLM_URL}/health" > /dev/null

echo "PASS: vLLM health"

echo "[2/3] Checking model availability..."

MODELS_RESPONSE=$(curl --fail --silent --show-error \
  --max-time 15 \
  "${VLLM_URL}/v1/models")

echo "$MODELS_RESPONSE" | jq -e \
  --arg model "$MODEL" \
  '.data | any(.id == $model)' > /dev/null

echo "PASS: Model ${MODEL} available"

echo "[3/3] Testing chat completion..."

RESPONSE=$(curl --fail --silent --show-error \
  --max-time 120 \
  -X POST "${VLLM_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg model "$MODEL" \
    '{
      model: $model,
      messages: [
        {
          role: "user",
          content: "Reply with exactly the word READY."
        }
      ],
      max_tokens: 32,
      temperature: 0
    }')")

echo "$RESPONSE" | jq -e \
  '.choices[0].message.content |
   strings |
   length > 0' > /dev/null

echo "PASS: Chat completion returned non-empty content"

echo "=================================="
echo "ALL SMOKE TESTS PASSED"
echo "=================================="
