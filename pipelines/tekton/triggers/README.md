# Webhook routing

`eventlistener.yaml` verifies GitHub push signatures and filters changed paths for each application. Both entries use `triggerbinding.yaml` for the push SHA and `triggertemplate.yaml` to create a PipelineRun with the selected pipeline, GitOps file, task service account, and isolated PVC.

See the [routing table and setup guide](../README.md).
