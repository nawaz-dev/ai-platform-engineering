# 02 — GPU Monitoring and vLLM Metrics

## NVIDIA GPU monitoring

```bash
nvidia-smi
nvidia-smi --query-gpu=timestamp,utilization.gpu,memory.used,memory.total,power.draw --format=csv -l 1
```

Run monitoring **during** a benchmark, not only after it completes. In WSL, NVIDIA-SMI utilization and power readings may not fully characterize inference efficiency.

## Prometheus metrics

```bash
curl -fsS http://localhost:8000/metrics | grep -E '^vllm:(num_requests_running|num_requests_waiting|kv_cache_usage_perc|time_to_first_token_seconds|inter_token_latency_seconds)'
```

To sample scheduler gauges during load:

```bash
while true; do
  date -Is
  curl -fsS http://localhost:8000/metrics     | grep -E '^vllm:(num_requests_running|num_requests_waiting|kv_cache_usage_perc)\{'
  sleep 0.5
done
```

Metrics interpretation:

- `num_requests_running`: currently scheduled requests.
- `num_requests_waiting`: requests queued for scheduling.
- `kv_cache_usage_perc`: KV-cache occupancy gauge, not total allocated VRAM.
- `time_to_first_token_seconds`: TTFT histogram; `sum/count` is a mean over observed requests, not a percentile.
- `inter_token_latency_seconds`: histogram of inter-token gaps; its reciprocal is not a robust measure of per-user throughput.

## Observed NVIDIA-SMI samples (2026-10-26)

- Idle utilization: approximately 0%.
- During load: many samples around 28–44%, with occasional lower readings.
- Allocated VRAM: steady at 4,064 / 6,141 MiB (about 66.2%).
- Peak utilization seen in supplied log: 44%.
- Idle scheduler/KV-cache readings after tests: all 0.

**Caution:** No synchronized scheduler/KV-cache gauge series was captured during the active benchmark. Do not infer GPU compute saturation, memory bandwidth saturation, or queue pressure from these readings alone.
