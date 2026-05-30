package widget

import (
	"strings"
	"unicode"
)

type NavigationRailItem struct {
	Widget[NavigationRailItem]
	HTMXAttrs

	Key      string
	Href     string
	Label    string
	Icon     string
	Value    string
	IsActive bool
	Children []*NavigationRailItem

	HasSmallBadge bool
	BadgeLabel    string

	IsSubheader         bool
	IsDisabled          bool
	IsCollapsible       bool
	IsExpandedByDefault bool
}

func NewNavigationRailItemFromDestination(destination *NavigationDestination) *NavigationRailItem {
	if destination == nil {
		return nil
	}
	return &NavigationRailItem{
		HTMXAttrs: destination.HTMXAttrs,
		Key:       firstNonEmpty(destination.Value, destination.Href, destination.Label),
		Href:      destination.Href,
		Label:     destination.Label,
		Icon:      destination.Icon,
		Value:     destination.Value,
		IsActive:  destination.IsActive,
	}
}

func (qq *NavigationRailItem) IsGroup() bool {
	return len(qq.Children) > 0
}

func (qq *NavigationRailItem) HasActiveChild() bool {
	for _, child := range qq.Children {
		if child == nil {
			continue
		}
		if child.IsActive || child.HasActiveChild() {
			return true
		}
	}
	return false
}

func (qq *NavigationRailItem) IsActiveInTree() bool {
	return qq.IsActive || qq.HasActiveChild()
}

func (qq *NavigationRailItem) SetActiveValue(value string) bool {
	if qq == nil || value == "" || qq.IsSubheader {
		return false
	}
	qq.IsActive = qq.Value == value || qq.Key == value
	if qq.IsActive {
		for _, child := range qq.Children {
			child.ClearActive()
		}
		return true
	}
	for _, child := range qq.Children {
		if child.SetActiveValue(value) {
			return true
		}
	}
	return false
}

func (qq *NavigationRailItem) ClearActive() {
	if qq == nil {
		return
	}
	qq.IsActive = false
	for _, child := range qq.Children {
		child.ClearActive()
	}
}

func (qq *NavigationRailItem) IsCollapsedByDefault() bool {
	return qq.IsCollapsible && !qq.IsExpandedByDefault
}

func (qq *NavigationRailItem) GetDefaultAriaExpanded() string {
	if qq.IsCollapsedByDefault() {
		return "false"
	}
	return "true"
}

func (qq *NavigationRailItem) GetKey() string {
	return firstNonEmpty(qq.Key, qq.Value, qq.Href, qq.Label, qq.Icon, qq.GetID())
}

func (qq *NavigationRailItem) GetDOMID() string {
	key := qq.GetKey()
	var builder strings.Builder
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	return "navigation-rail-item-" + builder.String()
}

func (qq *NavigationRailItem) HasNavigation() bool {
	return !qq.IsDisabled && (qq.Href != "" || qq.HTMXAttrs.IsLink())
}

func (qq *NavigationRailItem) ShouldRenderAnchor() bool {
	return !qq.IsDisabled && qq.GetHref() != ""
}

func (qq *NavigationRailItem) GetAriaLabel() string {
	if qq.BadgeLabel == "" {
		return qq.Label
	}
	return qq.Label + ", " + qq.BadgeLabel
}

func (qq *NavigationRailItem) GetHref() string {
	if qq.Href != "" {
		return qq.Href
	}
	if qq.HTMXAttrs.IsPageLink() {
		return qq.HTMXAttrs.GetHxGet()
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
