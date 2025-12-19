package widget

type Link struct {
	Widget[Link]
	HTMXAttrs

	Href          string
	Classes       string
	PopoverTarget string
	SubmitForm    bool // TODO name? // not used as of Aug 16 2024
	Child         IWidget
	IsResponsive  bool

	IsNoColor bool

	Filename string
	// IsDownload    bool
}

func (qq *Link) IsDownload() bool {
	return qq.Filename != ""
}

func (qq *Link) IsText() bool {
	_, ok := qq.Child.(*Text)
	return ok
}

func (qq *Link) GetClass() string {
	return ""
	/*
		classes := strings.Split(qq.Classes, " ")
		if qq.IsResponsive {
			classes = append(classes, "responsive")
		}
		if _, ok := qq.Child.(Text); ok {
			classes = append(classes, "link")
		}
		return strings.Join(classes, " ")

	*/
}

/*func NewLink(href string) Link {
	return Link{Href: href}
}
*/
