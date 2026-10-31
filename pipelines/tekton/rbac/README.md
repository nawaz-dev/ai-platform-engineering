PS C:\Users\nawaz\OneDrive\Desktop\DesktopFiles\projects\ai-infra-setup> oc create rolebinding fde-eventlistener-standard `
>>   --clusterrole=tekton-triggers-eventlistener-roles `
>>   --serviceaccount=ai-platform-ci:fde-eventlistener-sa `
>>   -n ai-platform-ci
rolebinding.rbac.authorization.k8s.io/fde-eventlistener-standard created
PS C:\Users\nawaz\OneDrive\Desktop\DesktopFiles\projects\ai-infra-setup> oc create clusterrolebinding fde-eventlistener-clusterbinding `
>>   --clusterrole=tekton-triggers-eventlistener-clusterroles `
>>   --serviceaccount=ai-platform-ci:fde-eventlistener-sa
clusterrolebinding.rbac.authorization.k8s.io/fde-eventlistener-clusterbinding created
PS C:\Users\nawaz\OneDrive\Desktop\DesktopFiles\projects\ai-infra-setup> oc create rolebinding fde-eventlistener-standard `
>>   --clusterrole=tekton-triggers-eventlistener-roles `
>>   --serviceaccount=ai-platform-ci:fde-eventlistener-sa `
>>   -n ai-platform-ci