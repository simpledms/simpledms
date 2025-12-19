package about

import (
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

type Actions struct {
	AboutPage *AboutPage
}

func NewActions(
	infra *common.Infra,
) *Actions {
	actions := new(Actions)
	*actions = Actions{
		AboutPage: NewAboutPage(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AboutActionsRoute() + path
}
