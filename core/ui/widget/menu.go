package widget

type Position int

const (
	PositionLeft Position = iota
	PositionRight
	PositionTop
	PositionBottom
)

type Menu struct {
	Widget[Menu]
	Position           Position
	Items              []*MenuItem
	EmptyLabel         *Text
	MatchesAnchorWidth bool
	IsAutoPopover      bool
}

// ContextMenuButton opens the same menu as the owning row or card's right-click gesture.
func (qq *Menu) ContextMenuButton() *IconButton {
	return &IconButton{
		Widget: Widget[IconButton]{
			ID: qq.GetID() + "-trigger",
		},
		Icon:          "more_vert",
		Label:         T("Actions"),
		Tooltip:       T("Actions"),
		PopoverTarget: qq.GetID(),
	}
}

// top
func (qq *Menu) GetInsetBlockStart() string {
	if qq.Position == PositionRight || qq.Position == PositionLeft {
		return "top"
	}
	if qq.Position == PositionBottom {
		return "bottom"
	}
	// TODO impl for bottom and top position
	return ""
}

// bottom
func (qq *Menu) GetInsetBlockEnd() string {
	return ""
}

// left
func (qq *Menu) GetInsetInlineStart() string {
	if qq.Position == PositionRight {
		return "right"
	}
	if qq.Position == PositionBottom {
		return "left"
	}
	// TODO impl for bottom and top position
	return ""
}

// right
func (qq *Menu) GetInsetInlineEnd() string {
	if qq.Position == PositionLeft {
		return "left"
	}
	// TODO impl for bottom and top position
	return ""
}

func (qq *Menu) IsPositionRight() bool {
	return qq.Position == PositionRight
}

func (qq *Menu) IsPositionLeft() bool {
	return qq.Position == PositionLeft
}
