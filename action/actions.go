package action

import (
	"github.com/simpledms/simpledms/action/about"
	"github.com/simpledms/simpledms/action/admin"
	"github.com/simpledms/simpledms/action/auth"
	"github.com/simpledms/simpledms/action/browse"
	acommon "github.com/simpledms/simpledms/action/common"
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
	Common    *acommon.Actions   `actions:"common"`
	// Find         *findq.Actions        `actions:"find"`
	DocumentType      *documenttype.Actions      `actions:"documentType"`
	Spaces            *spaces.Actions            `actions:"spaces"`
	Auth              *auth.Actions              `actions:"auth"`
	Property          *property.Actions          `actions:"property"`
	OpenFile          *openfile.Actions          `actions:"openFile"`
	Admin             *admin.Actions             `actions:"admin"`
	ManageTags        *managetags.Actions        `actions:"manageTags"`
	ManageTenantUsers *managetenantusers.Actions `actions:"manageTenantUsers"`
	ManageSpaceUsers  *managespaceusers.Actions  `actions:"manageSpaceUsers"`
	About             *about.Actions             `actions:"about"`
}

func NewActions(infra *common.Infra, tenantDBs *tenantdbs.TenantDBs) *Actions {
	commonActions := acommon.NewActions(infra)
	taggingActions := tagging.NewActions(infra, commonActions)
	browseActions := browse.NewActions(infra, commonActions, taggingActions)
	documentTypeActions := documenttype.NewActions(infra, commonActions)
	spacesActions := spaces.NewActions(infra)
	authActions := auth.NewActions(infra)
	adminActions := admin.NewActions(infra)

	return &Actions{
		Dashboard:         dashboard.NewActions(infra, commonActions, authActions, adminActions),
		Browse:            browseActions,
		Tagging:           taggingActions,
		Inbox:             inbox.NewActions(infra, commonActions, browseActions),
		Common:            commonActions,
		DocumentType:      documentTypeActions,
		Spaces:            spacesActions,
		Auth:              authActions,
		Property:          property.NewActions(infra),
		OpenFile:          openfile.NewActions(infra, commonActions, tenantDBs),
		Admin:             adminActions,
		ManageTags:        managetags.NewActions(infra, commonActions, taggingActions),
		ManageTenantUsers: managetenantusers.NewActions(infra),
		ManageSpaceUsers:  managespaceusers.NewActions(infra),
		About:             about.NewActions(infra),
	}
}
