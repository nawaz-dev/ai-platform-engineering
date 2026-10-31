# 05 — Troubleshooting

## Cannot reach localhost:8000

Verify `kubectl ... port-forward svc/vllm 8000:8000` is running and service endpoints exist:

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving get svc vllm
kubectl --context k3d-fde-gpu-v2 -n llm-serving get endpoints vllm
kubectl --context k3d-fde-gpu-v2 -n llm-serving get pods
```

## Metrics all show zero

Scheduler and KV-cache gauges return to zero after requests finish. Sample while the load generator is running.

## GPU memory stays allocated when idle

Model weights and vLLM's memory reservations may remain resident even when compute utilization is 0%; this is not necessarily a leak.

## Throughput plateaus as concurrency rises

Check TTFT/latency, scheduler queue, KV-cache pressure, GPU/CPU behavior, and workload characteristics. A plateau does not by itself prove compute saturation.

## Benchmark reports failure

Inspect response files under `/tmp/vllm-benchmark` and the HTTP status codes. A successful HTTP response must also contain a non-empty chat message. Check pod logs:

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving logs deployment/vllm --tail=150
```

## GitOps reminder

Changes to Helm chart may require updating the pinned engineering-repository revision in the Argo CD Application. Updating values alone will not expose templates that are absent from the pinned chart revision.
