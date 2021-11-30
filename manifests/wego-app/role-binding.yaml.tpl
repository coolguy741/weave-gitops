apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-resources
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: wego-app-service-account
    namespace: wego-system
roleRef:
  kind: Role
  name: resources-reader
  apiGroup: rbac.authorization.k8s.io
