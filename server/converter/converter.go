package converter

import (
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"sort"
	"time"
)

func APIKeyValueToLabel(apiKeyValues []*api.KeyValue) map[string]string {
	result := make(map[string]string)
	if apiKeyValues == nil {
		return result
	}

	for _, entry := range apiKeyValues {
		result[entry.Key] = entry.Value
	}

	return result
}

func MappingToKeyValue(mapping map[string]string) []*api.KeyValue {
	keyValues := make([]*api.KeyValue, 0)

	for key, value := range mapping {
		keyValues = append(keyValues, &api.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	sort.Slice(keyValues, func(i, j int) bool {
		return keyValues[i].Key < keyValues[j].Key
	})

	return keyValues
}

// MetricsToAPI converts Metrics to the API version
func MetricsToAPI(metrics v1.Metrics) []*api.Metric {
	result := make([]*api.Metric, 0)

	for _, metric := range metrics {
		newItem := &api.Metric{
			Name:   metric.Name,
			Value:  metric.Value,
			Format: metric.Format,
		}

		result = append(result, newItem)
	}

	return result
}

// APIMetricsToCore converts []*api.Metric to v1.Metrics
func APIMetricsToCore(metrics []*api.Metric) v1.Metrics {
	result := v1.Metrics{}

	for _, metric := range metrics {
		m := v1.Metric{
			Name:   metric.Name,
			Value:  metric.Value,
			Format: metric.Format,
		}

		// We don't override anything because we want the input to match how the user entered it
		// So if there are entries with the same name, that's fine.
		result.Add(&m, false)
	}

	return result
}

// LabelsToKeyValues converts []*v1.Label to []*api.Label
func LabelsToKeyValues(labels []*v1.Label) []*api.KeyValue {
	keyValues := make([]*api.KeyValue, 0)

	for _, label := range labels {
		keyValues = append(keyValues, LabelToKeyValue(label))
	}

	sort.Slice(keyValues, func(i, j int) bool {
		return keyValues[i].Key < keyValues[j].Key
	})

	return keyValues
}

// LabelToKeyValue converts a *v1.Label to *api.Label
func LabelToKeyValue(label *v1.Label) *api.KeyValue {
	return &api.KeyValue{
		Key:   label.Key,
		Value: label.Value,
	}
}

func ParameterOptionToAPI(option *v1.ParameterOption) *api.ParameterOption {
	apiOption := &api.ParameterOption{
		Name:  option.Name,
		Value: option.Value,
	}

	return apiOption
}

func APIParameterOptionToInternal(option *api.ParameterOption) *v1.ParameterOption {
	result := &v1.ParameterOption{
		Name:  option.Name,
		Value: option.Value,
	}

	return result
}

func ParameterOptionsToAPI(options []*v1.ParameterOption) []*api.ParameterOption {
	result := make([]*api.ParameterOption, len(options))

	for i := range options {
		newItem := ParameterOptionToAPI(options[i])
		result[i] = newItem
	}

	return result
}

func APIParameterOptionsToInternal(options []*api.ParameterOption) []*v1.ParameterOption {
	result := make([]*v1.ParameterOption, len(options))

	for i := range options {
		newItem := APIParameterOptionToInternal(options[i])
		result[i] = newItem
	}

	return result
}

// ParameterToAPI converts a v1.Parameter to a *api.Parameter
func ParameterToAPI(param v1.Parameter) *api.Parameter {
	apiParam := &api.Parameter{
		Name:     param.Name,
		Type:     param.Type,
		Required: param.Required,
	}
	if param.Value != nil {
		apiParam.Value = *param.Value
	}
	if param.DisplayName != nil {
		apiParam.DisplayName = *param.DisplayName
	}
	if param.Hint != nil {
		apiParam.Hint = *param.Hint
	}
	if param.Visibility != nil {
		apiParam.Visibility = *param.Visibility
	}
	if param.Options != nil {
		apiParam.Options = ParameterOptionsToAPI(param.Options)
	}

	return apiParam
}

func ParametersToAPI(params []v1.Parameter) []*api.Parameter {
	result := make([]*api.Parameter, len(params))

	for i := range params {
		newItem := ParameterToAPI(params[i])
		result[i] = newItem
	}

	return result
}

func APIParameterToInternal(param *api.Parameter) *v1.Parameter {
	result := &v1.Parameter{
		Name:     param.Name,
		Type:     param.Type,
		Required: param.Required,
	}

	if param.Value != "" {
		result.Value = &param.Value
	}
	if param.DisplayName != "" {
		result.DisplayName = &param.DisplayName
	}
	if param.Hint != "" {
		result.Hint = &param.Hint
	}

	if param.Options != nil {
		result.Options = APIParameterOptionsToInternal(param.Options)
	}

	return result
}

// TimestampToAPIString converts a *time.Time to an API string in the RFC3339 format
// if ts is nil, an empty string is returned
func TimestampToAPIString(ts *time.Time) string {
	if ts == nil {
		return ""
	}

	return ts.UTC().Format(time.RFC3339)
}

// WorkflowExecutionStatisticsReportToAPI converts v1.WorkflowExecutionStatisticReport to api.WorkflowExecutionStatisticReport
func WorkflowExecutionStatisticsReportToAPI(report *v1.WorkflowExecutionStatisticReport) *api.WorkflowExecutionStatisticReport {
	if report == nil {
		return nil
	}

	stats := &api.WorkflowExecutionStatisticReport{
		Total:        report.Total,
		Running:      report.Running,
		Completed:    report.Completed,
		Failed:       report.Failed,
		Terminated:   report.Terminated,
		LastExecuted: TimestampToAPIString(report.LastExecuted),
	}

	return stats
}

// WorkspaceStatisticsReportToAPI converts v1.WorkspaceStatisticReport to api.WorkspaceStatisticReport
func WorkspaceStatisticsReportToAPI(report *v1.WorkspaceStatisticReport) *api.WorkspaceStatisticReport {
	if report == nil {
		return nil
	}

	stats := &api.WorkspaceStatisticReport{
		LastCreated:       TimestampToAPIString(report.LastCreated),
		Launching:         report.Launching,
		Running:           report.Running,
		Updating:          report.Updating,
		Pausing:           report.Pausing,
		Paused:            report.Paused,
		Terminating:       report.Terminating,
		Terminated:        report.Terminated,
		FailedToPause:     report.FailedToPause,
		FailedToResume:    report.FailedToResume,
		FailedToTerminate: report.FailedToTerminate,
		FailedToLaunch:    report.FailedToLaunch,
		FailedToUpdate:    report.FailedToUpdate,
		Failed:            report.Failed,
		Total:             report.Total,
	}

	return stats
}
