package widget

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/simpledms/simpledms/ctxx"
)

type IWidget interface{}

// TODO find a betteRsolution
type IWidgetWithID interface {
	GetID() string
}

// TODO is it possible to not export this? would TemplateName still be accessible?
type Widget[T any] struct {
	// be careful, if ID is manually defined on a widget, for example ListItem, it has
	// to overwrite the GetID method
	ID      string
	context ctxx.Context
}

func (qq *Widget[T]) GetID() string {
	if qq.ID == "" {
		// prefix is necessary because querySelector is invalid if first char is a number, according
		// to HTML 4 spec, ids have to start with letter
		qq.ID = qq.TemplateName() + "-" + uuid.NewString()
	}
	return qq.ID
}

func (qq *Widget[T]) TemplateName() string {
	name := fmt.Sprintf("%T", new(T))
	name = strings.TrimPrefix(name, "*widget.")
	return name
}

// called in Renderer
func (qq *Widget[T]) SetContext(ctx ctxx.Context) {
	qq.context = ctx
}

func (qq *Widget[T]) GetContext() ctxx.Context {
	/*if qq.context == nil {
		panic("context not set in widget")
	}*/
	return qq.context
}

// out of band
func (qq *Widget[T]) IsOOB() bool {
	return false
}
