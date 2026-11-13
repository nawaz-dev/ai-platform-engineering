# Application pipelines

`vllm-ci-pipeline.yaml` validates the vLLM Helm chart and scans application/chart files. `model-gateway-ci.yaml` also tests and builds the Go gateway. Each pipeline promotes its validated SHA only after all its checks succeed.

See [setup, routing, and GitOps requirements](../README.md).
