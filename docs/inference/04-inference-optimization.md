# 04 — Controlled Inference Optimization Plan

## Goal

Improve user-facing latency and/or aggregate throughput without increasing errors or exhausting 6 GB GPU memory.

## Step 1: Capture the actual deployment

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving get deployment vllm -o json   | jq '.spec.template.spec.containers[] | {name,image,command,args,resources}'
```

Also capture chart revision, model revision, GPU driver, vLLM version, and values YAML.
```bash
kubectl --context k3d-fde-gpu-v2 \
  -n llm-serving get deployment vllm \
  -o json | jq '.spec.template.spec.containers[] |
    {name, resources}'
```

And capture the scheduler metrics during an active benchmark:
```bash
while true; do
  curl -s http://localhost:8000/metrics |
    grep -E '^vllm:(num_requests_running|num_requests_waiting|kv_cache_usage_perc)\{'
  sleep 0.5
done
```

## Step 2: Establish a reproducible baseline

1. Warm up the model with several requests.
2. Repeat the same workload at least three times per concurrency level.
3. Record throughput, success rate, TTFT, end-to-end latency P50/P95, scheduler gauges, GPU utilization, and VRAM.
4. Separate warm/cold start measurements.

## Step 3: Tune one setting at a time

| Candidate | What to investigate | Caveat |
|---|---|---|
| `max_num_seqs` | Scheduling/concurrent sequence capacity | Higher is not always faster; consumes KV cache |
| `max_num_batched_tokens` | Per-iteration token budget | Impacts prefill/decode scheduling and latency |
| `gpu_memory_utilization` | vLLM GPU memory budget | 6 GB GPU has limited headroom; avoid OOM |
| `max_model_len` | Context window / KV cache needs | Do not reduce below application requirements |
| Chunked prefill | Long prompt responsiveness | Availability/defaults depend on vLLM version |

**Do not change settings until actual startup args and resource limits are captured.** Apply Helm/GitOps changes, not ad-hoc drift in Kubernetes.

## Step 4: Compare and decide

Use the same benchmark input, concurrency, model, environment, and repetitions. Record before/after results and revert if latency, reliability, or memory pressure worsens.

## Step 5: Next platform milestones

Model Gateway (FastAPI, auth, rate limiting, routing) → enterprise RAG (ingestion, embeddings, vector search, grounded answers) → tool-calling/agents → evaluation and observability → production hardening.
