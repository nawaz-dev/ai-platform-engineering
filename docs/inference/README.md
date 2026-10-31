# vLLM Inference Engineering Runbook

This runbook documents local vLLM serving, smoke testing, Prometheus metrics, concurrency benchmarking, and controlled optimization on a Windows/WSL2 + k3d GPU setup.

## Environment (observed)

- GPU: NVIDIA GeForce RTX 4050 Laptop GPU, 6,141 MiB VRAM (WSL2)
- Kubernetes context: `k3d-fde-gpu-v2`; namespace: `llm-serving`
- Helm release / service: `vllm`; API port: `8000`
- Model alias: `qwen-small` (Qwen2.5-0.5B-Instruct)
- GitOps: Argo CD; chart under `helm/vllm`; environment values in separate `ai-gitops` repository

## Workflow

1. [Deployment and connectivity](01-vllm-deployment.md)
2. [GPU and vLLM metrics](02-gpu-monitoring.md)
3. [Concurrency benchmark and results](03-performance-benchmarking.md)
4. [Controlled tuning plan](04-inference-optimization.md)
5. [Troubleshooting](05-troubleshooting.md)

## Verified versus pending

**Verified:** serving endpoint responds, manual smoke test passes, 1/5/10/20-user concurrency benchmark succeeded, metrics endpoint available.

**Pending:** per-request TTFT/P50/P95 latency under load, active scheduler/KV-cache sampling, reproducible tuning comparisons, GitOps promotion end-to-end verification, automated PostSync smoke-test hook.

Do not commit tokens, kubeconfigs, secrets, model credentials, or environment-specific private data.
