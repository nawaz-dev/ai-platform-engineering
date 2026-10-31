cd ~/projects/git-nawaz-dev/ai-platform-engineering

chmod +x tests/smoke/vllm-smoke-test.sh

VLLM_URL=http://localhost:8000 \
MODEL=qwen-small \
bash tests/smoke/vllm-smoke-test.sh