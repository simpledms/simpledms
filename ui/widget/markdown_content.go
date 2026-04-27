package widget

import "html/template"

type MarkdownContent struct {
	Widget[MarkdownContent]
	HTML template.HTML
}
