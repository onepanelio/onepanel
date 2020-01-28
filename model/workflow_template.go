package model

import (
	"github.com/google/uuid"
	"time"
)

type WorkflowTemplate struct {
	ID        uint64
	CreatedAt time.Time `db:"created_at"`
	UID       string
	Name      string
	Manifest  string
	Version   int32
	IsLatest  bool `db:"is_latest"`
}

func (wt *WorkflowTemplate) GetManifestBytes() []byte {
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
