package manager

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
)

func (r *ResourceManager) CreateWorkflow(namespace string, workflow *model.Workflow) (*model.Workflow, error) {
	workflowTemplate, err := r.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		return nil, err
	}

	// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &argo.Options{
		Namespace: namespace,
	}
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, argo.Parameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}
	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[viper.GetString("k8s.labelKeyPrefix")+"workflow-template-uid"] = workflowTemplate.UID
	(*opts.Labels)[viper.GetString("k8s.labelKeyPrefix")+"workflow-template-version"] = fmt.Sprint(workflowTemplate.Version)
	createdWorkflows, err := r.argClient.CreateWorkflow(workflowTemplate.GetManifestBytes(), opts)
	if err != nil {
		return nil, err
	}

	workflow.Name = createdWorkflows[0].Name
	workflow.UID = string(createdWorkflows[0].ObjectMeta.UID)
	workflow.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	workflow.WorkflowTemplate.Manifest = ""

	return workflow, nil
}

func (r *ResourceManager) GetWorkflow(namespace, name string) (workflow *model.Workflow, err error) {
	wf, err := r.argClient.GetWorkflow(name, &argo.Options{Namespace: namespace})
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	uid := wf.ObjectMeta.Labels[viper.GetString("k8s.labelKeyPrefix")+"workflow-template-uid"]
	version, err := strconv.ParseInt(
		wf.ObjectMeta.Labels[viper.GetString("k8s.labelKeyPrefix")+"workflow-template-version"],
		10,
		32,
	)
	if err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid version number.")
	}
	workflowTemplate, err := r.GetWorkflowTemplate(namespace, uid, int32(version))
	if err != nil {
		return
	}

	// TODO: Do we need to parse parameters into workflow.Parameters?
	status, err := json.Marshal(wf.Status)
	if err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid status.")
	}
	workflow = &model.Workflow{
		UID:              string(wf.UID),
		Name:             wf.Name,
		Status:           string(status),
		WorkflowTemplate: workflowTemplate,
	}

	return
}

func (r *ResourceManager) WatchWorkflow(namespace, name string) (<-chan *model.Workflow, error) {
	wf, err := r.GetWorkflow(namespace, name)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	watcher, err := r.argClient.WatchWorkflow(name, &argo.Options{Namespace: namespace})
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	var workflow *argo.Workflow
	workflowWatcher := make(chan *model.Workflow)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case next := <-watcher.ResultChan():
				workflow, _ = next.Object.(*argo.Workflow)
			case <-ticker.C:
			}

			if workflow == nil {
				continue
			}
			status, err := json.Marshal(workflow.Status)
			if err != nil {
				continue
			}
			workflowWatcher <- &model.Workflow{
				UID:              string(workflow.UID),
				Name:             workflow.Name,
				Status:           string(status),
				WorkflowTemplate: wf.WorkflowTemplate,
			}

			if !workflow.Status.FinishedAt.IsZero() {
				break
			}
		}
		close(workflowWatcher)
		watcher.Stop()
	}()

	return workflowWatcher, nil
}

func (r *ResourceManager) ListWorkflows(namespace, workflowTemplateUID string) (workflows []*model.Workflow, err error) {
	opts := &argo.Options{
		Namespace: namespace,
	}
	if workflowTemplateUID != "" {
		opts.ListOptions = &argo.ListOptions{
			LabelSelector: fmt.Sprintf("%sworkflow-template-uid=%s", viper.GetString("k8s.labelKeyPrefix"), workflowTemplateUID),
		}
	}
	wfs, err := r.argClient.ListWorkflows(opts)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflows not found.")
	}

	for _, wf := range wfs {
		workflows = append(workflows, &model.Workflow{
			Name: wf.ObjectMeta.Name,
			UID:  string(wf.ObjectMeta.UID),
		})
	}

	return
}

func (r *ResourceManager) CreateWorkflowTemplate(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	workflowTemplate, err := r.workflowRepository.CreateWorkflowTemplate(namespace, workflowTemplate)
	if err != nil {
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}

	return workflowTemplate, nil
}

func (r *ResourceManager) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *model.WorkflowTemplate, err error) {
	workflowTemplate, err = r.workflowRepository.GetWorkflowTemplate(namespace, uid, version)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if err == nil && workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return
}

func (r *ResourceManager) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	workflowTemplateVersions, err = r.workflowRepository.ListWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template versions not found.")
	}

	return
}

func (r *ResourceManager) ListWorkflowTemplates(namespace string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	workflowTemplateVersions, err = r.workflowRepository.ListWorkflowTemplates(namespace)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow templates not found.")
	}

	return
}
