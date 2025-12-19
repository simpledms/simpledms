package event

import (
	"fmt"
	"html/template"
	"strings"

	wx "github.com/simpledms/simpledms/ui/widget"
)

type EventWithID string
type Event string

const (
	SuperTagUpdated EventWithID = "superTagUpdated"
	TagCreated      Event       = "tagCreated"
	TagUpdated      Event       = "tagUpdated"
	TagDeleted      Event       = "tagDeleted"
	// use TagUpdated for the moment
	// TagMovedToGroup    EventWithID = "tagMovedToGroup"

	AccountUpdated Event = "accountUpdated"

	FilterTagsChanged            Event = "filterTagsChanged"
	DocumentTypeFilterChanged    Event = "documentTypeFilterChanged"
	DetailsClosed                Event = "detailsClosed"
	DocumentTypeCreated          Event = "documentTypeCreated"
	DocumentTypeDeleted          Event = "documentTypeDeleted"
	DocumentTypeUpdated          Event = "documentTypeUpdated"
	DocumentTypeAttributeCreated Event = "documentTypeAttributeCreated" // TODO own namespace / package?
	DocumentTypeAttributeDeleted Event = "documentTypeAttributeDeleted"
	DocumentTypeAttributeUpdated Event = "documentTypeAttributeUpdated"
	FolderModeToggled            Event = "folderModeToggled"
	SearchQueryUpdated           Event = "searchQueryUpdated" // TODO more generic for all inputs?

	PropertyFilterChanged Event = "propertyFilterChanged"
	PropertyCreated       Event = "propertyCreated"
	PropertyUpdated       Event = "propertyUpdated"
	PropertyDeleted       Event = "propertyDeleted"

	FilePropertyUpdated Event = "filePropertyUpdated"

	FileUpdated        Event = "fileUpdated"
	FileDeleted        Event = "fileDeleted"
	FileUploaded       Event = "fileUploaded" // used in JS, FileUpload widget
	ZIPArchiveUnzipped Event = "zipArchiveUnzipped"

	SortByUpdated Event = "sortByUpdated"

	SpaceCreated Event = "spaceCreated"
	SpaceUpdated Event = "spaceUpdated"
	SpaceDeleted Event = "spaceDeleted"

	InitialPasswordSet       Event = "initialPasswordSet"
	TemporaryPasswordCleared Event = "temporaryPasswordCleared"
	PasswordChanged          Event = "passwordChanged"

	SideSheetToggled Event = "sideSheetToggled" // used in JS
	CloseSideSheet   Event = "closeSideSheet"   // used in JS
	// TODO separate Command type or implicit via name?
	CloseDialog Event = "closeDialog" // used in JS, thus do not change
	// CollapseListItem Event = "collapseListItem"
	CloseAllSnackbars Event = "closeAllSnackbars" // used in JS

	AppUnlocked          Event = "appUnlocked"
	AppInitialized       Event = "appInitialized"
	AppPassphraseChanged Event = "appPassphraseChanged"

	UserCreated Event = "userCreated"
	UserUpdated Event = "userUpdated"
	UserDeleted Event = "userDeleted"

	UserAssignedToSpace     Event = "userAssignedToSpace"
	UserUnassignedFromSpace Event = "userUnassignedFromSpace"
)

// doesn't support modifiers at the moment...
func HxTrigger(eventx ...Event) string {
	handlers := make([]string, len(eventx))
	for qi, event := range eventx {
		handlers[qi] = event.Handler()
	}
	return strings.Join(handlers, ",")
}

func (qq Event) String() string {
	return string(qq)
}

func (qq Event) Handler() string {
	return fmt.Sprintf("%s from:body", qq.String())
}

// modifier can for example be delay:100ms
func (qq Event) HandlerWithModifier(modifier string) string {
	return fmt.Sprintf("%s from:body %s", qq.String(), modifier)
}

func (qq Event) HxOn(event string) *wx.HxOn {
	return &wx.HxOn{
		Event: event,
		Handler: template.JS(
			fmt.Sprintf(
				"this.dispatchEvent(new CustomEvent('%s', { bubbles: true }))",
				qq.String(),
			),
		),
	}
}

// this is safe to use as long as param value is not rendered in frontend; via Go template it
// is safe
func (qq Event) HxOnWithQueryParam(event string, paramName string) *wx.HxOn {
	return &wx.HxOn{
		Event: event,
		Handler: template.JS(
			fmt.Sprintf(
				"_setQueryParam(event, '%s'); this.dispatchEvent(new CustomEvent('%s', { bubbles: true }))",
				paramName,
				qq.String(),
			),
		),
	}
}

// IMPORTANT be careful with value because of XSS
func (qq Event) UnsafeHxOnWithQueryParamAndValue(event string, paramName string, paramValue string) *wx.HxOn {
	return &wx.HxOn{
		Event: event,
		Handler: template.JS(
			fmt.Sprintf(
				"_setQueryParam(event, '%s', '%s'); this.dispatchEvent(new CustomEvent('%s', { bubbles: true }))",
				paramName,
				paramValue,
				qq.String(),
			),
		),
	}
}

// IMPORTANT be careful with value because of XSS
func UnsafeHxOnQueryParamAppendToSlice(event string, paramName string, paramValue string) *wx.HxOn {
	return &wx.HxOn{
		Event: event,
		Handler: template.JS(
			fmt.Sprintf(
				"_appendQueryParamSliceValue('%s', '%s');",
				paramName,
				paramValue,
			),
		),
	}
}

// IMPORTANT be careful with value because of XSS
func UnsafeHxOnQueryParamDeleteFromSlice(event string, paramName string, paramValue string) *wx.HxOn {
	return &wx.HxOn{
		Event: event,
		Handler: template.JS(
			fmt.Sprintf(
				"_deleteQueryParamSliceValue('%s', '%s');",
				paramName,
				paramValue,
			),
		),
	}
}

func (qq EventWithID) String(id int64) string {
	return fmt.Sprintf("%s-%d", qq, id)
	// return fmt.Sprintf("{'%s', %d}", qq, id)
}

func (qq EventWithID) Handler(id int64) string {
	return fmt.Sprintf("%s from:body", qq.String(id))
}

/*
type AssignedTagsUpdated struct {
	TagsCount int64 `json:"assignedTagsUpdated"`
}


func NewAssignedTagsUpdated(tagsCount int64) *AssignedTagsUpdated {
	return &AssignedTagsUpdated{TagsCount: tagsCount}
}

*/
