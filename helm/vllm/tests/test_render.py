"""Check vLLM chart behavior with Helm and PyYAML, without a Kubernetes cluster.

Run with: python3 -m unittest discover -s helm/vllm/tests -v
Overrides live in temporary files; the chart's values.yaml is never modified.
"""

from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml


CHART = Path(__file__).resolve().parents[1]


class RenderTests(unittest.TestCase):
    def render(self, overrides=None, error=None):
        with tempfile.TemporaryDirectory(prefix="vllm-render-") as temporary:
            values = Path(temporary) / "overrides.yaml"
            values.write_text(yaml.safe_dump(overrides or {}), encoding="utf-8")
            result = subprocess.run(
                ["helm", "template", "render-test", str(CHART), "--values", str(values)],
                capture_output=True,
                text=True,
                check=False,
            )
        if error is not None:
            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn(error, result.stderr)
            return None
        self.assertEqual(result.returncode, 0, result.stderr)
        return [document for document in yaml.safe_load_all(result.stdout) if document]

    def resource(self, documents, kind):
        matches = [document for document in documents if document["kind"] == kind]
        self.assertEqual(len(matches), 1, f"Expected one {kind}, found {len(matches)}")
        return matches[0]

    def workload(self, documents):
        pod = self.resource(documents, "Deployment")["spec"]["template"]["spec"]
        return pod, pod["containers"][0]

    def test_default_inference_workload_and_existing_cache(self):
        documents = self.render()
        deployment = self.resource(documents, "Deployment")
        pod, container = self.workload(documents)
        self.assertEqual(deployment["metadata"]["name"], "vllm")
        self.assertEqual(deployment["spec"]["replicas"], 1)
        self.assertEqual(deployment["spec"]["strategy"], {"type": "Recreate"})
        self.assertEqual(pod["runtimeClassName"], "nvidia")
        self.assertFalse(pod["enableServiceLinks"])
        self.assertEqual(container["image"], "vllm/vllm-openai:v0.10.2")
        self.assertEqual(container["args"][:4], [
            "--model", "Qwen/Qwen2.5-0.5B-Instruct", "--served-model-name", "qwen-small"
        ])
        self.assertEqual(container["args"][-2:], ["--port", "8000"])
        self.assertEqual(container["resources"]["limits"]["nvidia.com/gpu"], 1)
        self.assertEqual(container["startupProbe"]["httpGet"], {"path": "/health", "port": "http"})
        self.assertNotIn("livenessProbe", container)
        self.assertEqual(pod["volumes"], [
            {"name": "model-cache", "persistentVolumeClaim": {"claimName": "vllm-model-cache"}},
            {"name": "dxcore", "hostPath": {"path": "/usr/lib64/libdxcore.so", "type": "File"}},
        ])
        self.assertEqual(container["volumeMounts"], [
            {"name": "model-cache", "mountPath": "/models", "readOnly": False},
            {"name": "dxcore", "mountPath": "/usr/lib/x86_64-linux-gnu/libdxcore.so", "readOnly": True},
        ])
        self.assertFalse(any(document["kind"] == "PersistentVolumeClaim" for document in documents))

    def test_custom_labels_preserve_deployment_and_service_selectors(self):
        documents = self.render({"commonLabels": {"app": "conflicting-name", "team": "platform"}})
        deployment = self.resource(documents, "Deployment")
        service = self.resource(documents, "Service")
        selector = deployment["spec"]["selector"]["matchLabels"]
        self.assertEqual(selector, {"app": "vllm"})
        self.assertEqual(service["spec"]["selector"], selector)
        for labels in [deployment["metadata"]["labels"], deployment["spec"]["template"]["metadata"]["labels"]]:
            self.assertEqual(labels["app"], "vllm")
            self.assertEqual(labels["team"], "platform")
        self.assertEqual(service["spec"]["ports"][0]["targetPort"], 8000)

    def test_mixed_storage_environment_and_extra_containers(self):
        documents = self.render({
            "pvc": {"volumes": [{"volumeName": "cache", "claimName": "shared-cache", "mountPath": "/cache", "readOnly": True}]},
            "configurationVolume": {"enabled": True,
                "configMaps": [{"volumeName": "settings", "configMapName": "server-settings", "mountPath": "/settings", "readOnly": False}],
                "secrets": [{"volumeName": "credentials", "secretName": "test-credentials", "mountPath": "/credentials", "readOnly": True}]},
            "customVolumes": {"enabled": True, "volumes": [{"name": "scratch", "emptyDir": {"medium": "Memory"}}]},
            "customVolumeMountforMainContainer": {"enabled": True, "volumeMounts": [{"name": "scratch", "mountPath": "/scratch", "subPath": "worker"}]},
            "env": [{"name": "MODEL_MODE", "value": "test"}],
            "environmentVariable": {"enabled": True, "configMaps": ["env-first", "env-second"], "secrets": ["test-env-secret"]},
            "image": {"command": ["vllm"], "args": ["serve", "test-model"]},
            "securityContext": {"runAsUser": 1000},
            "additionalContainerPorts": {"enabled": True, "additionalPorts": [{"portName": "metrics", "containerPort": 9090}]},
            "customContainers": {"enabled": True, "containers": [{"name": "sidecar", "image": "busybox:1.36", "args": ["sleep", "60"]}]},
            "initContainers": {"enabled": True, "initContainers": [{"name": "warm-cache", "image": "busybox:1.36", "command": ["true"]}]},
        })
        pod, container = self.workload(documents)
        self.assertEqual(pod["volumes"], [
            {"name": "cache", "persistentVolumeClaim": {"claimName": "shared-cache"}},
            {"name": "settings", "configMap": {"name": "server-settings"}},
            {"name": "credentials", "secret": {"secretName": "test-credentials"}},
            {"name": "scratch", "emptyDir": {"medium": "Memory"}},
        ])
        self.assertEqual(container["volumeMounts"], [
            {"name": "cache", "mountPath": "/cache", "readOnly": True},
            {"name": "settings", "mountPath": "/settings", "readOnly": False},
            {"name": "credentials", "mountPath": "/credentials", "readOnly": True},
            {"name": "scratch", "mountPath": "/scratch", "subPath": "worker"},
        ])
        self.assertEqual(container["envFrom"], [
            {"configMapRef": {"name": "env-first"}},
            {"configMapRef": {"name": "env-second"}},
            {"secretRef": {"name": "test-env-secret"}},
        ])
        self.assertEqual(container["env"], [{"name": "MODEL_MODE", "value": "test"}])
        self.assertEqual(container["command"], ["vllm"])
        self.assertEqual(container["args"], ["serve", "test-model"])
        self.assertEqual(container["securityContext"], {"runAsUser": 1000})
        self.assertEqual(container["ports"][1], {"name": "metrics", "containerPort": 9090, "protocol": "TCP"})
        self.assertEqual([item["name"] for item in pod["containers"]], ["vllm", "sidecar"])
        self.assertEqual(pod["initContainers"][0]["name"], "warm-cache")

    def test_enabled_empty_optional_lists_are_omitted(self):
        documents = self.render({
            "pvc": {"volumes": []},
            "configurationVolume": {"enabled": True, "configMaps": [], "secrets": []},
            "customVolumes": {"enabled": True, "volumes": []},
            "customVolumeMountforMainContainer": {"enabled": True, "volumeMounts": []},
            "environmentVariable": {"enabled": True, "configMaps": [], "secrets": []},
            "customContainers": {"enabled": True, "containers": []},
            "initContainers": {"enabled": True, "initContainers": []},
        })
        pod, container = self.workload(documents)
        self.assertNotIn("volumes", pod)
        self.assertNotIn("initContainers", pod)
        self.assertNotIn("volumeMounts", container)
        self.assertNotIn("envFrom", container)
        self.assertEqual(len(pod["containers"]), 1)

    def test_image_source_precedence(self):
        cases = [
            ({"ImageFullurl": "complete.example/server:v1", "ImagePath": "server:v2", "registryDomain": "local.example"}, "global.example", "complete.example/server:v1"),
            ({"ImageFullurl": "", "ImagePath": "server:v2", "registryDomain": "local.example"}, "global.example", "global.example/server:v2"),
            ({"ImageFullurl": "", "ImagePath": "server:v2", "registryDomain": "local.example"}, "", "local.example/server:v2"),
            ({"ImageFullurl": "", "ImagePath": "server:v2"}, "", "server:v2"),
        ]
        for image, global_registry, expected in cases:
            with self.subTest(image=expected):
                documents = self.render({"image": image, "global": {"enterpriseCharts": {"image": {"registryDomain": global_registry}}}})
                _, container = self.workload(documents)
                self.assertEqual(container["image"], expected)

    def test_registry_without_image_path_fails(self):
        cases = [
            {"image": {"ImageFullurl": "", "registryDomain": "local.example"}},
            {"image": {"ImageFullurl": ""}, "global": {"enterpriseCharts": {"image": {"registryDomain": "global.example"}}}},
        ]
        for overrides in cases:
            with self.subTest(overrides=overrides):
                self.render(overrides, error="ImagePath")

    def test_autoscaling_cpu_memory_and_combined_metrics(self):
        for metrics in [
            {"targetCPUUtilizationPercentage": 70},
            {"targetMemoryUtilizationPercentage": 80},
            {"targetCPUUtilizationPercentage": 70, "targetMemoryUtilizationPercentage": 80},
        ]:
            with self.subTest(metrics=metrics):
                documents = self.render({"autoscaling": {"enabled": True, "minReplicas": 2, "maxReplicas": 4, **metrics}})
                hpa = self.resource(documents, "HorizontalPodAutoscaler")["spec"]
                self.assertEqual(hpa["minReplicas"], 2)
                self.assertEqual(hpa["maxReplicas"], 4)
                self.assertEqual(hpa["scaleTargetRef"], {"apiVersion": "apps/v1", "kind": "Deployment", "name": "vllm"})
                expected = {name: metrics[key] for name, key in [
                    ("cpu", "targetCPUUtilizationPercentage"), ("memory", "targetMemoryUtilizationPercentage")
                ] if key in metrics}
                actual = {metric["resource"]["name"]: metric["resource"]["target"]["averageUtilization"] for metric in hpa["metrics"]}
                self.assertEqual(actual, expected)
                self.assertNotIn("replicas", self.resource(documents, "Deployment")["spec"])

    def test_invalid_autoscaling_configuration_fails(self):
        for settings in [
            {"minReplicas": 4, "maxReplicas": 2, "targetCPUUtilizationPercentage": 70},
            {"minReplicas": 0, "maxReplicas": 2, "targetCPUUtilizationPercentage": 70},
            {"minReplicas": 1.5, "maxReplicas": 2, "targetCPUUtilizationPercentage": 70},
            {"maxReplicas": 2, "targetCPUUtilizationPercentage": 70.5},
            {"maxReplicas": 2, "targetCPUUtilizationPercentage": -1},
            {"maxReplicas": 2, "targetMemoryUtilizationPercentage": -1},
            {"maxReplicas": 2},
        ]:
            with self.subTest(settings=settings):
                self.render({"autoscaling": {"enabled": True, **settings}}, error="autoscaling")

    def test_disabled_deployment_has_no_autoscaler(self):
        documents = self.render({"deployToCluster": False, "autoscaling": {"enabled": True}})
        self.assertFalse(any(document["kind"] in {"Deployment", "HorizontalPodAutoscaler"} for document in documents))
        self.resource(documents, "Service")

    def test_service_account_selection(self):
        for create, name, expected in [
            (False, "", "default"), (False, "existing-account", "existing-account"),
            (True, "", "vllm"), (True, "inference-account", "inference-account"),
        ]:
            with self.subTest(create=create, name=name):
                documents = self.render({"serviceAccount": {"create": create, "name": name}})
                pod, _ = self.workload(documents)
                self.assertEqual(pod["serviceAccountName"], expected)
                accounts = [document for document in documents if document["kind"] == "ServiceAccount"]
                self.assertEqual([account["metadata"]["name"] for account in accounts], [expected] if create else [])


if __name__ == "__main__":
    unittest.main()
