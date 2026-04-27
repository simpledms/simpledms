package widget

type MenuItem struct {
	Widget[MenuItem]
	HTMXAttrs
	// Link

	LeadingIcon  string
	Label        *Text // TODO Text or Label?
	TrailingIcon string
	TrailingText string

	// just for convienience, items on Menu can be of type []*MenuItem
	// TODO can also be implicit if everything else is empty
	IsDivider  bool
	IsDisabled bool

	// cannot be used together with HTMXAttrs
	DownloadLinkURL      string
	DownloadLinkFilename string

	RadioGroupName string // not all chars are allow, UUIDs not working
	RadioValue     string
	IsSelected     bool

	divider *Divider
}

func (qq *MenuItem) GetDivider() *Divider {
	if qq.divider == nil {
		qq.divider = &Divider{}
	}
	return qq.divider
}

func (qq *MenuItem) IsTypeRadio() bool {
	return qq.RadioGroupName != ""
}

func (qq *MenuItem) GetTrailingIcon() string {
	if qq.TrailingIcon != "" {
		return qq.TrailingIcon
	}
	// TODO not optimal if not applied if trailing icon is set...
	if qq.IsSelected {
		return "check"
	}
	return ""
}
