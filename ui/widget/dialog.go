package widget

import (
	"log"
	"strings"
)

// TODO DialogType or DialogLayout?
type DialogLayout int

const (
	DialogLayoutDefault DialogLayout = iota
	DialogLayoutStable
	DialogLayoutSideSheet
)

type Dialog struct {
	Widget[Dialog]
	HTMXAttrs

	// Mobile version is always fullscreen
	Layout DialogLayout

	Headline    *Text
	SubmitLabel *Text // TODO name?? PrimaryActionLabel??
	FormID      string
	// CloseLabel  string

	IsOpenOnLoad                    bool
	IsOpenOnLoadOnExtraLargeScreens bool
	// Deprecated: should no longer be necessary with URL state
	KeepInDOMOnClose bool

	submitButton    *Button
	closeIconButton *IconButton
	closeButton     *Button

	// ContentID string // TODO or InnerID?
	Child IWidget
}

func (qq *Dialog) GetCloseIconButton() *IconButton {
	if qq.closeIconButton == nil {
		qq.closeIconButton = &IconButton{
			Icon: "close",
			// PopoverTarget:       qq.ID,
			// PopoverTargetAction: "hide",
		}
	}
	return qq.closeIconButton
}
func (qq *Dialog) GetCloseIconButtonID() string {
	return qq.GetCloseIconButton().GetID()
}

func (qq *Dialog) GetCloseButton() *Button {
	if qq.closeButton == nil {
		qq.closeButton = &Button{
			Label: T("Close"),
		}
	}
	return qq.closeButton
}
func (qq *Dialog) GetCloseButtonID() string {
	return qq.GetCloseButton().GetID()
}

func (qq *Dialog) GetSubmitButton() *Button {
	if qq.SubmitLabel == nil {
		return nil
	}
	if qq.submitButton == nil {
		if qq.FormID == "" {
			form, isForm := qq.Child.(*Form)
			if isForm {
				qq.FormID = form.GetID()
			}
		}

		if qq.FormID == "" {
			log.Println("no form found")
		}

		qq.submitButton = &Button{
			Type:      "submit",
			Label:     qq.SubmitLabel,
			FormID:    qq.FormID,
			StyleType: ButtonStyleTypeTonal, // TODO okay? also on mobile?
		}
	}

	return qq.submitButton
}

// TODO nameing
func (qq *Dialog) IsStableLayout() bool {
	return qq.Layout == DialogLayoutStable
}
func (qq *Dialog) IsSideSheetLayout() bool {
	return qq.Layout == DialogLayoutSideSheet
}
func (qq *Dialog) IsDefaultLayout() bool {
	return qq.Layout == DialogLayoutDefault
}

func (qq *Dialog) GetClass() string {
	var classes []string

	return strings.Join(classes, " ")
}
