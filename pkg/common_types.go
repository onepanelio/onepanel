package v1

import (
	"fmt"
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
	DisplayName *string            `json:"displayName,omitempty" yaml:"displayName"`
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

// Arguments are the arguments in a manifest file.
type Arguments struct {
	Parameters []Parameter `json:"parameters"`
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
