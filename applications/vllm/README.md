# Applications — GPU-Accelerated vLLM on k3d (WSL2)

This guide documents a working **vLLM v0.10.2** deployment serving **`Qwen/Qwen2.5-0.5B-Instruct`** through an OpenAI-compatible API on a dedicated GPU-enabled **k3d/K3s** cluster. It also records the actual failures encountered and their fixes.

> **Scope:** Development lab running Docker Desktop + WSL2 Ubuntu, NVIDIA RTX 4050 Laptop GPU (~6 GiB VRAM), and k3d cluster `fde-gpu-v2`. Do **not** apply WSL2-specific NVIDIA workarounds to production OpenShift clusters without validation. Keep existing `dev-cluster` and OpenShift Local CRC untouched.

## 1. Architecture

```text
WSL2 Ubuntu / Docker Desktop / NVIDIA GPU
  └─ k3d cluster: fde-gpu-v2
      └─ K3s agent: k3d-fde-gpu-v2-agent-0 (nvidia.com/gpu: 1)
          └─ namespace: llm-serving
              ├─ Deployment/vllm (RuntimeClass: nvidia)
              │   └─ vllm/vllm-openai:v0.10.2
              │       └─ Qwen/Qwen2.5-0.5B-Instruct
              ├─ PVC/vllm-model-cache (15Gi; HF_HOME=/models/huggingface)
              └─ Service/vllm (ClusterIP; TCP 8000)
                  └─ kubectl port-forward → localhost:8000
```

The GPU device plugin advertises `nvidia.com/gpu`; Kubernetes schedules the pod using that extended resource. The NVIDIA runtime enables GPU access in the container. **GPU memory is not shared automatically across pods.**

## 2. Prerequisites and checks

```bash
kubectl config get-contexts
kubectl config use-context k3d-fde-gpu-v2
kubectl get nodes -o wide
kubectl get runtimeclass nvidia
kubectl get node k3d-fde-gpu-v2-agent-0 \
  -o jsonpath='{.status.allocatable.nvidia\.com/gpu}{"\n"}'
nvidia-smi
```

Expected: dedicated cluster context, `RuntimeClass/nvidia`, node GPU allocatable count `1`, and visible NVIDIA GPU on the WSL host. If GPU count is zero, check the NVIDIA device plugin and container runtime before deploying vLLM:

```bash
kubectl get daemonsets -A | grep -i nvidia
kubectl describe node k3d-fde-gpu-v2-agent-0
kubectl get pods -A | grep -i nvidia
```

### WSL2-specific GPU runtime notes

This lab uses a custom K3s image (`fde-k3s-nvidia:v1.31.5-persistent`) containing NVIDIA Container Toolkit binaries and working `ldconfig` binaries. The GPU-enabled cluster was created separately to avoid affecting existing clusters. A verification cluster can be created with:

```bash
k3d cluster create fde-gpu-verify \
  --image fde-k3s-nvidia:v1.31.5-persistent \
  --servers 1 --agents 1 --gpus all --wait
```

The NVIDIA device plugin, `RuntimeClass/nvidia`, and WSL2 `libdxcore.so` access must be configured and tested before deploying the application. On this lab, the pod mounts:

```yaml
volumeMounts:
  - name: dxcore
    mountPath: /usr/lib/x86_64-linux-gnu/libdxcore.so
    readOnly: true
volumes:
  - name: dxcore
    hostPath:
      path: /usr/lib64/libdxcore.so
      type: File
```

The source path must actually exist **inside the k3d node container**, not just on the WSL host. This mount is **specific to the validated WSL2 lab**, not a general-purpose Kubernetes practice. The NVIDIA device plugin may require the same WSL2 adjustment. Do not introduce a duplicate NVIDIA runtime in K3s `config.toml.tmpl`; K3s runtime autodetection is used here.

## 3. Repository layout

```text
applications/
├── README.md
└── vllm/
    ├── pvc.yaml
    ├── deployment.yaml
    └── service.yaml
```

