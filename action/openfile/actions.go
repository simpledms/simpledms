package openfile

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/uix/route"
)

type Actions struct {
	Common          *acommon.Actions
	SelectSpacePage *SelectSpacePage
	UploadFilesCmd  *UploadFilesCmd
	// SelectSpace  *SelectSpace
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions, tenantDBs *tenantdbs.TenantDBs) *Actions {
	var actions = new(Actions)

	// cachex := NewFileUploadCache()

	*actions = Actions{
		Common:          commonActions,
		SelectSpacePage: NewSelectSpacePage(infra, actions, tenantDBs),
		UploadFilesCmd:  NewUploadFilesCmd(infra, actions),
		// SelectSpace:  NewSelectSpace(infra, actions, cachex),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.OpenFileActionsRoute() + path
}
