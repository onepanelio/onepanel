package v1

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"regexp"
	"strings"
)

func (c *Client) CreateCronWorkflow(namespace string, cronWorkflow *CronWorkflow) (*CronWorkflow, error) {

	//todo get CronWorkflowTemplate?
	//todo moving todo
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &WorkflowExecutionOptions{}
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	opts.GenerateName = strings.ToLower(re.ReplaceAllString(workflowTemplate.Name, `-`)) + "-"
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, WorkflowExecutionParameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	(*opts.Labels)[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	workflows, err := unmarshalWorkflows([]byte(workflowTemplate.Manifest), true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	var createdWorkflows []*wfv1.Workflow
	for _, wf := range workflows {
		createdWorkflow, err := c.createWorkflow(namespace, &wf, opts)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":    namespace,
				"CronWorkflow": cronWorkflow,
				"Error":        err.Error(),
			}).Error("Error parsing workflow.")
			return nil, err
		}
		createdWorkflows = append(createdWorkflows, createdWorkflow)
	}

	workflow.Name = createdWorkflows[0].Name
	workflow.CreatedAt = createdWorkflows[0].CreationTimestamp.UTC()
	workflow.UID = string(createdWorkflows[0].ObjectMeta.UID)
	workflow.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	workflow.WorkflowTemplate.Manifest = ""

	return cronWorkflow, nil
}

func (c *Client) createCronWorkflow(namespace string, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions) (createdCronWorkflow *wfv1.CronWorkflow, err error) {
	if opts == nil {
		opts = &WorkflowExecutionOptions{}
	}

	if opts.Name != "" {
		cwf.ObjectMeta.Name = opts.Name
	}
	if opts.GenerateName != "" {
		cwf.ObjectMeta.GenerateName = opts.GenerateName
	}
	if opts.Entrypoint != "" {
		cwf.Spec.WorkflowSpec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		cwf.Spec.WorkflowSpec.ServiceAccountName = opts.ServiceAccount
	}
	if len(opts.Parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, param := range opts.Parameters {
			newParams = append(newParams, wfv1.Parameter{
				Name:  param.Name,
				Value: param.Value,
			})
			passedParams[param.Name] = true
		}

		for _, param := range cwf.Spec.WorkflowSpec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		cwf.Spec.WorkflowSpec.Arguments.Parameters = newParams
	}
	if opts.Labels != nil {
		cwf.ObjectMeta.Labels = *opts.Labels
	}

	//todo move this earlier in the process
	//if err = c.injectAutomatedFields(namespace, cwf.Spec.WorkflowSpec, opts); err != nil {
	//	return nil, err
	//}

	createdCronWorkflow, err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Create(cwf)
	if err != nil {
		return nil, err
	}

	return
}
