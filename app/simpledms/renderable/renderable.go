package renderable

import (
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
)

type Renderable interface {
	TemplateName() string
	SetContext(ctx ctxx.Context)
	GetContext() ctxx.Context
	IsOOB() bool
	// Render(writer io.Writer, templates *template.Template) error
}
