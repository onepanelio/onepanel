package v1

import (
	"regexp"
	"strings"
	"time"
)

type WorkspaceTemplate struct {
	ID               uint64
	UID              string
	Name             string `valid:"stringlength(3|30)~Name should be between 3 to 30 characters,required"`
	Version          int64
	Manifest         string
	IsLatest         bool
	CreatedAt        time.Time         `db:"created_at"`
	WorkflowTemplate *WorkflowTemplate `db:"workflow_template"`
	Labels           map[string]string
}

func (wt *WorkspaceTemplate) GenerateUID() (string, error) {
	re, err := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	if err != nil {
		return "", err
	}
	wt.UID = strings.ToLower(re.ReplaceAllString(wt.Name, `-`))

	return wt.UID, nil
}
