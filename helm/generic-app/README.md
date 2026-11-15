# FDE generic-app Helm chart

Reusable **stateless** application chart for deployment into `ai-platform` on `dev-cluster`. Each app should use its own Argo CD Application / Helm release. This is a starter implementation; validate the rendered output and application requirements before production use.

## Contents

- `Chart.yaml`, `values.yaml`, `values.schema.json`
- Templates: Deployment, Service, ServiceAccount, ConfigMap, PVC, HPA, Ingress, OpenShift Route, NetworkPolicy, ServiceMonitor, PDB, Job, CronJob
- `examples/`: Model Gateway, RAG, existing/shared PVC, init-only PVC, NFS PV/PVC, dynamic PVC

## Render and install

```bash
helm lint ./generic-app
helm template model-gateway ./generic-app --namespace ai-platform \
  -f ./generic-app/examples/values-model-gateway.yaml > model-gateway-rendered.yaml
kubectl --context k3d-dev-cluster apply --dry-run=server -f model-gateway-rendered.yaml
```

Argo CD example (`Application` spec fragment):

```yaml
spec:
  destination:
    server: https://kubernetes.default.svc
    namespace: ai-platform
  source:
    repoURL: https://github.com/your-org/your-repo.git
    targetRevision: main
    path: helm/generic-app
    helm:
      releaseName: model-gateway
      valueFiles:
        - examples/values-model-gateway.yaml
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

Replace Git URL and image references before deployment. Local cluster address works if Argo CD is installed on `dev-cluster`.

## Storage

- `volumes` are pod-level storage sources. Each container lists its own `volumeMounts`.
- `persistentVolumeClaims` creates PVC objects owned by THIS release. **Do not** create an existing/shared claim in multiple releases. Provision shared storage in an infrastructure chart and reference it with `volumes[].persistentVolumeClaim.claimName`.
- `examples/values-init-only-pvc.yaml` mounts source PVC **only** in an init container; main container mounts a copied `emptyDir`. Source PVC remains in pod `volumes` by design.
- For multi-node, multi-pod shared writes, require storage that supports RWX (NFS or supported CSI provisioner). The usual k3d local-path class does not provide generic multi-node RWX access.
- NFS server and exported directory in examples are placeholders; configure an actual NFS server, network access, and filesystem permissions. NFS static PV is cluster-scoped; PVC is namespaced.

## Important notes

- Probe endpoints `/health` and `/ready` are **examples** and must exist in the app.
- `readOnlyRootFilesystem: true` usually requires mounting a writable `/tmp` or another application-specific path.
- `serviceMonitor.enabled` needs the ServiceMonitor CRD and matching Service port named `metrics` (not present in default Service). Prometheus must select the namespace and labels. This chart does not install Prometheus Operator.
- `route.enabled` requires OpenShift, not k3d. Do not enable it on `dev-cluster`.
- HPA needs metrics-server/custom-metrics adapters; avoid enabling HPA with RWO persistence that cannot safely scale.
- NetworkPolicy examples may block DNS, cross-cluster model inference, and ingress traffic; tailor rules before enabling.
- `logging`, `tracing`, `existingSecrets`, and `gpu` values document application-level configuration, but do **not** automatically create collector integrations, secret resources or GPU limits. Configure respective `env`, `resources`, and cluster services explicitly.
- `jobs` and `cronJobs` have independent volumes; they do not inherit Deployment volumes. The example templates implement only their documented simple fields.
- PVC lifecycle: Helm/Argo prune can delete PVC objects and potentially storage. Keep valuable/shared claims out of individual app releases and set retention policy as appropriate.
- For Redis, PostgreSQL, and other stateful data stores use dedicated StatefulSet-oriented charts.
