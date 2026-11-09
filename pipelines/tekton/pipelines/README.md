# Tekton Pipeline Definitions

Composes reusable Tekton tasks into end-to-end workflows for building and validating platform services.


## Monorepo CI/CD Architecture — Tekton

### Overview

The `ai-platform-engineering` repository follows a **monorepo architecture**, where multiple applications share one Git repository but have independent CI pipelines.

**Objective:** Execute CI only for applications whose source files have changed, avoiding unnecessary builds.

### 1. Repository Structure

```text
ai-platform-engineering/
├── applications/
│   ├── model-gateway/
│   ├── rag-service/
│   ├── graph-db/
│   └── vllm/
│
├── pipelines/
│   └── tekton/
│       ├── triggers/
│       │   ├── eventlistener.yaml
│       │   ├── triggerbinding.yaml
│       │   └── triggertemplate.yaml
│       ├── model-gateway-ci.yaml
│       ├── rag-service-ci.yaml
│       ├── graph-db-ci.yaml
│       └── vllm-ci.yaml
│
└── README.md
```

### 2. CI Workflow

```text
Developer
    |
    | git push
    v
GitHub Repository
    |
    | Webhook Event
    v
Tekton EventListener
    |
    v
GitHub Interceptor
    |
    | Validate webhook signature
    | Retrieve changed files
    v
CEL Path Filters
    |
    ├── applications/model-gateway/**
    |       └── model-gateway-ci
    |
    ├── applications/rag-service/**
    |       └── rag-service-ci
    |
    ├── applications/graph-db/**
    |       └── graph-db-ci
    |
    └── applications/vllm/**
            └── vllm-ci
```

### 3. Path-Based Pipeline Triggers

| Changed files | Pipeline |
|---|---|
| `applications/model-gateway/**` | `model-gateway-ci` |
| `applications/rag-service/**` | `rag-service-ci` |
| `applications/graph-db/**` | `graph-db-ci` |
| `applications/vllm/**` | `vllm-ci` |
| `README.md` | No application CI |
| Multiple application folders | All matching CI pipelines |

### 4. Tekton Trigger Configuration

Tekton Triggers uses GitHub and CEL interceptors to filter incoming webhook events.

**Example: Model Gateway CI Trigger**

```yaml
triggers:
  - name: model-gateway-trigger
    interceptors:
      - ref:
          name: github
        params:
          - name: secretRef
            value:
              secretName: github-secret
              secretKey: secretToken
          - name: eventTypes
            value: ["push"]
          - name: addChangedFiles
            value:
              enabled: true
              # Private repos require a GitHub token.
              # personalAccessToken:
              #   secretName: github-pat
              #   secretKey: token

      - ref:
          name: cel
        params:
          - name: filter
            value: >-
              body.ref == 'refs/heads/main' &&
              extensions.changed_files.matches(
                '(^|,)applications/model-gateway/'
              )

    bindings:
      - ref: model-gateway-binding
    template:
      ref: model-gateway-template
```

This is a trigger entry to place under `spec.triggers` in an `EventListener`, not a complete standalone Kubernetes manifest.

The referenced TriggerBinding and TriggerTemplate must also exist. The TriggerTemplate creates a `PipelineRun` referencing `model-gateway-ci`.

Other applications use equivalent trigger entries with their own paths, bindings, and templates.

### 5. Example Execution

**Scenario A — Single application change**

```bash
git add applications/model-gateway/
git commit -m "Update model gateway authentication"
git push origin main
```

Result:

```text
model-gateway-ci   → TRIGGERED
rag-service-ci     → SKIPPED
graph-db-ci        → SKIPPED
vllm-ci            → SKIPPED
```

**Scenario B — Multiple application changes**

```bash
git add applications/model-gateway/
git add applications/rag-service/
git commit -m "Update gateway and RAG integration"
git push origin main
```

Result:

```text
model-gateway-ci   → TRIGGERED
rag-service-ci     → TRIGGERED
graph-db-ci        → SKIPPED
vllm-ci            → SKIPPED
```

### 6. CI Pipeline Responsibilities

Each application's CI pipeline can perform:

1. Git checkout at the triggering revision.
2. Dependency installation.
3. Linting and static analysis.
4. Unit tests.
5. Container image build.
6. Container image security scan.
7. Push versioned image to the container registry.

Once CI succeeds, the deployment configuration can be updated in the GitOps repository. Argo CD reconciles the desired state with Kubernetes or OpenShift.

### 7. Design Principles

- **Independent CI:** Each application has its own pipeline.
- **Path-based triggering:** Only affected applications are built.
- **Reusable tasks:** Common build, test, and security tasks can be shared.
- **GitOps deployment:** Argo CD manages deployments separately from CI.
- **Shared dependencies:** Changes to shared components must trigger all dependent applications.
- **Secure webhooks:** GitHub signatures must be verified.
- **Branch protection:** Production image publishing occurs only from approved branches after required checks pass.

**Implementation note:** `addChangedFiles` requires the Tekton GitHub interceptor and GitHub API access. Private repositories require an appropriately scoped token. Pull-request validation needs separate event and branch-handling rules. The example above handles pushes to `main` only.

### References

- [Tekton Triggers — Interceptors](https://tekton.dev/docs/triggers/interceptors/)
- [Tekton TriggerBindings](https://tekton.dev/docs/triggers/triggerbindings/)
- [Tekton Pipelines](https://tekton.dev/docs/pipelines/pipelines/)
