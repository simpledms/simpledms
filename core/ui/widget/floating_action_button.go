package widget

type FABSize int
type FABType int

const (
	FABSizeNormal FABSize = iota // must be first because default
	FABSizeSmall
	FABSizeLarge
)

const (
	FABTypePrimary FABType = iota
	FABTypeSecondary
	FABTypeTertiary
)

type FloatingActionButton struct {
	Widget[FloatingActionButton]
	HTMXAttrs

	ID      string
	Tooltip *Text
	FABSize FABSize
	FABType FABType
	Icon    string
	Child   IWidget
}

func (qq *FloatingActionButton) GetTooltip() string {
	if qq.Tooltip != nil {
		return qq.textString(qq.Tooltip)
	}
	return qq.GetLabel()
}

func (qq *FloatingActionButton) GetLabel() string {
	if qq.Tooltip != nil {
		return qq.textString(qq.Tooltip)
	}
	return qq.childLabel(qq.Child)
}

func (qq *FloatingActionButton) childLabel(child IWidget) string {
	switch childx := child.(type) {
	case nil:
		return ""
	case *Text:
		return qq.textString(childx)
	case []IWidget:
		for _, item := range childx {
			label := qq.childLabel(item)
			if label != "" {
				return label
			}
		}
	}
	return ""
}

func (qq *FloatingActionButton) textString(text *Text) string {
	ctx := qq.GetContext()
	if ctx == nil {
		return text.StringUntranslated()
	}
	return text.String(ctx)
}

func (qq *FloatingActionButton) SetSize(size FABSize) *FloatingActionButton {
	qq.FABSize = size
	return qq
}

func (qq *FloatingActionButton) SetType(typex FABType) *FloatingActionButton {
	qq.FABType = typex
	return qq
}

func (qq *FloatingActionButton) IsPrimary() bool {
	return qq.FABType == FABTypePrimary
}
func (qq *FloatingActionButton) IsSecondary() bool {
	return qq.FABType == FABTypeSecondary
}
func (qq *FloatingActionButton) IsTertiary() bool {
	return qq.FABType == FABTypeTertiary
}

func (qq *FloatingActionButton) IsSmall() bool {
	return qq.FABSize == FABSizeSmall
}
func (qq *FloatingActionButton) IsNormal() bool {
	return qq.FABSize == FABSizeNormal
}
func (qq *FloatingActionButton) IsLarge() bool {
	return qq.FABSize == FABSizeLarge
}
