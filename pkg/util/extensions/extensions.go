package extensions

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

// NodePair is a convenience wrapper for two nodes, usually a key/value pair.
type NodePair struct {
	Key   *yaml.Node
	Value *yaml.Node
}

// YamlIndex identifies a path in a Yaml Node Tree
type YamlIndex struct {
	parts []string
}

// String returns the YamlIndex indicated by the parts separated by "."
// e.g. parent.children.favoriteNumber
func (y *YamlIndex) String() string {
	return strings.Join(y.parts, ".")
}

// CreateYamlIndex creates a YamlIndex that specifies the Key via string parts.
// e.g. a key maybe be: parent.child.favoriteNumber and the returned YamlIndex would reflect this.
// Note: this does not yet support indexing array values.
func CreateYamlIndex(parts ...string) *YamlIndex {
	copyParts := make([]string, len(parts))

	for i, part := range parts {
		copyParts[i] = part
	}

	return &YamlIndex{
		parts: copyParts,
	}
}

// HasNode returns true if the root node has the key
func HasNode(root *yaml.Node, key *YamlIndex) bool {
	if key == nil || len(key.parts) == 0 {
		return false
	}

	currentNode := root
	if len(root.Content) == 1 {
		currentNode = root.Content[0]
	}

	for _, keyPart := range key.parts {
		found := false
		for j := 0; j < len(currentNode.Content)-1; j += 2 {
			keyNode := currentNode.Content[j]
			valueNode := currentNode.Content[j+1]

			if keyNode.Value == keyPart {
				currentNode = valueNode
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// GetNode returns the node that contains the content for the key
// TODO support indexes
func GetNode(root *yaml.Node, key *YamlIndex) (*yaml.Node, error) {
	if key == nil || len(key.parts) == 0 {
		return root, nil
	}

	currentNode := root
	if len(root.Content) == 1 {
		currentNode = root.Content[0]
	}

	for _, keyPart := range key.parts {
		found := false
		for j := 0; j < len(currentNode.Content)-1; j += 2 {
			keyNode := currentNode.Content[j]
			valueNode := currentNode.Content[j+1]

			if keyNode.Value == keyPart {
				currentNode = valueNode
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("%v not found - stopped at %v", key.String(), keyPart)
		}
	}

	return currentNode, nil
}

// SetKeyValue set's the content node's value to value for the indicated key
func SetKeyValue(node *yaml.Node, key string, value string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("not a mapping node")
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == key {
			valueNode.Value = value
			break
		}
	}

	return nil
}

// HasKeyValue checks if the node (assumed to be a mapping node) has a key with the given value(s). If many values, any is ok.
// If it does not, (false, nil) is returned. If there is an error, like a key not existing, an error is returned.
func HasKeyValue(node *yaml.Node, key string, values ...string) (bool, error) {
	if node.Kind != yaml.MappingNode {
		return false, fmt.Errorf("not a mapping node")
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == key {
			for _, val := range values {
				if valueNode.Value == val {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetKeyValue gets the value of the key from the node (assumed to be a mapping node)
func GetKeyValue(node *yaml.Node, key string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("not a mapping node")
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == key {
			return valueNode, nil
		}
	}

	return nil, fmt.Errorf("not found")
}

// Iterate runs through all of the content nodes in the indicated root node
func Iterate(root *yaml.Node, callable func(parent, value *yaml.Node)) {
	for _, child := range root.Content {
		callable(root, child)
		Iterate(child, callable)
	}
}

// DeleteNode will delete the key and content nodes for the given key
func DeleteNode(node *yaml.Node, key *YamlIndex) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("not a mapping node")
	}

	currentNode := node
	for i, keyPart := range key.parts {
		found := false
		for j := 0; j < len(currentNode.Content)-1; j += 2 {
			keyNode := currentNode.Content[j]
			valueNode := currentNode.Content[j+1]

			if keyNode.Value == keyPart {
				if i != (len(key.parts) - 1) {
					currentNode = valueNode
				}
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("%v not found - stopped at %v", key.String(), keyPart)
		}
	}

	keptNodes := make([]*yaml.Node, 0)
	finalKey := key.parts[len(key.parts)-1]
	for i := 0; i < len(currentNode.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value != finalKey {
			keptNodes = append(keptNodes, keyNode, valueNode)
		}
	}

	currentNode.Content = keptNodes

	return nil
}

// ReplaceMapValues will replace strings that are keys in the input map with their values
// the result is returned
func ReplaceMapValues(value string, replaceMap map[string]string) string {
	replacePairs := make([]string, 0)

	for key, value := range replaceMap {
		replacePairs = append(replacePairs, key)
		replacePairs = append(replacePairs, value)
	}

	return strings.NewReplacer(replacePairs...).
		Replace(value)
}
