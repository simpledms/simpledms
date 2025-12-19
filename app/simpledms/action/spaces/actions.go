package spaces

import (
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

type Actions struct {
	SpacesPage *SpacesPage // TODO SpacePage?
	SpaceCards *SpaceCards

	CreateSpace *CreateSpace
	EditSpace   *EditSpace
	DeleteSpace *DeleteSpace
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		SpacesPage: NewSpacesPage(infra, actions),
		SpaceCards: NewSpaceCards(infra, actions),

		CreateSpace: NewCreateSpace(infra, actions),
		EditSpace:   NewRenameSpace(infra, actions),
		DeleteSpace: NewDeleteSpace(infra, actions),
	}
	return actions
}

func (qq *Actions) Route(path string) string {
	return route.SpacesActionsRoute() + path
}
