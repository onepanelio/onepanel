package kube

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/pkg/json"
	"github.com/onepanelio/core/util/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type Workflow = wfv1.Workflow

type WorkflowParameter = wfv1.Parameter
type PodGCStrategy = wfv1.PodGCStrategy

type WorkflowOptions struct {
	Name           string
	GeneratedName  string
	Entrypoint     string
	Parameters     []WorkflowParameter
	ServiceAccount string
	Labels         *map[string]string
	ListOptions    *ListOptions
	PodGCStrategy  *PodGCStrategy
}

func unmarshalWorkflows(wfBytes []byte, strict bool) (wfs []Workflow, err error) {
	var wf Workflow
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err = json.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []Workflow{wf}, nil
	}
	wfs, err = common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return
	}

	return
}

func (c *Client) create(namespace string, wf *Workflow, opts *WorkflowOptions) (createdWorkflow *Workflow, err error) {

	if opts == nil {
		opts = &WorkflowOptions{}
	}

	if opts.Name != "" {
		wf.ObjectMeta.Name = opts.Name
	}
	if opts.GeneratedName != "" {
		wf.ObjectMeta.GenerateName = opts.GeneratedName
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	if len(opts.Parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, param := range opts.Parameters {
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}

		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	if opts.Labels != nil {
		wf.ObjectMeta.Labels = *opts.Labels
	}

	//TODO - Load this data from onepanel config-map or secret
	podGCStrategy := env.GetEnv("ARGO_POD_GC_STRATEGY", "OnPodCompletion")
	if podGCStrategy != "" {
		strategy := PodGCStrategy(podGCStrategy)
		opts.PodGCStrategy = &strategy
	} else {
		strategy := PodGCStrategy("OnPodCompletion")
		opts.PodGCStrategy = &strategy
	}

	if wf.Spec.PodGC == nil {
		wf.Spec.PodGC = &wfv1.PodGC{
			Strategy: *opts.PodGCStrategy,
		}
	}

	createdWorkflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) ValidateWorkflow(manifest []byte) (err error) {
	_, err = unmarshalWorkflows(manifest, true)

	return
}

func (c *Client) CreateWorkflow(namespace string, manifest []byte, opts *WorkflowOptions) (createdWorkflows []*Workflow, err error) {
	workflows, err := unmarshalWorkflows(manifest, true)
	if err != nil {
		return nil, err
	}

	for _, wf := range workflows {
		createdWorkflow, err := c.create(namespace, &wf, opts)
		if err != nil {
			return nil, err
		}
		createdWorkflows = append(createdWorkflows, createdWorkflow)
	}

	return
}

func (c *Client) GetWorkflow(namespace, name string) (workflow *Workflow, err error) {
	workflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Get(name, v1.GetOptions{})

	return
}

func (c *Client) ListWorkflows(namespace string, opts *WorkflowOptions) (workflows []*Workflow, err error) {
	if opts.ListOptions == nil {
		opts.ListOptions = &ListOptions{}
	}
	workflowList, err := c.ArgoprojV1alpha1().Workflows(namespace).List(*opts.ListOptions)
	if err != nil {
		return
	}

	for i := range workflowList.Items {
		workflows = append(workflows, &(workflowList.Items[i]))
	}

	return
}

func (c *Client) WatchWorkflow(namespace, name string) (watcher watch.Interface, err error) {
	fieldSelector, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	if err != nil {
		return
	}
	watcher, err = c.ArgoprojV1alpha1().Workflows(namespace).Watch(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})

	return
}
