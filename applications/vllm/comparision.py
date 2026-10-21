import yaml

with open("/tmp/vllm-rendered.yaml") as f:
    rendered = list(yaml.safe_load_all(f))

with open("/tmp/vllm-live-deployment.yaml") as f:
    live = yaml.safe_load(f)

desired = next(
    item for item in rendered
    if item and item.get("kind") == "Deployment"
)

live_container = live["spec"]["template"]["spec"]["containers"][0]
helm_container = desired["spec"]["template"]["spec"]["containers"][0]

fields = [
    "name",
    "image",
    "imagePullPolicy",
    "command",
    "args",
    "env",
    "ports",
    "resources",
    "volumeMounts",
    "startupProbe",
    "readinessProbe",
    "livenessProbe",
    "securityContext"
]

print("\nLIVE vs HELM CONTAINER COMPARISON\n")

for field in fields:
    a = live_container.get(field)
    b = helm_container.get(field)

    if a == b:
        print(f"[MATCH]     {field}")
    else:
        print(f"[DIFFERENT] {field}")
        print(f"  Live: {a}")
        print(f"  Helm: {b}")
        print()
