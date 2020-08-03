package label

import (
	"strings"
)

const (
	OnepanelPrefix              = "onepanel.io/"
	TagPrefix                   = "tags.onepanel.io/"
	WorkflowTemplate            = OnepanelPrefix + "workflow-template"
	WorkflowTemplateUid         = OnepanelPrefix + "workflow-template-uid"
	WorkflowTemplateVersionUid  = OnepanelPrefix + "workflow-template-version-uid"
	WorkspaceTemplateVersionUid = OnepanelPrefix + "workspace-template-version-uid"
	WorkflowUid                 = OnepanelPrefix + "workflow-uid"
	CronWorkflowUid             = OnepanelPrefix + "cron-workflow-uid"
	Version                     = OnepanelPrefix + "version"
	VersionLatest               = OnepanelPrefix + "version-latest"
)

// Function that modifies an input string
type StringModifier func(string) string

// Returns a map where only the keys that have the input prefix are kept.
func FilterByPrefix(prefix string, inputLabels map[string]string) (labels map[string]string) {
	labels = make(map[string]string)

	for key, value := range inputLabels {
		if strings.HasPrefix(key, prefix) {
			labels[key] = value
		}
	}

	return
}

func RemovePrefix(prefix string, inputLabels map[string]string) (labels map[string]string) {
	labels = make(map[string]string)

	prefixLen := len(prefix)
	for key, value := range inputLabels {
		formattedKey := key[prefixLen:]
		labels[formattedKey] = value
	}

	return
}

// Delete all of the keys in the inputLabels
func Delete(inputLabels map[string]string, keys ...string) {
	for _, key := range keys {
		delete(inputLabels, key)
	}

	return
}

// Delete all keys that have the passed in prefix.
func DeleteWithPrefix(inputLabels map[string]string, prefix string) {
	for key := range inputLabels {
		if strings.HasPrefix(key, prefix) {
			delete(inputLabels, key)
		}
	}
}

// Adds all of the key/values in additions to destination.
// Each key is formatted according to the modifier function.
func MergeLabels(destination, additions map[string]string, modifier StringModifier) {
	for key, value := range additions {
		formattedKey := modifier(key)
		destination[formattedKey] = value
	}
}

// Adds all of the keys/values in additions to destination
// Each key in addition will be modified by having prefix prepended to it.
func MergeLabelsPrefix(destination, additions map[string]string, prefix string) {
	MergeLabels(destination, additions, func(s string) string {
		return prefix + s
	})
}
