package tagging

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

// TODO rename to FileActions?
type AssignedTagActions struct {
	List                  *ListAssignedTagsPartial
	ListItem              *ListItemAssignedTagsPartial
	CreateAndAssignTagCmd *CreateAndAssignTagCmd // TODO on FileActions or TagActions
	Edit                  *EditAssignedTagsPartial
	EditListItem          *EditAssignedTagsItemPartial
	Count                 *CountAssignedTagsPartial

	// TODO move to FileActions
	AssignTagCmd   *AssignTagCmd
	UnassignTagCmd *UnassignTagCmd
}

type SubTagActions struct {
	// TODO is name unique enough? only composed tags can have subtags, thus probably?
	List              *ListSubTagsPartial
	Edit              *EditSubTagsPartial
	AssignSubTagCmd   *AssignSubTagCmd
	UnassignSubTagCmd *UnassignSubTagCmd
}

type Actions struct {
	Common *acommon.Actions

	CreateTagCmd      *CreateTagCmd
	EditTagCmd        *EditTagCmd
	DeleteTagCmd      *DeleteTagCmd
	MoveTagToGroupCmd *MoveTagToGroupCmd

	ToggleFileTagCmd *ToggleFileTagCmd

	AssignedTags *AssignedTagActions `actions:"assigned-tags"`
	SubTags      *SubTagActions      `actions:"sub-tags"`
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common: commonActions,

		CreateTagCmd:      NewCreateTagCmd(infra, actions),
		EditTagCmd:        NewEditTagCmd(infra, actions),
		DeleteTagCmd:      NewDeleteTagCmd(infra, actions),
		MoveTagToGroupCmd: NewMoveTagToGroupCmd(infra, actions),

		ToggleFileTagCmd: NewToggleFileTagCmd(infra, actions),

		AssignedTags: &AssignedTagActions{
			List:                  NewListAssignedTagsPartial(infra, actions),
			ListItem:              NewListItemAssignedTagsPartial(infra, actions),
			CreateAndAssignTagCmd: NewCreateAndAssignTagCmd(infra, actions),
			Edit:                  NewEditAssignedTagsPartial(infra, actions),
			EditListItem:          NewEditAssignedTagsItemPartial(infra, actions),
			Count:                 NewCountAssignedTagsPartial(infra, actions),
			AssignTagCmd:          NewAssignTagCmd(infra, actions),
			UnassignTagCmd:        NewUnassignTagCmd(infra, actions),
		},

		SubTags: &SubTagActions{
			List:              NewListSubTagsPartial(infra, actions),
			Edit:              NewEditSubTagsPartial(infra, actions),
			AssignSubTagCmd:   NewAssignSubTagCmd(infra, actions),
			UnassignSubTagCmd: NewUnassignSubTagCmd(infra, actions),
		},
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.TaggingActionsRoute() + path
}
