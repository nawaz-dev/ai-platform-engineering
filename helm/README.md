# Helm Charts

Stores reusable Helm charts for packaging and deploying platform services to Kubernetes and OpenShift.

1. ai-platform-engineering/helm/monitoring/values.yaml — Default configuration for the reusable Helm chart.

2. ai-gitops/environments/dev/monitoring/values.yaml — Environment-specific configuration (Docker images, PVC sizes, StorageClass, etc.).

How it works: Argo CD loads the default values first, then overrides them with values from ai-gitops.

Recommendation: Keep common defaults in ai-platform-engineering and only environment-specific overrides in ai-gitops. This avoids duplication.
