package widget

// TODO State or Type?
type AssistChipState int

const (
	AssistChipStateDefault     AssistChipState = iota
	AssistChipStateHighlighted                 // TODO name...
)

type AssistChip struct {
	Widget[AssistChip]
	HTMXAttrs

	State    AssistChipState
	IsActive bool

	Label        *Text
	LeadingIcon  string
	TrailingIcon string

	Badge *Badge

	// be aware that it is neccesarry to set IsOpenOnLoad on the Dialog if you replace
	// the chip with HTMX; otherwise the site would get unresponsible because inert
	// is set on everything except the (closed) dialog
	// Dialog *Dialog // used when Dialog must always be rendered (for example tags filter form)
}

func (qq *AssistChip) GetBadge() *Badge {
	if qq.Badge == nil {
		return qq.Badge
	}
	if qq.State != AssistChipStateDefault {
		qq.Badge.IsInverse = true
	}
	return qq.Badge
}

func (qq *AssistChip) IsDefaultState() bool {
	return qq.State == AssistChipStateDefault
}
func (qq *AssistChip) IsHighlightedState() bool {
	return qq.State == AssistChipStateHighlighted
}
