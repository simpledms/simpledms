package ui

import (
	"html/template"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/util/httpx"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(templates *template.Template) *Renderer {
	return &Renderer{
		templates: templates,
	}
}

func (qq *Renderer) RenderX(rw httpx.ResponseWriter, ctx ctxx.Context, widgets ...renderable.Renderable) {
	err := qq.Render(rw, ctx, widgets...)
	if err != nil {
		panic(err)
	}
}

func (qq *Renderer) Render(rw httpx.ResponseWriter, ctx ctxx.Context, widgets ...renderable.Renderable) error {
	widgets = append(widgets, rw.Renderables()...)

	renderedOOBWidgetsOnly := true
	for _, widget := range widgets {
		if !widget.IsOOB() {
			renderedOOBWidgetsOnly = false
		}
	}
	if renderedOOBWidgetsOnly {
		// header must be set before any data is written, thus separate loop
		rw.Header().Set("HX-Reswap", "none")
	}

	for _, widget := range widgets {
		widget.SetContext(ctx)
		if ctx == nil {
			log.Printf("ctx is nil, widget was %T, %+v", widget, widget)
		}

		err := qq.templates.ExecuteTemplate(rw, widget.TemplateName(), widget)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return err
		}
	}

	return nil
}
