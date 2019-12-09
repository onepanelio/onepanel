package argo

import (
	"flag"
	"os"
	"testing"

	"github.com/onepanelio/core/util/ptr"
)

var TestInstanceWorkflowManifest = []byte(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: vscode-
spec:
  podGC:
    strategy: OnWorkflowCompletion
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
            targetPort: 8080
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
                image: codercom/code-server:v2
                args: ["--auth", "none"]
                ports:
                - containerPort: 80
                  name: http
                volumeMounts:
                  - name: vol1
                    mountPath: /home/coder
          volumeClaimTemplates:
          - metadata:
              name: vol1
            spec:
              accessModes: ["ReadWriteOnce"]
              storageClassName: default
              resources:
                requests:
                  storage: 1Gi
`)

var (
	namespace = flag.String("namespace", "default", "namespace of workflows")
	options   = &Options{
		Parameters: []Parameter{
			{
				Name:  "name",
				Value: ptr.String("vscode"),
			},
			{
				Name:  "machine-type",
				Value: ptr.String("default-pool"),
			},
			{
				Name:  "replicas",
				Value: ptr.String("1"),
			},
			{
				Name:  "host",
				Value: ptr.String("test-cluster-11.onepanel.io"),
			},
		},
	}
)

func TestUnmarshalWorkflows(t *testing.T) {
	wfs, err := unmarshalWorkflows([]byte(TestInstanceWorkflowManifest), true)
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

	options.Parameters = append(options.Parameters, Parameter{
		Name:  "action",
		Value: ptr.String("create"),
	})

	wf, err := c.Create(TestInstanceWorkflowManifest, options)
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

	options.Parameters = append(options.Parameters, Parameter{
		Name:  "action",
		Value: ptr.String("apply"),
	}, Parameter{
		Name:  "replicas",
		Value: ptr.String("0"),
	})

	wf, err := c.Create(TestInstanceWorkflowManifest, options)
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

	options.Parameters = append(options.Parameters, Parameter{
		Name:  "action",
		Value: ptr.String("apply"),
	}, Parameter{
		Name:  "replicas",
		Value: ptr.String("1"),
	})

	wf, err := c.Create(TestInstanceWorkflowManifest, options)
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

	options.Parameters = append(options.Parameters, Parameter{
		Name:  "action",
		Value: ptr.String("apply"),
	}, Parameter{
		Name:  "machine-type",
		Value: ptr.String("cpu-1-4"),
	})

	wf, err := c.Create(TestInstanceWorkflowManifest, options)
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

	options.Parameters = append(options.Parameters, Parameter{
		Name:  "action",
		Value: ptr.String("delete"),
	})

	wf, err := c.Create(TestInstanceWorkflowManifest, options)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(wf[0].Name)
}

/**** Some other test scenarios
- System-wide environment variables
- System-wide parameters like `host`
- Startup script that can be executed in:
  - Init Container: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
  - LifeCycle Hooks: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks
****/
