
# to run concurrent requests
mkdir -p tests/benchmark
chmod +x tests/benchmark/vllm-concurrency.sh

bash tests/benchmark/vllm-concurrency.sh

# to monitor metrics
watch -n 1 \
'curl -s http://localhost:8000/metrics |
grep -E "^vllm:(num_requests_running|num_requests_waiting|kv_cache_usage_perc)\{"'

# to monitor GPU
watch -n 1 nvidia-smi