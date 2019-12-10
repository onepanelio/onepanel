package model

type WorkflowTemplate struct {
	ID       uint64
	UID      string
	Name     string
	Manifest string
}

func (wt *WorkflowTemplate) GetManifest() []byte {
	return []byte(wt.Manifest)
}