From the repository root:

```bash
mkdir -p applications/vllm
kubectl create namespace llm-serving --dry-run=client -o yaml | kubectl apply -f -
```

## 4. Kubernetes manifests

### `applications/vllm/pvc.yaml`

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: vllm-model-cache
  namespace: llm-serving
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 15Gi
```

### `applications/vllm/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vllm
  namespace: llm-serving
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: vllm
  template:
    metadata:
      labels:
        app: vllm
    spec:
      # Critical: prevent Kubernetes from injecting VLLM_PORT=tcp://... .
      enableServiceLinks: false
      runtimeClassName: nvidia
      # Lab-specific; replace with GPU node labels/affinity in production.
      nodeSelector:
        kubernetes.io/hostname: k3d-fde-gpu-v2-agent-0
      containers:
        - name: vllm
          image: vllm/vllm-openai:v0.10.2
          imagePullPolicy: IfNotPresent
          args:
            - --model
            - Qwen/Qwen2.5-0.5B-Instruct
            - --served-model-name
            - qwen-small
            - --dtype
            - half
            - --max-model-len
            - "1024"
            - --gpu-memory-utilization
            - "0.60"
            - --max-num-seqs
            - "2"
            - --host
            - 0.0.0.0
            - --port
            - "8000"
          env:
            - name: HF_HOME
              value: /models/huggingface
            - name: NVIDIA_DRIVER_CAPABILITIES
              value: compute,utility
          ports:
            - name: http
              containerPort: 8000
          resources:
            requests:
              cpu: "1"
              memory: 3Gi
              nvidia.com/gpu: 1
            limits:
              cpu: "4"
              memory: 8Gi
              nvidia.com/gpu: 1
          volumeMounts:
            - name: model-cache
              mountPath: /models
            - name: dxcore
              mountPath: /usr/lib/x86_64-linux-gnu/libdxcore.so
              readOnly: true
          startupProbe:
            httpGet:
              path: /health
              port: http
            periodSeconds: 10
            failureThreshold: 90
          readinessProbe:
            httpGet:
              path: /health
              port: http
            periodSeconds: 10
            failureThreshold: 3
      volumes:
        - name: model-cache
          persistentVolumeClaim:
            claimName: vllm-model-cache
        - name: dxcore
          hostPath:
            path: /usr/lib64/libdxcore.so
            type: File
```

> The `8Gi` container memory limit refers to **system RAM**, not GPU VRAM. The `--gpu-memory-utilization 0.60` setting controls vLLM's target fraction of GPU memory and does not guarantee that every workload will fit. `Recreate` avoids overlapping old and new replicas competing for the single GPU.

### `applications/vllm/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: vllm
  namespace: llm-serving
spec:
  type: ClusterIP
  selector:
    app: vllm
  ports:
    - name: http
      port: 8000
      targetPort: http
      protocol: TCP
```

## 5. Deploy and monitor

```bash
kubectl config use-context k3d-fde-gpu-v2
kubectl apply -f applications/vllm/
kubectl -n llm-serving get pods,pvc,svc -o wide
kubectl -n llm-serving rollout status deployment/vllm --timeout=900s
kubectl -n llm-serving logs deployment/vllm --tail=150 --timestamps
```

The initial run may download both a large vLLM container image and Hugging Face model weights. The PVC retains the model cache across pod restarts. If the pod restarts, inspect the **previous** container's logs:

```bash
kubectl -n llm-serving get pods
kubectl -n llm-serving logs deployment/vllm --previous --tail=150
kubectl -n llm-serving get events --sort-by=.lastTimestamp | tail -30
```

## 6. Test the OpenAI-compatible API

In terminal 1:

```bash
kubectl -n llm-serving port-forward svc/vllm 8000:8000
```

In terminal 2:

```bash
curl -fsS http://localhost:8000/health
curl -sS http://localhost:8000/v1/models | python3 -m json.tool
curl -sS http://localhost:8000/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "qwen-small",
    "messages": [
      {"role": "user", "content": "Explain how Kubernetes schedules pods requesting NVIDIA GPUs."}
    ],
    "max_tokens": 300,
    "temperature": 0.2
  }' | python3 -m json.tool
