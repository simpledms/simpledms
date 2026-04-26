package openfile

import (
	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	temporaryfilemodel "github.com/simpledms/simpledms/model/tenant/temporaryfile"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	Common            *acommon.Actions
	SelectSpacePage   *SelectSpacePage
	UploadFromURLPage *UploadFromURLPage
	UploadFromURLCmd  *UploadFromURLCmd
	UploadFilesCmd    *UploadFilesCmd
	// SelectSpace  *SelectSpace
}

func NewActions(
	infra *common.Infra,
	commonActions *acommon.Actions,
	tenantDBs *tenantdbs.TenantDBs,
	isDevMode bool,
) *Actions {
	var actions = new(Actions)
	uploadFromURLService := temporaryfilemodel.NewUploadFromURLService(
		infra.FileSystem(),
		isDevMode,
	)

	// cachex := NewFileUploadCache()

	*actions = Actions{
		Common:            commonActions,
		SelectSpacePage:   NewSelectSpacePage(infra, actions, tenantDBs),
		UploadFromURLCmd:  NewUploadFromURLCmd(actions, uploadFromURLService),
		UploadFromURLPage: NewUploadFromURLPage(infra, actions, uploadFromURLService),
		UploadFilesCmd:    NewUploadFilesCmd(infra, actions),
		// SelectSpace:  NewSelectSpace(infra, actions, cachex),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.OpenFileActionsRoute() + path
}
