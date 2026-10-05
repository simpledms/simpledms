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
	// LazyItems loads the items with a query each time the menu opens, instead of rendering
	// Items with the owning view; only HxPost and HxVals are used. Long lists stay small, and
	// the items reflect the current state, also after the owning list was morphed.
	LazyItems *HTMXAttrs
}

func (qq *Menu) GetLazyItemsHTMXAttrs() HTMXAttrs {
	return HTMXAttrs{
		HxPost: qq.LazyItems.HxPost,
		HxVals: qq.LazyItems.HxVals,
		// toggle doesn't bubble, so listen on the popover itself
		HxTrigger: "toggle[newState=='open'] from:closest menu",
		HxTarget:  "this",
		HxSwap:    "innerHTML",
		HxSync:    "this:replace",
	}
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
		HasPopupMenu:  true,
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
