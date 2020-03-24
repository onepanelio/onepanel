package v1

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"regexp"
	"strings"
)
func (c *Client) createCronWorkflow(namespace string, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions)(createdCronWorkflow *wfv1.CronWorkflow, err error)  {
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