package data

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// ManifestFile represents a file that contains information about a workflow or workspace template
type ManifestFile struct {
	Metadata ManifestFileMetadata `yaml:"metadata"`
	Spec     interface{}          `yaml:"spec"`
}

// ManifestFileMetadata represents information about the tempalte we are working with
type ManifestFileMetadata struct {
	Name        string
	Kind        string // {Workflow, Workspace}
	Version     uint64
	Action      string // {create,update}
	Description *string
	Labels      map[string]string
	Deprecated  *bool
	Source      *string
}

// SpecString returns the spec of a manifest file as a string
func (m *ManifestFile) SpecString() (string, error) {
	data, err := yaml.Marshal(m.Spec)
	if err != nil {
		return "", err
	}

	return string(data), err
}

// ManifestFileFromFile loads a manifest from a yaml file.
func ManifestFileFromFile(path string) (*ManifestFile, error) {
	fileData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	manifest := &ManifestFile{}
	if err := yaml.Unmarshal(fileData, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}
