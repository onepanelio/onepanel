package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argojson "github.com/argoproj/pkg/json"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"sort"
	"strings"
)

func (c *Client) UpdateCronWorkflow(namespace string, name string, cronWorkflow *CronWorkflow) (*CronWorkflow, error) {
	workflow := cronWorkflow.WorkflowExecution
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
	var argoCronWorkflow wfv1.CronWorkflow
	argoCronWorkflow.Spec.Schedule = cronWorkflow.Schedule
	argoCronWorkflow.Spec.Timezone = cronWorkflow.Timezone
	argoCronWorkflow.Spec.Suspend = cronWorkflow.Suspend
	argoCronWorkflow.Spec.ConcurrencyPolicy = wfv1.ConcurrencyPolicy(cronWorkflow.ConcurrencyPolicy)
	argoCronWorkflow.Spec.StartingDeadlineSeconds = cronWorkflow.StartingDeadlineSeconds
	argoCronWorkflow.Spec.SuccessfulJobsHistoryLimit = cronWorkflow.SuccessfulJobsHistoryLimit
	argoCronWorkflow.Spec.FailedJobsHistoryLimit = cronWorkflow.FailedJobsHistoryLimit
	//UX prevents multiple workflows
	workflows, err := UnmarshalWorkflows([]byte(workflowTemplate.Manifest), true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	for _, wf := range workflows {
		argoCronWorkflow.Spec.WorkflowSpec = wf.Spec
		argoCreatedCronWorkflow, err := c.updateCronWorkflow(namespace, name, &argoCronWorkflow, opts)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":    namespace,
				"CronWorkflow": cronWorkflow,
				"Error":        err.Error(),
			}).Error("Error parsing workflow.")
			return nil, err
		}
		cronWorkflow.Name = argoCreatedCronWorkflow.Name
		cronWorkflow.CreatedAt = argoCreatedCronWorkflow.CreationTimestamp.UTC()
		cronWorkflow.UID = string(argoCreatedCronWorkflow.ObjectMeta.UID)
		cronWorkflow.WorkflowExecution.WorkflowTemplate = workflowTemplate
		// Manifests could get big, don't return them in this case.
		cronWorkflow.WorkflowExecution.WorkflowTemplate.Manifest = ""

		return cronWorkflow, nil
	}
	return nil, nil
}

func (c *Client) CreateCronWorkflow(namespace string, cronWorkflow *CronWorkflow) (*CronWorkflow, error) {

	workflow := cronWorkflow.WorkflowExecution
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
	var argoCronWorkflow wfv1.CronWorkflow
	argoCronWorkflow.Spec.Schedule = cronWorkflow.Schedule
	argoCronWorkflow.Spec.Timezone = cronWorkflow.Timezone
	argoCronWorkflow.Spec.Suspend = cronWorkflow.Suspend
	argoCronWorkflow.Spec.ConcurrencyPolicy = wfv1.ConcurrencyPolicy(cronWorkflow.ConcurrencyPolicy)
	argoCronWorkflow.Spec.StartingDeadlineSeconds = cronWorkflow.StartingDeadlineSeconds
	argoCronWorkflow.Spec.SuccessfulJobsHistoryLimit = cronWorkflow.SuccessfulJobsHistoryLimit
	argoCronWorkflow.Spec.FailedJobsHistoryLimit = cronWorkflow.FailedJobsHistoryLimit
	//UX prevents multiple workflows
	workflows, err := UnmarshalWorkflows([]byte(workflowTemplate.Manifest), true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	for _, wf := range workflows {
		argoCronWorkflow.Spec.WorkflowSpec = wf.Spec
		argoCreatedCronWorkflow, err := c.createCronWorkflow(namespace, &argoCronWorkflow, opts)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":    namespace,
				"CronWorkflow": cronWorkflow,
				"Error":        err.Error(),
			}).Error("Error parsing workflow.")
			return nil, err
		}
		cronWorkflow.Name = argoCreatedCronWorkflow.Name
		cronWorkflow.CreatedAt = argoCreatedCronWorkflow.CreationTimestamp.UTC()
		cronWorkflow.UID = string(argoCreatedCronWorkflow.ObjectMeta.UID)
		cronWorkflow.WorkflowExecution.WorkflowTemplate = workflowTemplate
		// Manifests could get big, don't return them in this case.
		cronWorkflow.WorkflowExecution.WorkflowTemplate.Manifest = ""

		return cronWorkflow, nil
	}
	return nil, nil
}

func (c *Client) GetCronWorkflow(namespace, name string) (cronWorkflow *CronWorkflow, err error) {
	cwf, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

	cronWorkflow = &CronWorkflow{
		CreatedAt:                  cwf.CreationTimestamp.UTC(),
		UID:                        string(cwf.UID),
		Name:                       cwf.Name,
		Schedule:                   cwf.Spec.Schedule,
		Timezone:                   cwf.Spec.Timezone,
		Suspend:                    cwf.Spec.Suspend,
		ConcurrencyPolicy:          string(cwf.Spec.ConcurrencyPolicy),
		StartingDeadlineSeconds:    cwf.Spec.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: cwf.Spec.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     cwf.Spec.FailedJobsHistoryLimit,
		WorkflowExecution:          nil,
	}

	return
}

func (c *Client) ListCronWorkflows(namespace string) (cronWorkflows []*CronWorkflow, err error) {
	cronWorkflowList, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).List(ListOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("CronWorkflows not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflows not found.")
	}
	cwfs := cronWorkflowList.Items
	sort.Slice(cwfs, func(i, j int) bool {
		ith := cwfs[i].CreationTimestamp.Time
		jth := cwfs[j].CreationTimestamp.Time
		//Most recent first
		return ith.After(jth)
	})

	for _, cwf := range cwfs {
		cronWorkflows = append(cronWorkflows, &CronWorkflow{
			CreatedAt:                  cwf.CreationTimestamp.UTC(),
			UID:                        string(cwf.ObjectMeta.UID),
			Name:                       cwf.Name,
			Schedule:                   cwf.Spec.Schedule,
			Timezone:                   cwf.Spec.Timezone,
			Suspend:                    cwf.Spec.Suspend,
			ConcurrencyPolicy:          string(cwf.Spec.ConcurrencyPolicy),
			StartingDeadlineSeconds:    cwf.Spec.StartingDeadlineSeconds,
			SuccessfulJobsHistoryLimit: cwf.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     cwf.Spec.FailedJobsHistoryLimit,
			WorkflowExecution:          nil,
		})
	}
	return
}

func (c *Client) updateCronWorkflow(namespace string, name string, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions) (updatedCronWorkflow *wfv1.CronWorkflow, err error) {
	//Make sure the CronWorkflow exists before we edit it
	toUpdateCWF, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

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

	cwf.Name = name
	cwf.ResourceVersion = toUpdateCWF.ResourceVersion
	updatedCronWorkflow, err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Update(cwf)
	if err != nil {
		return nil, err
	}

	return
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

func (c *Client) TerminateCronWorkflow(namespace, name string) (err error) {
	err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Delete(name, nil)
	return
}

func unmarshalCronWorkflows(cwfBytes []byte, strict bool) (cwfs wfv1.CronWorkflow, err error) {
	var cwf wfv1.CronWorkflow
	var jsonOpts []argojson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)
	}
	err = argojson.Unmarshal(cwfBytes, &cwf, jsonOpts...)
	if err == nil {
		return cwf, nil
	}
	return
}
