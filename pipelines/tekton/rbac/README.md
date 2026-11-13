# Tekton RBAC

`serviceaccount.yaml` provides the task service account with API token automount disabled. `eventlistener-rbac.yaml` provides the listener account, namespace permissions to create PipelineRuns and read trigger resources/webhook secrets, and cluster-scoped permission to discover ClusterInterceptors.

Apply with cluster-admin access as described in the [setup guide](../README.md).


oc adm policy add-scc-to-user privileged `
  -z fde-tekton-sa `
  -n ai-platform-ci

oc adm policy add-scc-to-user privileged `
  -z fde-tekton-sa `
  -n ai-platform-ci

oc auth can-i use scc/privileged `
  --as=system:serviceaccount:ai-platform-ci:fde-tekton-sa