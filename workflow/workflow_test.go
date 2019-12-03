package workflow

import (
	"flag"
	"os"
	"testing"
)

var instanceWorkflowTemplate = `
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
          name: {{workflow.parameters.name}}
        spec:
          ports:
          - name: http
            port: 80
            protocol: TCP
            targetPort: 80
          selector:
            instanceUID: {{workflow.parameters.name}}
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
          name: {{workflow.parameters.name}}
        spec:
          hosts:
          - {{workflow.parameters.name}}.{{workflow.parameters.host}}
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
                host: {{workflow.parameters.name}}
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
          name: {{workflow.parameters.name}}
        spec:
          replicas: {{workflow.parameters.replicas}}
          serviceName: {{workflow.parameters.name}}
          selector:
            matchLabels:
              instanceUID: {{workflow.parameters.name}}
          template:
            metadata:
              labels:
                instanceUID: {{workflow.parameters.name}}
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

var namespace = flag.String("namespace", "default", "namespace of workflows")

func TestUnmarshalWorkflows(t *testing.T) {
	wfs, err := unmarshalWorkflows([]byte(instanceWorkflowTemplate), true)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wfs[0])
}

func TestCreateInstance(t *testing.T) {
	c, err := NewClient(*namespace, os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Error(err)
		return
	}

	wf, err := c.Create(instanceWorkflowTemplate, []string{"name=http-test-1", "action=create", "replicas=1", "machine-type=default-pool", "host=test-cluster-11.onepanel.io"})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf)
}

func TestPauseInstance(t *testing.T) {
	c, err := NewClient(*namespace, os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Error(err)
		return
	}

	wf, err := c.Create(instanceWorkflowTemplate, []string{"name=http-test-1", "action=apply", "replicas=0", "machine-type=default-pool", "host=test-cluster-11.onepanel.io"})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf)
}

func TestResumeInstance(t *testing.T) {
	c, err := NewClient(*namespace, os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Error(err)
		return
	}

	wf, err := c.Create(instanceWorkflowTemplate, []string{"name=http-test-1", "action=apply", "replicas=1", "machine-type=default-pool", "host=test-cluster-11.onepanel.io"})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf)
}

func TestChangeInstanceMachineType(t *testing.T) {
	c, err := NewClient(*namespace, os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Error(err)
		return
	}

	wf, err := c.Create(instanceWorkflowTemplate, []string{"name=http-test-1", "action=apply", "replicas=1", "machine-type=cpu-1-4", "host=test-cluster-11.onepanel.io"})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf)
}

func TestDeleteInstance(t *testing.T) {
	c, err := NewClient(*namespace, os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Error(err)
		return
	}

	wf, err := c.Create(instanceWorkflowTemplate, []string{"name=http-test-1", "action=delete", "replicas=1", "machine-type=default-pool", "host=test-cluster-11.onepanel.io"})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf)
}
