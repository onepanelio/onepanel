package kube

import (
	"encoding/json"
	"errors"
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	argoutil "github.com/argoproj/argo/workflow/util"
	argojson "github.com/argoproj/pkg/json"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util/env"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type ListOptions = metav1.ListOptions

type Workflow = wfv1.Workflow

type WorkflowParameter = wfv1.Parameter

type PodGCStrategy = wfv1.PodGCStrategy

type WorkflowOptions struct {
	Name           string
	GenerateName   string
	Entrypoint     string
	Parameters     []WorkflowParameter
	ServiceAccount string
	Labels         *map[string]string
	ListOptions    *ListOptions
	PodGCStrategy  *PodGCStrategy
}

func modelWorkflow(wf *wfv1.Workflow) (workflow *model.Workflow) {
	manifest, err := json.Marshal(wf)
	if err != nil {
		return
	}
	workflow = &model.Workflow{
		UID:       string(wf.UID),
		CreatedAt: wf.CreationTimestamp.UTC(),
		Name:      wf.Name,
		Manifest:  string(manifest),
	}

	return
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

func (c *Client) create(namespace string, wf *Workflow, opts *WorkflowOptions) (createdWorkflow *Workflow, err error) {

	if opts == nil {
		opts = &WorkflowOptions{}
	}

	if opts.Name != "" {
		wf.ObjectMeta.Name = opts.Name
	}
	if opts.GenerateName != "" {
		wf.ObjectMeta.GenerateName = opts.GenerateName
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

	if opts.PodGCStrategy == nil {
		if wf.Spec.PodGC == nil {
			//TODO - Load this data from onepanel config-map or secret
			podGCStrategy := env.GetEnv("ARGO_POD_GC_STRATEGY", "OnPodCompletion")
			strategy := PodGCStrategy(podGCStrategy)
			wf.Spec.PodGC = &wfv1.PodGC{
				Strategy: strategy,
			}
		}
	} else {
		wf.Spec.PodGC = &wfv1.PodGC{
			Strategy: *opts.PodGCStrategy,
		}
	}

	addSecretValsToTemplate := true
	secret, err := c.GetSecret(namespace, "onepanel-default-env")
	if err != nil {
		var statusError *k8serrors.StatusError
		if errors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				addSecretValsToTemplate = false
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	for i, template := range wf.Spec.Templates {
		if template.Container == nil {
			continue
		}

		wf.Spec.Templates[i].Outputs.Artifacts = append(template.Outputs.Artifacts, wfv1.Artifact{
			Name:     "metrics",
			Path:     "/tmp/metrics.json",
			Optional: true,
			Archive: &wfv1.ArchiveStrategy{
				None: &wfv1.NoneStrategy{},
			},
		})

		if !addSecretValsToTemplate {
			continue
		}

		//Generate ENV vars from secret, if there is a container present in the workflow
		//Get template ENV vars, avoid over-writing them with secret values
		for key, value := range secret.Data {
			//Flag to prevent over-writing user's envs
			addSecretAsEnv := true
			for _, templateEnv := range template.Container.Env {
				if templateEnv.Name == key {
					addSecretAsEnv = false
					break
				}
			}
			if addSecretAsEnv {
				template.Container.Env = append(template.Container.Env, corev1.EnvVar{
					Name:  key,
					Value: string(value),
				})
			}
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
	workflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})

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

func (c *Client) RetryWorkflow(namespace, name string) (workflow *Workflow, err error) {
	workflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	workflow, err = util.RetryWorkflow(c, c.ArgoprojV1alpha1().Workflows(namespace), workflow)

	return
}

func (c *Client) ResubmitWorkflow(namespace, name string, memoized bool) (workflow *model.Workflow, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	wf, err = util.FormulateResubmitWorkflow(wf, memoized)
	if err != nil {
		return
	}

	wf, err = util.SubmitWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), c, namespace, wf, &util.SubmitOpts{})
	if err != nil {
		return
	}

	workflow = modelWorkflow(wf)

	return
}

func (c *Client) ResumeWorkflow(namespace, name string) (workflow *Workflow, err error) {
	err = util.ResumeWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name)
	if err != nil {
		return
	}

	workflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})

	return
}

func (c *Client) SuspendWorkflow(namespace, name string) (err error) {
	err = util.SuspendWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name)

	return
}

func (c *Client) TerminateWorkflow(namespace, name string) (err error) {
	err = argoutil.TerminateWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name)

	return
}
