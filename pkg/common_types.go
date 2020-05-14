package v1

import (
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
	Type        string             `json:"type" protobuf:"bytes,3,opt,name=type"`
	DisplayName *string            `json:"displayName" protobuf:"bytes,4,opt,name=displayName"`
	Hint        *string            `json:"hint" protobuf:"bytes,5,opt,name=hint"`
	Options     []*ParameterOption `json:"options,omitempty" protobuf:"bytes,6,opt,name=options"`
	Required    bool               `json:"required,omitempty" protobuf:"bytes,7,opt,name=required"`
}

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

type Arguments struct {
	Parameters []Parameter `json:"parameters" protobuf:"bytes,1,opt,name=parameters"`
}

func ParseParametersFromManifest(manifest []byte) ([]Parameter, error) {
	var parameters []Parameter

	mappedData := make(map[string]interface{})

	if err := yaml.Unmarshal(manifest, mappedData); err != nil {
		return nil, err
	}

	arguments, ok := mappedData["arguments"]
	if !ok {
		return parameters, nil
	}

	argumentsMap := arguments.(map[interface{}]interface{})
	parametersRaw, ok := argumentsMap["parameters"]
	if !ok {
		return parameters, nil
	}

	parametersArray, ok := parametersRaw.([]interface{})
	for _, parameter := range parametersArray {
		paramMap, ok := parameter.(map[interface{}]interface{})
		if !ok {
			continue
		}

		workflowParameter := ParameterFromMap(paramMap)

		parameters = append(parameters, *workflowParameter)
	}

	return parameters, nil
}
