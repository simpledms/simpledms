package managetags

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/action/tagging"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

type Actions struct {
	Common  *acommon.Actions
	Tagging *tagging.Actions

	ManageTagsPage *ManageTagsPage
	ToggleTagGroup *ToggleTagGroup
	TagList        *TagList
	// TagDetails     *TagDetails
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions, taggingActions *tagging.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common:  commonActions,
		Tagging: taggingActions,

		ManageTagsPage: NewManageTagsPage(infra, actions),
		ToggleTagGroup: NewToggleTagGroup(infra, actions),
		TagList:        NewTagList(infra, actions),
		// TagDetails:     NewTagDetails(infra, actions),
	}
	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageTagsActionsRoute() + path
}
