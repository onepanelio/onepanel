package template

const VirtualService = `apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: {{.InstanceName}}
spec:
  hosts:
  - {{.InstanceName}}-2.b.onepanel.io
  gateways:
  - {{.SystemIstioGateway}}
  http:
  - match:
    - uri:
        prefix: /ssh
    route:
    - destination:
        port:
          number: 5001
        host: {{.InstanceName}}
  - match:
    - uri:
        prefix: /tensorboard/
    route:
    - destination:
        port:
          number: 6006
        host: {{.InstanceName}}
  - match:
    - uri:
        prefix: /
    route:
    - destination:
        port:
          number: 80
        host: {{.InstanceName}}`
