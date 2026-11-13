# Pipelines

Contains CI/CD pipeline definitions that build, test, scan, package, and promote platform components.

# Apply everything
```oc apply -R -f pipelines/tekton/ -n ai-platform-ci```


-R recursively reads YAML files from all subdirectories.

# Tasks first
```oc apply -f pipelines/tekton/tasks/ -n ai-platform-ci```

# Pipelines next
```oc apply -f pipelines/tekton/pipelines/ -n ai-platform-ci```

# Triggers last
```oc apply -f pipelines/tekton/triggers/ -n ai-platform-ci```