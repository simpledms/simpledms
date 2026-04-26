package action

import (
	coreaction "github.com/marcobeierer/go-core/action"
	"github.com/simpledms/simpledms/action/browse"
	"github.com/simpledms/simpledms/action/dashboard"
	"github.com/simpledms/simpledms/action/documenttype"
	"github.com/simpledms/simpledms/action/inbox"
	"github.com/simpledms/simpledms/action/managespaceusers"
	"github.com/simpledms/simpledms/action/managetags"
	"github.com/simpledms/simpledms/action/managetenantusers"
	"github.com/simpledms/simpledms/action/openfile"
	"github.com/simpledms/simpledms/action/property"
	"github.com/simpledms/simpledms/action/spaces"
	"github.com/simpledms/simpledms/action/tagging"
	"github.com/simpledms/simpledms/action/trash"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
)

// value of actions tag seems not important, just that not empty...
// see Router.RegisterActions:54
// TODO rename to Pages?
type Actions struct {
	Dashboard *dashboard.Actions `actions:"dashboard"`
	Browse    *browse.Actions    `actions:"browse"`
	Tagging   *tagging.Actions   `actions:"tagging"`
	Inbox     *inbox.Actions     `actions:"inbox"`
	// Find         *findq.Actions        `actions:"find"`
	DocumentType      *documenttype.Actions      `actions:"documentType"`
	Spaces            *spaces.Actions            `actions:"spaces"`
	Property          *property.Actions          `actions:"property"`
	OpenFile          *openfile.Actions          `actions:"openFile"`
	ManageTags        *managetags.Actions        `actions:"manageTags"`
	ManageTenantUsers *managetenantusers.Actions `actions:"manageTenantUsers"`
	ManageSpaceUsers  *managespaceusers.Actions  `actions:"manageSpaceUsers"`
	Trash             *trash.Actions             `actions:"trash"`
}

func NewActions(
	infra *common.Infra,
	tenantDBs *tenantdbs.TenantDBs,
	isDevMode bool,
	coreActions *coreaction.Actions,
) *Actions {
	registerFormDecoderSetup()

	commonActions := coreActions.Common
	taggingActions := tagging.NewActions(infra, commonActions)
	browseActions := browse.NewActions(infra, commonActions, taggingActions)
	documentTypeActions := documenttype.NewActions(infra, commonActions)
	spacesActions := spaces.NewActions(infra)
	trashActions := trash.NewActions(infra, browseActions)

	return &Actions{
		Dashboard: dashboard.NewActions(
			infra,
			commonActions,
			coreActions.Auth,
			coreActions.Admin,
		),
		Browse:            browseActions,
		Tagging:           taggingActions,
		Inbox:             inbox.NewActions(infra, commonActions, browseActions),
		DocumentType:      documentTypeActions,
		Spaces:            spacesActions,
		Property:          property.NewActions(infra),
		OpenFile:          openfile.NewActions(infra, commonActions, tenantDBs, isDevMode),
		ManageTags:        managetags.NewActions(infra, commonActions, taggingActions),
		ManageTenantUsers: managetenantusers.NewActions(infra),
		ManageSpaceUsers:  managespaceusers.NewActions(infra),
		Trash:             trashActions,
	}
}
