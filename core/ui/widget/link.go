package widget

type Link struct {
	Widget[Link]
	HTMXAttrs

	Href          string
	PopoverTarget string
	SubmitForm    bool // TODO name? // not used as of Aug 16 2024
	Child         IWidget
	IsResponsive  bool
	WrapAnywhere  bool

	IsNoColor bool

	Filename string

	CopyValue         string
	CopyTooltip       *Text
	CopiedMessage     *Text
	CopyFailedMessage *Text
	// IsDownload    bool
}

func (qq *Link) IsDownload() bool {
	return qq.Filename != ""
}

func (qq *Link) IsText() bool {
	_, ok := qq.Child.(*Text)
	return ok
}

func (qq *Link) IsCopyable() bool {
	return qq.CopyValue != ""
}

/*func NewLink(href string) Link {
	return Link{Href: href}
}
*/
