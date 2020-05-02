package v1

import (
	"github.com/google/uuid"
	"time"
)

type WorkspaceTemplate struct {
	ID               uint64
	UID              string
	Name             string
	Version          int64
	Manifest         string
	IsLatest         bool
	CreatedAt        time.Time `db:"created_at"`
	WorkflowTemplate *WorkflowTemplate
}

func (wt *WorkspaceTemplate) GenerateUID() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	wt.UID = uid.String()

	return wt.UID, nil
}
