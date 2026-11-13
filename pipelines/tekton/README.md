# Monorepo Tekton CI

One GitHub webhook feeds `monorepo-github-listener`. Its GitHub interceptor verifies the signature and retrieves changed files; CEL filters start each affected application's pipeline at the exact push SHA. Only non-deletion pushes to `main` in `nawaz-dev/ai-platform-engineering` are accepted.

| Changed path | Pipeline |
| --- | --- |
| `applications/vllm/`, `helm/vllm/` | `vllm-ci-pipeline` |
| `applications/model-gateway/` | `model-gateway-ci` |
| `pipelines/tekton/` | Both |
| Other paths, including the Python gateway | Neither |

Changes touching both applications start both pipelines independently. Added, modified, and deleted paths participate in routing. Each run gets its own PVC. Shared tasks live in `tasks/`; the two small pipelines compose them. One TriggerBinding and one parameterized TriggerTemplate serve both routes. To add another application, add a pipeline and a trigger entry with its path filter, pipeline name, and GitOps file.

```text
GitHub push → EventListener → changed-path filters
                               ├─ vllm-ci-pipeline: Helm validation + application/chart scans
                               └─ model-gateway-ci: Go test/build + application scan
                                  ↓ after all checks in that pipeline succeed
                               GitOps promotion PR → merge → Argo CD
```

Go validation runs `go test ./...` and `go build ./...` with Go 1.26. Helm validation lints and renders the selected chart. Trivy scans application dependencies/secrets/configuration and chart configuration/secrets, failing on HIGH or CRITICAL findings. This workflow validates source and charts; it does not build or publish container images. The gateway deployment configuration and published image must already exist in your GitOps setup; this repository currently has no gateway Helm chart.

## Install

Requires Tekton Pipelines (`tekton.dev/v1`), Tekton Triggers with GitHub and CEL ClusterInterceptors, cluster-admin access for the interceptor RBAC, dynamic PVC provisioning, and outbound access to GitHub, image registries, Go dependencies, and the Trivy database.

```sh
kubectl create namespace ai-platform-ci --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n ai-platform-ci -f pipelines/tekton/rbac/
kubectl apply -n ai-platform-ci -f pipelines/tekton/tasks/
kubectl apply -n ai-platform-ci -f pipelines/tekton/pipelines/
kubectl apply -n ai-platform-ci -f pipelines/tekton/triggers/
```

Before sending events, provision these secrets in `ai-platform-ci` through your secret management process:

- `github-webhook-secret`, key `secretToken`: the same secret configured on the GitHub webhook.
- `github-gitops-token`, key `token`: a token with contents and pull-request write access to the GitOps repository.
- `dockerhub-credentials`: a `kubernetes.io/dockerconfigjson` secret for the `nawaz343` Docker Hub account. The `fde-tekton-sa` service account uses it to push `docker.io/nawaz343/model-gateway`.

For example, create the registry secret without storing the password in Git:

```sh
kubectl create secret docker-registry dockerhub-credentials \
  -n ai-platform-ci \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=nawaz343 \
  --docker-password='YOUR_DOCKER_HUB_TOKEN'

kubectl annotate secret dockerhub-credentials \
  -n ai-platform-ci \
  tekton.dev/docker-0=https://index.docker.io/v1/ \
  --overwrite
```

Configure GitHub to send **push** events as `application/json` to the listener endpoint. On OpenShift, customize the host in `pipelines/routes/github-webhook-route.yaml` and apply that file. Else expose `el-monorepo-github-listener` using your ingress. For private source repositories, configure `addChangedFiles.personalAccessToken` on both GitHub interceptors and provide separate authenticated checkout credentials; the default clone supports public repositories.

## GitOps contract

The shared promotion task defaults to `nawaz-dev/ai-gitops`, branch `main`. The triggers select `argocd/applications/vllm-dev.yaml` and `argocd/applications/model-gateway-dev.yaml`. Confirm these files exist in your GitOps repository before enabling webhooks. Each must contain exactly one Helm source with `repoURL: https://github.com/nawaz-dev/ai-platform-engineering.git` immediately followed by `targetRevision:`. Promotion updates that revision to the validated SHA and opens an application-specific PR. Argo CD deploys after that PR is merged, according to its sync policy. The task retains the existing `nawaz343/fde-gitops-tools:v1` image contract: Git, curl, jq, base64, awk, and sed must be available.

The two pipelines promote independently; a failure in one blocks only that application's promotion. Existing promotion branches and PRs are reused for a repeated application/SHA pair.

## Migration and verification

Applying renamed resources does not delete their old versions. After reviewing the new listener and updating the webhook URL, remove the old `vllm-github-listener`, `go-model-gateway-ci`, `vllm-github-binding`, `vllm-github-template`, `go-model-gateway-github-binding`, and `go-model-gateway-github-template` resources if installed. Remove the old OpenShift `vllm-github-webhook` Route after switching endpoints. Existing PipelineRuns are unaffected.

Check `kubectl get eventlisteners,pipelineruns -n ai-platform-ci` and listener logs after test pushes. A gateway-only change should create one `model-gateway-ci` run; a vLLM-only change should create one `vllm-ci-pipeline` run; changes to both should create two runs; a root README-only change should create none. A failed validation or scan must prevent the promotion task from starting.

Changed-file routing follows the [Tekton GitHub interceptor documentation](https://tekton.dev/docs/triggers/interceptors/#adding-changed-files).
