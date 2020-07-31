package v1

import (
	"fmt"
	"github.com/onepanelio/core/pkg/util/ptr"
	"gopkg.in/yaml.v2"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ParameterOption struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

type Parameter struct {
	Name        string             `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value       *string            `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	Visibility  *string            `json:"visibility,omitempty"`
	Type        string             `json:"type,omitempty" protobuf:"bytes,3,opt,name=type"`
	DisplayName *string            `json:"displayName,omitempty" protobuf:"bytes,4,opt,name=displayName"`
	Hint        *string            `json:"hint,omitempty" protobuf:"bytes,5,opt,name=hint"`
	Options     []*ParameterOption `json:"options,omitempty" protobuf:"bytes,6,opt,name=options"`
	Required    bool               `json:"required,omitempty" protobuf:"bytes,7,opt,name=required"`
}

// IsValidParameter returns nil if the parameter is valid or an error otherwise
func IsValidParameter(parameter Parameter) error {
	if parameter.Visibility == nil {
		return nil
	}

	visibility := *parameter.Visibility
	if visibility != "public" && visibility != "protected" && visibility != "internal" && visibility != "private" {
		return fmt.Errorf("invalid visibility '%v' for parameter '%v'", visibility, parameter.Name)
	}

	return nil
}

// IsValidParameters returns nil if all parameters are valid or an error otherwise
func IsValidParameters(parameters []Parameter) error {
	for _, param := range parameters {
		if err := IsValidParameter(param); err != nil {
			return err
		}
	}

	return nil
}

type Arguments struct {
	Parameters []Parameter `json:"parameters" protobuf:"bytes,1,opt,name=parameters"`
}

// WorkflowTemplateManifest is a client representation of a WorkflowTemplate
// It is usually provided as YAML by a client and this struct helps to marshal/unmarshal it
type WorkflowTemplateManifest struct {
	Arguments Arguments
}

// WorkflowExecutionSpec is a client representation of a WorkflowExecution.
// It is usually provided as YAML by a client and this struct helps to marshal/unmarshal it
// This may be redundant with WorkflowTemplateManifest and should be looked at. # TODO
type WorkflowExecutionSpec struct {
	Arguments Arguments
}

// ParameterFromMap parses a parameter given a generic map of values
// this should not be used anyway in favor of yaml marshaling/unmarshaling
// left until it is refactored and tested
// Deprecated
func ParameterFromMap(paramMap map[interface{}]interface{}) *Parameter {
	workflowParameter := Parameter{
		Options: []*ParameterOption{},
	}

	// TODO choose a consistent way and use that.
	if value, ok := paramMap["displayname"]; ok {
		if displayName, ok := value.(string); ok {
			workflowParameter.DisplayName = &displayName
		}
	} else if value, ok := paramMap["displayName"]; ok {
		if displayName, ok := value.(string); ok {
			workflowParameter.DisplayName = &displayName
		}
	}

	if value, ok := paramMap["hint"]; ok {
		if hint, ok := value.(string); ok {
			workflowParameter.Hint = ptr.String(hint)
		}
	}

	if value, ok := paramMap["required"]; ok {
		if required, ok := value.(bool); ok {
			workflowParameter.Required = required
		}
	}

	if value, ok := paramMap["type"]; ok {
		if typeValue, ok := value.(string); ok {
			workflowParameter.Type = typeValue
		}
	}

	if value, ok := paramMap["name"]; ok {
		if nameValue, ok := value.(string); ok {
			workflowParameter.Name = nameValue
		}
	}

	if value, ok := paramMap["value"]; ok {
		if valueValue, ok := value.(string); ok {
			workflowParameter.Value = &valueValue
		}
	}

	options := paramMap["options"]
	optionsArray, ok := options.([]interface{})
	if !ok {
		return &workflowParameter
	}

	for _, option := range optionsArray {
		optionMap := option.(map[interface{}]interface{})

		newOption := ParameterOption{
			Name:  optionMap["name"].(string),
			Value: optionMap["value"].(string),
		}

		workflowParameter.Options = append(workflowParameter.Options, &newOption)
	}

	return &workflowParameter
}

// ParseParametersFromManifest takes a manifest and picks out the parameters and returns them as structs
func ParseParametersFromManifest(manifest []byte) ([]Parameter, error) {
	manifestResult := &WorkflowTemplateManifest{
		Arguments: Arguments{},
	}

	err := yaml.Unmarshal(manifest, manifestResult)
	if err != nil {
		return nil, err
	}

	if err := IsValidParameters(manifestResult.Arguments.Parameters); err != nil {
		return nil, err
	}

	return manifestResult.Arguments.Parameters, nil
}

// MapParametersByName returns a map where the parameter name is the key and the parameter is the value
func MapParametersByName(parameters []Parameter) map[string]Parameter {
	result := make(map[string]Parameter)

	for _, param := range parameters {
		result[param.Name] = param
	}

	return result
}
