package staticpage

import (
	"html/template"
	"log"
	"net/http"
	"sync"

	acommon "github.com/simpledms/simpledms/core/action/common"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
)

type StaticPage struct {
	acommon.Page

	infra    *common.Infra
	actions  *Actions
	renderer *MarkdownRenderer

	renderedBySlug map[string]template.HTML
	rwMutex        sync.RWMutex
}

func NewStaticPage(infra *common.Infra, actions *Actions) *StaticPage {
	page := &StaticPage{
		infra:          infra,
		actions:        actions,
		renderer:       NewMarkdownRenderer(contentFS),
		renderedBySlug: make(map[string]template.HTML),
	}

	for _, definition := range staticPageDefinitions {
		content, err := page.renderer.Render(definition.MarkdownPath)
		if err != nil {
			log.Printf("static page markdown render failed for %s: %v", definition.Slug, err)
			continue
		}

		page.renderedBySlug[definition.Slug] = content
	}

	return page
}

func (qq *StaticPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	slug := req.PathValue("slug")
	definition, hasDefinition := StaticPageDefinitionBySlug(slug)
	if !hasDefinition {
		return e.NewHTTPErrorf(http.StatusNotFound, "The requested page was not found.")
	}

	htmlContent, err := qq.pageContent(definition)
	if err != nil {
		log.Printf(
			"static page content load failed for %s: %v",
			definition.Slug,
			err,
		)
		return e.NewHTTPErrorf(
			http.StatusInternalServerError,
			"The requested page could not be loaded.",
		)
	}

	return qq.Render(
		rw,
		req,
		ctx,
		qq.infra,
		widget.T(definition.PageTitle).String(ctx),
		qq.Widget(ctx, definition, htmlContent),
	)
}

func (qq *StaticPage) Widget(ctx ctxx.Context, definition StaticPageDefinition, htmlContent template.HTML) renderable.Renderable {
	var fabs []*widget.FloatingActionButton

	mainLayout := &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, definition.Slug, fabs),
		Content: &widget.DefaultLayout{
			AppBar: qq.appBar(ctx, definition),
			Content: &widget.Column{
				GapYSize:         widget.Gap4,
				NoOverflowHidden: true,
				Children: []widget.IWidget{
					&widget.MarkdownContent{HTML: htmlContent},
				},
			},
		},
	}

	return mainLayout
}

func (qq *StaticPage) appBar(ctx ctxx.Context, definition StaticPageDefinition) *widget.AppBar {
	iconName := definition.IconName
	if iconName == "" {
		iconName = "description"
	}

	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: iconName,
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T(definition.AppBarTitle),
		},
		Actions: []widget.IWidget{},
	}
}

func (qq *StaticPage) pageContent(definition StaticPageDefinition) (template.HTML, error) {
	qq.rwMutex.RLock()
	content, hasContent := qq.renderedBySlug[definition.Slug]
	qq.rwMutex.RUnlock()
	if hasContent {
		return content, nil
	}

	rendered, err := qq.renderer.Render(definition.MarkdownPath)
	if err != nil {
		return "", err
	}

	content = rendered

	qq.rwMutex.Lock()
	qq.renderedBySlug[definition.Slug] = content
	qq.rwMutex.Unlock()

	return content, nil
}
