package widget

import (
	"strings"

	"github.com/google/uuid"
)

type ListItemType int

const (
	ListItemTypeDefault ListItemType = iota
	ListItemTypeHelper
	// ListItemTypeProgressIndicator // TODO not styled according to spec yet
	// ListItemTypeRadio
)

type ListItem struct {
	Widget[ListItem]
	HTMXAttrs

	// TODO could this be implicit by analyzing Trailing?
	//
	// means Trailing is excluded
	// HTMXAttrsContentOnly HTMXAttrs

	// from ListItemContent
	Leading        IWidget
	Headline       IWidget // can also be a link
	SupportingText *Text

	// Icon           string // TODO or Text or Icon?
	// EndIcon        string // TODO or Text or Icon?
	// Content IWidget // usually Link or ListItemContent
	// Leading        IWidget
	// Headline       IWidget // can also be a link
	// SupportingText Text
	Trailing IWidget

	IsSelected bool

	// not implicit when Child or Children exist because
	// they may be set later on demand
	// TODO is this a good strategy?
	// TODO find a better name, IsHeader or IsCollapsible?
	IsCollapsible bool
	IsOpen        bool
	Child         IWidget

	ContextMenu *Menu
	IsDisabled  bool // TODO implement

	// TODO just temporary for neobrutalism experiment
	BackgroundColor string

	Type           ListItemType
	RadioGroupName string // not all chars are allow, UUIDs not working
	RadioValue     string

	// PopoverTarget string

	// HxGet     string
	// HxTarget  string
	// HxSwap    string
	// HxPushURL string
	// HxSelect  string
	// HxPost   string
	// HxVals   template.JS
	// HxTarget string
	// HxSwap   string
}

func (qq *ListItem) GetID() string {
	if qq.ID == "" {
		qq.ID = "listItem-" + uuid.NewString()
	}
	return qq.ID
}

// TODO just temporary for neobrutalism experiment
func (qq *ListItem) GetBackgroundColor() string {
	if qq.BackgroundColor != "" {
		return qq.BackgroundColor
	}
	return "var(--surface)"
}

func (qq *ListItem) IsTypeHelper() bool {
	return qq.Type == ListItemTypeHelper
}

func (qq *ListItem) IsTypeRadio() bool {
	return qq.RadioGroupName != ""
	// return qq.Type == ListItemTypeRadio
}

func (qq *ListItem) GetClass() string {
	classes := []string{}

	if qq.SupportingText != nil {
		// classes = append(classes, "h-16")
	}

	return strings.Join(classes, " ")
}

func (qq *ListItem) GetTrailing() IWidget {
	if checkbox, isCheckbox := qq.Trailing.(*Checkbox); isCheckbox {
		checkbox.SetIsStateInherited(true)
	}
	return qq.Trailing
}

/*
func (qq *ListItem) IsCollapsible() bool {
	return len(qq.Children) > 0
}
*/

/*
func (qq *ListItem) GetContent() IWidget {
	link, ok := qq.Content.(*Link)
	if ok {
		// TODO make sure that done just once...
		// TODO is beercss specific...
		link.Classes += "wave flex row padding max"
		return qq.Content
	}

	return &Container{
		Classes: []string{
			"wave", "flex", "padding", "max",
		},
		Child: qq.Content,
	}
}

*/
