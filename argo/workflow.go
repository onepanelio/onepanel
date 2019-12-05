package argo

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/pkg/json"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type Workflow = wfv1.Workflow

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

func (c *Client) create(wf *Workflow, parameters []string) (createdWf *Workflow, err error) {
	if len(parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				return nil, fmt.Errorf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
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

	createdWf, err = c.WorkflowInterface.Create(wf)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) Create(workflowTemplate string, parameters []string) (workflowNames []string, err error) {
	workflows, err := unmarshalWorkflows([]byte(workflowTemplate), true)
	if err != nil {
		return nil, err
	}

	for _, wf := range workflows {
		createdWf, err := c.create(&wf, parameters)
		if err != nil {
			return nil, err
		}
		workflowNames = append(workflowNames, createdWf.Name)
	}

	return
}
