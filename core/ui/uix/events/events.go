package events

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/simpledms/simpledms/core/ui/widget"
)

type EventWithID string
type Event string

const AccountUpdated Event = "accountUpdated"

const AccountDeleted Event = "accountDeleted"

const DetailsClosed Event = "detailsClosed"

const InitialPasswordSet Event = "initialPasswordSet"

const TemporaryPasswordCleared Event = "temporaryPasswordCleared"

const PasswordChanged Event = "passwordChanged"

const SideSheetToggled Event = "sideSheetToggled" // used in JS

const CloseSideSheet Event = "closeSideSheet" // used in JS

const CloseDialog Event = "closeDialog" // used in JS, thus do not change

const CloseAllSnackbars Event = "closeAllSnackbars" // used in JS

const AppUnlocked Event = "appUnlocked"

const AppInitialized Event = "appInitialized"

const AppPassphraseChanged Event = "appPassphraseChanged"

const UserCreated Event = "userCreated"

const UserUpdated Event = "userUpdated"

const UserDeleted Event = "userDeleted"

const TenantCreated Event = "tenantCreated"

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
	// TODO impl more generic
	if qq == SideSheetToggled || qq == CloseSideSheet {
		// return fmt.Sprintf("%s from:#popovers", qq.String())
	}
	if qq == CloseDialog {
		// return fmt.Sprintf("%s from:#popovers target:.js-dialog", qq.String())
	}
	return fmt.Sprintf("%s from:body", qq.String())
}

// modifier can for example be delay:100ms
func (qq Event) HandlerWithModifier(modifier string) string {
	// TODO impl more generic
	if qq == SideSheetToggled || qq == CloseSideSheet {
		// return fmt.Sprintf("%s from:body target:.js-side-sheet-dialog %s", qq.String(), modifier)
	}
	if qq == CloseDialog {
		// return fmt.Sprintf("%s from:body target:.js-dialog %s", qq.String(), modifier)
	}
	return fmt.Sprintf("%s from:body %s", qq.String(), modifier)
}

func (qq Event) HxOn(event string) *widget.HxOn {
	return &widget.HxOn{
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
func (qq Event) HxOnWithQueryParam(event string, paramName string) *widget.HxOn {
	return &widget.HxOn{
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
func (qq Event) UnsafeHxOnWithQueryParamAndValue(event string, paramName string, paramValue string) *widget.HxOn {
	return &widget.HxOn{
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
func UnsafeHxOnQueryParamAppendToSlice(event string, paramName string, paramValue string) *widget.HxOn {
	return &widget.HxOn{
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
func UnsafeHxOnQueryParamDeleteFromSlice(event string, paramName string, paramValue string) *widget.HxOn {
	return &widget.HxOn{
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
	return fmt.Sprintf("%s", qq.String(id))
}

const UploadLimitUpdated Event = "uploadLimitUpdated"
