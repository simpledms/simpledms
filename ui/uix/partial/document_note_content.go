package partial

import "github.com/simpledms/simpledms/core/ui/widget"

// DocumentNoteContent displays plain-text summaries or full note details and attribution.
type DocumentNoteContent struct {
	widget.Widget[DocumentNoteContent]
	Body                 string
	Metadata             []*widget.Text
	ReplacementHref      string
	ReplacementLabel     *widget.Text
	ReplacementReference *widget.Text
	ReplacementAttrs     widget.HTMXAttrs
	Title                *widget.Text
	IsSummary            bool
	Tooltip              *widget.Text
	Status               *widget.Text
}

// TemplateName keeps application components independent of the core widget package name.
func (qq *DocumentNoteContent) TemplateName() string {
	return "DocumentNoteContent"
}

// Name supplies the same template name to the nested render template function.
func (qq *DocumentNoteContent) Name() string {
	return qq.TemplateName()
}