```

**Verified result in this lab:** The API returned `object: chat.completion`, `model: qwen-small`, and token usage (`37` prompt + `100` completion = `137` total tokens) for a test request. `finish_reason: length` occurred because the test used `max_tokens: 100`; increase that limit for longer responses. A successful API call proves inference works, **not** that the model's factual answer is necessarily correct.

Optional metrics inspection:

```bash
curl -sS http://localhost:8000/metrics | grep '^vllm:' | head -30
nvidia-smi
```

## 7. Troubleshooting: actual incidents

### Incident A — `ImagePullBackOff`

**Observed:** `0/1 ImagePullBackOff`; PVC was `Bound`, Service existed.

**Meaning:** The kubelet could not pull the vLLM container image. The precise underlying registry error was not preserved, so do not assume it was an invalid tag or an authentication failure.

**Diagnose:**

```bash
kubectl -n llm-serving describe pod <POD_NAME> | tail -40
kubectl -n llm-serving get events --sort-by=.lastTimestamp | tail -30
kubectl -n llm-serving get deployment vllm \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
```

**Possible remediation if Docker can pull the image but k3d cannot:**

```bash
docker pull vllm/vllm-openai:v0.10.2
k3d image import vllm/vllm-openai:v0.10.2 -c fde-gpu-v2
kubectl -n llm-serving rollout restart deployment/vllm
```

**Verified later:** Pod events reported `Container image "vllm/vllm-openai:v0.10.2" already present on machine` and successful `Created`/`Started` events. This confirms the image problem was no longer blocking startup; it does not establish which pull remediation caused the recovery.

### Incident B — `Startup probe failed: connection refused`

**Observed:**

```text
Startup probe failed: Get "http://10.42.0.19:8000/health":
dial tcp 10.42.0.19:8000: connect: connection refused
```

**Meaning:** The HTTP listener was not ready. This is a **symptom**, not a root cause: it can occur during normal model startup or when the engine has crashed.

**Diagnose:**

```bash
kubectl -n llm-serving get pods -o wide
kubectl -n llm-serving logs deployment/vllm --tail=150
kubectl -n llm-serving logs deployment/vllm --previous --tail=150
```

Do not change probe timing before inspecting engine logs. The configured startup probe allows roughly 15 minutes for initialization, but a crashing process will not recover by waiting longer.

### Incident C — vLLM `VLLM_PORT` Kubernetes Service collision (confirmed root cause)

**Observed root-cause log:**

```text
ValueError: VLLM_PORT 'tcp://10.43.35.210:8000' appears to be a URI.
This may be caused by a Kubernetes service discovery issue
RuntimeError: Engine core initialization failed.
```

**Cause:** A Service named `vllm` caused Kubernetes to inject `VLLM_PORT=tcp://<ClusterIP>:8000` into the container. vLLM also uses `VLLM_PORT` internally and expects a numeric port value. The variable collision prevented EngineCore startup. The explicit `--port 8000` CLI flag controls the API listener and does not prevent this internal environment-variable collision.

**Permanent fix:** Add the following under `Deployment.spec.template.spec`:

```yaml
enableServiceLinks: false
```

**Apply the version-controlled fix:**

```bash
kubectl apply -f applications/vllm/deployment.yaml
kubectl -n llm-serving rollout status deployment/vllm --timeout=900s
kubectl -n llm-serving logs deployment/vllm --tail=100
```

**Temporary diagnostic patch only** (also update the YAML, or GitOps will revert the change):

```bash
kubectl -n llm-serving patch deployment vllm --type=merge \
  -p '{"spec":{"template":{"spec":{"enableServiceLinks":false}}}}'
```

