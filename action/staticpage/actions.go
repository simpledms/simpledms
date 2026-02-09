package staticpage

import "github.com/simpledms/simpledms/common"

type Actions struct {
	StaticPage *StaticPage
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)
	*actions = Actions{
		StaticPage: NewStaticPage(infra, actions),
	}

	return actions
}
