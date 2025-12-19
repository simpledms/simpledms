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
