# 03 — Performance Benchmarking

## Objective

Measure request success, aggregate output tokens/sec, and batch completion time for different numbers of concurrent chat-completion requests. Then extend to per-request TTFT, end-to-end latency percentiles, and time per output token.

## Preconditions

- vLLM service reachable at `http://localhost:8000` through port-forward.
- `jq`, `curl`, `bash`, and GNU `date` available.
- Model alias `qwen-small` exposed.

## Run

```bash
VLLM_URL=http://localhost:8000 MODEL=qwen-small bash tests/benchmark/vllm-concurrency.sh
```

Workload: one user prompt (`Explain Kubernetes in approximately 100 words.`), `max_tokens=128`, `temperature=0`, and concurrency 1, 5, 10, 20. Each observed request generated 128 completion tokens. This is a **single-prompt, single-run** benchmark, not a production capacity certification.

## Measured results (2026-10-26, latest supplied run)

| Concurrent requests | Successful | Completion tokens | Batch elapsed (s) | Aggregate output tok/s |
|---:|---:|---:|---:|---:|
| 1 | 1/1 | 128 | 3.122 | 41.00 |
| 5 | 5/5 | 640 | 8.875 | 72.11 |
| 10 | 10/10 | 1,280 | 17.809 | 71.87 |
| 20 | 20/20 | 2,560 | 36.056 | 71.00 |

Previous run: 10 users = 71.19 tok/s; 20 users = 74.14 tok/s. Variability is expected.

## Interpretation

- All requests succeeded in the reported tests.
- Aggregate throughput improved between 1 and 5 concurrent requests, then plateaued around 71–74 tok/s for this workload.
- Batch completion time rose as request count increased.
- The plateau alone does not establish the underlying bottleneck.

## Earlier Prometheus observation (single sample)

- TTFT histogram: count=1, sum=1.6875288486480713 s → observed mean 1.688 s.
- Inter-token latency histogram: count=1, sum=0.2610242600021593 s → observed 0.261 s.

One observation is not statistically useful for P50/P95 or generalized latency claims.

## Next measurements

Capture per-request response latency, streaming TTFT, P50/P95/P99, throughput under varied prompt/output lengths, failures/timeouts, and concurrent scheduler/KV-cache gauges. Run multiple repetitions with warmup; record environment and Git commit SHA for each run.
