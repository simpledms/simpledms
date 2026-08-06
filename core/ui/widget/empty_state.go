package widget

type EmptyState struct {
	Widget[EmptyState]
	Icon        *Icon // TODO how to enforce `extra` size?
	Headline    *Text
	Description *Text // TODO must be paragraph...
	Actions     []IWidget
}

func (qq *EmptyState) GetIcon() *Icon {
	if qq.Icon == nil {
		return qq.Icon
	}
	qq.Icon.Size = IconSizeLarge
	return qq.Icon
}
