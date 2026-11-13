# Tekton Tasks

Reusable Tekton task definitions for steps such as building images, running tests, scanning artifacts, and publishing results.


# Tasks first
oc apply -f pipelines/tekton/tasks/ -n ai-platform-ci

# Pipelines next
oc apply -f pipelines/tekton/pipelines/ -n ai-platform-ci

# Triggers last
oc apply -f pipelines/tekton/triggers/ -n ai-platform-ci