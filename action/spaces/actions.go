package spaces

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	SpacesPage        *SpacesPage // TODO SpacePage?
	SpaceCardsPartial *SpaceCardsPartial

	CreateSpaceCmd    *CreateSpaceCmd
	CreateSpaceDialog *CreateSpaceDialog
	EditSpaceCmd      *EditSpaceCmd
	DeleteSpaceCmd    *DeleteSpaceCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		SpacesPage:        NewSpacesPage(infra, actions),
		SpaceCardsPartial: NewSpaceCardsPartial(infra, actions),

		CreateSpaceCmd:    NewCreateSpaceCmd(infra, actions),
		CreateSpaceDialog: NewCreateSpaceDialog(infra, actions),
		EditSpaceCmd:      NewRenameSpace(infra, actions),
		DeleteSpaceCmd:    NewDeleteSpaceCmd(infra, actions),
	}
	return actions
}

func (qq *Actions) Route(path string) string {
	return route.SpacesActionsRoute() + path
}
