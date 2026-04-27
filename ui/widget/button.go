package widget

import (
	"strings"
)

type ButtonStyleType int

const (
	ButtonStyleTypeText ButtonStyleType = iota
	ButtonStyleTypeElevated
	ButtonStyleTypeFilled
	ButtonStyleTypeTonal
	ButtonStyleTypeOutlined
)

// TODO merge with icon button?
type Button struct {
	Widget[Button]
	HTMXAttrs

	Label *Text
	Icon  *Icon
	Badge *Badge

	Type   string // TODO enum: submit, reset, button
	FormID string // form to submit if button is outside form

	IsSmall      bool // TODO enum: Small / Medium / Large / Extra
	IsResponsive bool
	// TODO how to handle transparent? no border, no fill
	// Fill string // TODO enum light, default,
	StyleType ButtonStyleType

	PopoverTarget       string
	PopoverTargetAction string
	ReplaceURL          string
}

func (qq *Button) GetType() string {
	if qq.Type == "" {
		return "button"
	}
	return qq.Type
}

func (qq *Button) Small() *Button {
	qq.IsSmall = true
	return qq
}

func (qq *Button) GetClass() string {
	var classes []string
	return strings.Join(classes, " ")
}

func (qq *Button) IsText() bool {
	return qq.StyleType == ButtonStyleTypeText
}

func (qq *Button) IsElevated() bool {
	return qq.StyleType == ButtonStyleTypeElevated
}

func (qq *Button) IsFilled() bool {
	return qq.StyleType == ButtonStyleTypeFilled
}

func (qq *Button) IsTonal() bool {
	return qq.StyleType == ButtonStyleTypeTonal
}

func (qq *Button) IsOutlined() bool {
	return qq.StyleType == ButtonStyleTypeOutlined
}
