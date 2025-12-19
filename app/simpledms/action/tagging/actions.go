package tagging

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

// TODO rename to FileActions?
type AssignedTagActions struct {
	List               *ListAssignedTags
	ListItem           *ListItemAssignedTags
	CreateAndAssignTag *CreateAndAssignTag // TODO on FileActions or TagActions
	Edit               *EditAssignedTags
	EditListItem       *EditAssignedTagsItem
	Count              *CountAssignedTags

	// TODO move to FileActions
	AssignTag   *AssignTag
	UnassignTag *UnassignTag
}

type SubTagActions struct {
	// TODO is name unique enough? only composed tags can have subtags, thus probably?
	List           *ListSubTags
	Edit           *EditSubTags
	AssignSubTag   *AssignSubTag
	UnassignSubTag *UnassignSubTag
}

type Actions struct {
	Common *acommon.Actions

	CreateTag      *CreateTag
	EditTag        *EditTag
	DeleteTag      *DeleteTag
	MoveTagToGroup *MoveTagToGroup

	ToggleFileTag *ToggleFileTag

	AssignedTags *AssignedTagActions `actions:"assigned-tags"`
	SubTags      *SubTagActions      `actions:"sub-tags"`
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common: commonActions,

		CreateTag:      NewCreateTag(infra, actions),
		EditTag:        NewEditTag(infra, actions),
		DeleteTag:      NewDeleteTag(infra, actions),
		MoveTagToGroup: NewMoveTagToGroup(infra, actions),

		ToggleFileTag: NewToggleFileTag(infra, actions),

		AssignedTags: &AssignedTagActions{
			List:               NewListAssignedTags(infra, actions),
			ListItem:           NewListItemAssignedTags(infra, actions),
			CreateAndAssignTag: NewCreateAndAssignTag(infra, actions),
			Edit:               NewEditAssignedTags(infra, actions),
			EditListItem:       NewEditAssignedTagsItem(infra, actions),
			Count:              NewCountAssignedTags(infra, actions),
			AssignTag:          NewAssignTag(infra, actions),
			UnassignTag:        NewUnassignTag(infra, actions),
		},

		SubTags: &SubTagActions{
			List:           NewListSubTags(infra, actions),
			Edit:           NewEditSubTags(infra, actions),
			AssignSubTag:   NewAssignSubTag(infra, actions),
			UnassignSubTag: NewUnassignSubTag(infra, actions),
		},
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.TaggingActionsRoute() + path
}