**Why this works:** Disables Kubernetes' legacy Service environment-variable injection while retaining DNS-based Service discovery. After this fix, the user successfully received a `chat.completion` response from `qwen-small`.

### Incident D — GPU missing or CUDA startup failure (lab setup checklist)

This is a **preventive checklist**, not the observed root cause of the final vLLM crash.

```bash
nvidia-smi
kubectl get runtimeclass nvidia
kubectl describe node k3d-fde-gpu-v2-agent-0 | grep -A8 -B4 nvidia.com/gpu
kubectl get pods -A | grep -i nvidia
kubectl -n llm-serving logs deployment/vllm --tail=150 \
  | grep -Ei 'cuda|nvml|gpu|dxcore|error'
```

Verify NVIDIA Container Toolkit, the device plugin, `RuntimeClass`, GPU resource request/limit, and WSL2 `libdxcore.so` mount. A previous lab issue required copying the real `ldconfig` binaries into the custom K3s image to fix NVIDIA runtime hook initialization. Avoid applying these host-specific fixes to unrelated clusters.

### Incident E — `CUDA out of memory` (potential issue)

Not observed in the shared successful run. If encountered, inspect `nvidia-smi` and vLLM logs, stop competing GPU workloads, reduce `--max-model-len`/`--max-num-seqs`, or choose a smaller model. Lowering `--gpu-memory-utilization` alone does not always fix memory pressure; it can also reduce available KV-cache capacity.

### Incident F — Model download/cache issues (potential issue)

Check whether Hugging Face is reachable and whether model files were cached:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' \
  https://huggingface.co/api/models/Qwen/Qwen2.5-0.5B-Instruct
kubectl -n llm-serving exec deployment/vllm -- \
  ls -lah /models/huggingface/hub/
```

Expected cache directory name includes `models--Qwen--Qwen2.5-0.5B-Instruct`. A directory alone does not prove that all weight files were downloaded. Locally stored LM Studio `.gguf` models are not automatically reusable as the Hugging Face-format model loaded by this vLLM configuration.

## 8. Quick operational commands

```bash
# View rollout, pods, storage and service
kubectl -n llm-serving get deployment,pods,pvc,svc -o wide

# Live logs
kubectl -n llm-serving logs -f deployment/vllm

# Restart after configuration change
kubectl -n llm-serving rollout restart deployment/vllm

# See GPU capacity and allocated requests
kubectl describe node k3d-fde-gpu-v2-agent-0

# Port-forward for local testing
kubectl -n llm-serving port-forward svc/vllm 8000:8000

# View API metrics (with port-forward active)
curl -sS http://localhost:8000/metrics | head -40
```

## 9. Security, portability and next steps

This is a **functional development deployment**, not yet a hardened production serving platform:

- **Security:** Keep the ClusterIP service private; do not expose an unauthenticated vLLM API publicly. Add an authenticated gateway, per-user quotas, TLS, NetworkPolicies, and audit logs.
- **Scheduling:** Replace the hardcoded node hostname with stable GPU node labels/affinity in multi-node clusters.
- **Storage:** `local-path` RWO storage is node-local. Use a suitable persistent storage class for production; do not assume the model cache can move between nodes.
- **Supply chain:** Pin images by digest, validate model revision/checksums, and mirror images/models to internal registries for disconnected environments.
- **Reliability:** Add monitoring, alerts, realistic load tests, capacity planning, and appropriate resource/probe tuning.
- **CI/CD:** Convert manifests into a Helm chart; use **Tekton** to lint/test/scan and publish artifacts, and **Argo CD** to reconcile the GitOps repository. Avoid having Tekton and Argo CD independently manage the same Deployment.
- **Performance:** Measure time to first token (TTFT), output tokens/sec, throughput, request latency, GPU utilization, and KV-cache behavior.

### Outcome

**Validated:** vLLM successfully served `Qwen/Qwen2.5-0.5B-Instruct` as `qwen-small` through `/v1/chat/completions` after resolving the Kubernetes `VLLM_PORT` collision.
