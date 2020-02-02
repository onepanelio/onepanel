package server

import (
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
)

var userError *util.UserError

func modelListOptions(lo *api.ListOptions) (listOptions model.ListOptions) {
	if lo == nil {
		listOptions = model.ListOptions{}
		return
	}
	listOptions = model.ListOptions{
		LabelSelector: lo.LabelSelector,
		FieldSelector: lo.FieldSelector,
	}

	return
}
