apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: resources-reader
  namespace: {{.Namespace}}
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["list", "get"]
  - apiGroups: ["kustomize.toolkit.fluxcd.io"]
    resources: ["kustomizations"]
    verbs: ["*"]
  - apiGroups: ["helm.toolkit.fluxcd.io"]
    resources: ["helmreleases"]
    verbs: ["*"]
  - apiGroups: ["source.toolkit.fluxcd.io"]
    resources: ["buckets", "helmcharts", "gitrepositories", "helmrepositories"]
    verbs: ["*"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["*"]
