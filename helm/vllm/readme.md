<!-- Purpose: Explains the vLLM chart's template structure, compatibility, and validation. -->
# vLLM inference chart

Deploys the vLLM model server with GPU scheduling, health probes, persistent
model caching, and optional network exposure. Configuration continues to use
the existing `values.yaml` keys, including the legacy registry settings.

## Template responsibilities

| File | Purpose |
| --- | --- |
| `deployment.yaml` | Assembles the pod, scheduling, service account, and companion containers. |
| `_container.tpl` | Renders the inference container's image, model arguments, environment, ports, probes, and resources. |
| `_storage.tpl` | Builds volumes and matching mounts together for model caches, ConfigMaps, and Secrets; appends custom storage. |
| `_helpers.tpl` | Resolves resource names, protected identity labels, image references, and the service account. |
| `service.yaml` | Maps the primary inference port and any additional ports to the selected pods. |
| `ingress.yaml` | Renders host/path routing, TLS references, and backend ports for supported ingress API versions. |
| `hpa.yaml` | Validates replica bounds and CPU/memory targets, then creates the autoscaler. |
| `pvc.yaml` | Creates model-cache claims only when both PVC enablement and creation are requested. |
| `configmaps.yaml` | Creates configured non-sensitive configuration objects. |
| `secrets.yaml` | Creates configured Secret objects. |
| `serviceaccount.yaml` | Creates the pod identity when requested; does not grant RBAC permissions. |
| `NOTES.txt` | Prints model-server status, log, connection, and cache instructions. |

## Compatibility

The refactor preserves the existing chart identity, resource names, `app` pod
selector, and default rendered resource fields. `values.yaml` is unchanged.
The configured model image, GPU allocation, NVIDIA runtime, service-link setting,
DXCore mount, and existing cache claim retain their current behavior.

- `commonLabels` adds metadata; chart-owned identity labels take precedence to
  keep the Deployment and Service selectors consistent.
- Image resolution prefers `image.ImageFullurl`, then the legacy global registry,
  then the workload registry, then a direct `image.ImagePath`. A registry requires
  an image path.
- Enabled features with empty lists omit empty pod fields such as `envFrom`,
  `volumes`, `volumeMounts`, and `initContainers`.
- Autoscaling requires integer replica bounds with `1 <= minReplicas <= maxReplicas`
  and at least one positive integer CPU or memory utilization target.
  With no `minReplicas` setting, the minimum is
  one. Disabling deployment creation also disables its autoscaler.
- With `pvc.create=false`, model-cache claims must already exist in the release
  namespace. The chart does not create those claims.

## Validate locally

Run from the repository root:

```sh
helm lint helm/vllm
helm template vllm helm/vllm --namespace llm-serving
```

Regression tests require Helm, Python 3, and PyYAML. They render into temporary
files and do not connect to a Kubernetes cluster:

```sh
python3 -m unittest discover -s helm/vllm/tests -p 'test_*.py' -v
```
