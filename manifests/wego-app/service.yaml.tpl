apiVersion: v1
kind: Service
metadata:
  name: wego-app
  namespace: {{.Namespace}}
spec:
  selector:
    app: wego-app
  ports:
    - protocol: TCP
      port: 9001
      targetPort: 9001
