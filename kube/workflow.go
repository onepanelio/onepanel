package kube

import (
	"encoding/json"
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	argojson "github.com/argoproj/pkg/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type ListOptions = metav1.ListOptions

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
	var jsonOpts []argojson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)
	}
	err = argojson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []Workflow{wf}, nil
	}
	wfs, err = common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return
	}

	return
}

func (c *Client) ValidateWorkflow(manifest []byte) (err error) {
	_, err = unmarshalWorkflows(manifest, true)

	return
}

func (c *Client) CreateWorkflow(namespace string, wfs []*Workflow) (createdWorkflows []*Workflow, err error) {
	var createdWorkflow *Workflow
	for _, wf := range wfs {
		createdWorkflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
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

func (c *Client) TerminateWorkflow(namespace, name string) (err error) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"activeDeadlineSeconds": 0,
		},
	}
	patch, err := json.Marshal(obj)
	if err != nil {
		return
	}
	_, err = c.ArgoprojV1alpha1().Workflows(namespace).Patch(name, types.MergePatchType, patch)
	if err != nil {
		return
	}

	return
}
