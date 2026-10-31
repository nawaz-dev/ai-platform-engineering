# 01 — vLLM Deployment and Connectivity

## Verify Kubernetes resources

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving get deployments,pods,svc,pvc
kubectl --context k3d-fde-gpu-v2 -n llm-serving rollout status deployment/vllm
```

## Access OpenAI-compatible API

In a dedicated terminal:

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving port-forward svc/vllm 8000:8000
```

In another terminal:

```bash
curl -fsS http://localhost:8000/health
curl -fsS http://localhost:8000/v1/models | jq .
```

## Manual smoke test — verified passed

```bash
VLLM_URL=http://localhost:8000 MODEL=qwen-small bash tests/smoke/vllm-smoke-test.sh
```

Previously observed: health PASS, model available PASS, chat completion PASS.

## Inspect deployment configuration before tuning

```bash
kubectl --context k3d-fde-gpu-v2 -n llm-serving get deployment vllm -o json   | jq '.spec.template.spec.containers[] | {name,image,command,args,resources}'
```

Record the exact image digest, command, args, resource limits, chart revision, and values commit in the benchmark report. These values have **not yet been captured** in this runbook.

## GitOps discipline

Make chart changes in `ai-platform-engineering/helm/vllm` and environment overrides in `ai-gitops/environments/dev/vllm/values.yaml`. Validate with `helm lint` and `helm template`, then use the established CI/promotion workflow. Avoid modifying live deployments as the source of truth.
