# Tekton RBAC

`serviceaccount.yaml` provides the task service account with API token automount disabled. `eventlistener-rbac.yaml` provides the listener account, namespace permissions to create PipelineRuns and read trigger resources/webhook secrets, and cluster-scoped permission to discover ClusterInterceptors.

Apply with cluster-admin access as described in the [setup guide](../README.md).
