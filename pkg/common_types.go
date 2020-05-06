package v1

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
	displayName := paramMap["displayname"].(string)
	hint := paramMap["hint"].(string)
	required := paramMap["required"].(bool)
	typeValue := paramMap["type"].(string)
	name := paramMap["name"].(string)
	value := paramMap["value"].(string)

	options := paramMap["options"]
	optionsArray, ok := options.([]interface{})
	if !ok {
		return nil
	}

	newOptions := make([]ParameterOption, 0)
	for _, option := range optionsArray {
		optionMap := option.(map[interface{}]interface{})

		newOption := ParameterOption{
			Name:  optionMap["name"].(string),
			Value: optionMap["value"].(string),
		}

		newOptions = append(newOptions, newOption)
	}

	workflowParameter := Parameter{
		Name:     name,
		Required: required,
	}

	if displayName != "" {
		workflowParameter.DisplayName = &displayName
	}
	if hint != "" {
		workflowParameter.Hint = &hint
	}
	if value != "" {
		workflowParameter.Value = &value
	}
	if typeValue != "" {
		workflowParameter.Type = typeValue
	}

	return &workflowParameter
}

type Arguments struct {
	Parameters []Parameter `json:"parameters" protobuf:"bytes,1,opt,name=parameters"`
}
