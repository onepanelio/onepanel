package workflow

import (
	"os"
	"testing"
)

var workflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: instance-
spec:
  entrypoint: instance-tmpl
  templates:
  - name: instance-tmpl
    steps:
    - - name: instance-service
        template: instance-service-tmpl
    - - name: instance-virtual-service
        template: instance-virtual-service-tmpl
    - - name: instance-statefulset
        template: instance-statefulset-tmpl
  - name: instance-service-tmpl
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
    nodeSelector:
      cloud.google.com/gke-nodepool: default-pool
    resource:
      action: "{{workflow.parameters.action}}"
      manifest: |
        apiVersion: v1
        kind: Service
        metadata:
          name: {{workflow.parameters.instance-name}}
          namespace: {{workflow.parameters.instance-namespace}}
        spec:
          ports:
          - name: http
            port: 80
            protocol: TCP
            targetPort: 80
          selector:
            instanceUID: {{workflow.parameters.instance-name}}
          type: ClusterIP
  - name: instance-virtual-service-tmpl
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
    nodeSelector:
      cloud.google.com/gke-nodepool: default-pool
    resource:
      action: "{{workflow.parameters.action}}"
      manifest: |
        apiVersion: networking.istio.io/v1alpha3
        kind: VirtualService
        metadata:
          name: {{workflow.parameters.instance-name}}
          namespace: {{workflow.parameters.instance-namespace}}
        spec:
          hosts:
          - {{workflow.parameters.instance-name}}.{{workflow.parameters.host}}
          gateways:
          - istio-system/ingressgateway
          http:
          - match:
            - uri:
                prefix: /
            route:
            - destination:
                port:
                  number: 80
                host: {{workflow.parameters.instance-name}}
  - name: instance-statefulset-tmpl
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
    nodeSelector:
      cloud.google.com/gke-nodepool: default-pool
    resource:
      action: "{{workflow.parameters.action}}"
      manifest: |
        apiVersion: apps/v1
        kind: StatefulSet
        metadata:
          name: {{workflow.parameters.instance-name}}
          namespace: {{workflow.parameters.instance-namespace}}
        spec:
          replicas: {{workflow.parameters.instance-replicas}}
          serviceName: {{workflow.parameters.instance-name}}
          selector:
            matchLabels:
              instanceUID: {{workflow.parameters.instance-name}}
          template:
            metadata:
              labels:
                instanceUID: {{workflow.parameters.instance-name}}
            spec:
              nodeSelector:
                cloud.google.com/gke-nodepool: {{workflow.parameters.machine-type}}
              containers:
              - name: main
                image: nginxdemos/hello
                ports:
                - containerPort: 80
                  name: http
                volumeMounts:
                  - name: vol1
                    mountPath: /vol1
          volumeClaimTemplates:
          - metadata:
              name: vol1
            spec:
              accessModes: ["ReadWriteOnce"]
              storageClassName: default
              resources:
                requests:
                  storage: 1Gi
`

func TestUnmarshalWorkflows(t *testing.T) {
	wfs, err := unmarshalWorkflows([]byte(workflow), true)
	if err != nil {
		t.Log(err)
		return
	}

	t.Log(wfs[0])
}

func TestCreate(t *testing.T) {
	c, err := NewClient("default", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Log(err)
		return
	}

	err = c.Create([]byte(workflow), true)
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("success")
}
