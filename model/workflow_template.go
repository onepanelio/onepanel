package model

import "github.com/google/uuid"

type WorkflowTemplate struct {
	ID       uint64
	UID      string
	Name     string
	Manifest string
}

func (wt *WorkflowTemplate) GetManifest() []byte {
	return []byte(wt.Manifest)
}

func (wt *WorkflowTemplate) GenerateUID() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	wt.UID = uid.String()

	return wt.UID, nil
}
