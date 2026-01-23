package widget

import (
	"html/template"
	"strings"

	"github.com/simpledms/simpledms/util/actionx"
)

type HxOn struct {
	Event   string
	Handler template.JS
}

// cannot be rendered via `render` function because of security checks,
// but works fine with `template "HTMXAttrs" .HTMXAttrs`
// TODO is there a better solution for this without introducing security risk?
type HTMXAttrs struct {
	HxPushURL    string
	HxReplaceURL string
	HxTrigger    string
	HxPost       string
	HxGet        string
	HxVals       template.JS
	HxTarget     string
	HxSelect     string
	HxSwap       string
	HxConfirm    string
	HxInclude    string
	HxBoost      string // for example used for file download links to disable boosting
	HxHeaders    template.JS
	HxOn         *HxOn
	HxIndicator  string

	// Action        Actionable
	LoadInPopover bool
	// PopoverTargetID string // not in use as of 24.11.2024
	DialogID string
	// LoadInDialog  bool
}

/*
type Actionable interface {
	EndpointWithParams(wrapper actionx.ResponseWrapper, hxTarget string)
	Data()
}
*/

// used to apply `role=""` or cursor-pointer class
func (qq HTMXAttrs) IsLink() bool {
	// TODO how to handle DialogID? Should usually be used on a button, thus probably nothing to do, but
	//		if it also gets used on divs and other elements, we should consider refactoring IsLink to
	//		GetRole and return 'button' for dialogs

	isLink := (qq.GetHxPost() != "" || qq.GetHxGet() != "") && (qq.GetHxTrigger() == "" || strings.Contains(qq.GetHxTrigger(), "click"))
	// ignore PopoverTargetID because it only works on buttons and inputs, not on divs...
	return isLink
}

// used to create a real `a` element, in some cases a child
// TODO is this a good name, or would IsRoute be better?
func (qq HTMXAttrs) IsPageLink() bool {
	return qq.GetHxGet() != "" && (qq.GetHxTrigger() == "" || strings.Contains(qq.GetHxTrigger(), "click"))
}

func (qq HTMXAttrs) GetHxGet() string {
	return qq.HxGet
}

func (qq HTMXAttrs) GetHxBoost() string {
	return qq.HxBoost
}

func (qq HTMXAttrs) GetHxPost() string {
	// TODO parse URL and set wrapper or refactor HxPost from string to url.URL?
	// TODO suffix -dialog check isn't nice...
	if (qq.LoadInPopover /*|| qq.LoadInDialog*/) &&
		qq.HxPost != "" &&
		!strings.Contains(qq.HxPost, "wrapper=") &&
		// -dialog-partial shouldn't be neccessary because dialog don't have partial
		// suffix usually, but just for safety
		!(strings.HasSuffix(qq.HxPost, "-dialog") || strings.HasSuffix(qq.HxPost, "-dialog-partial")) {
		wrapper := actionx.ResponseWrapperDialog.String()
		// if qq.LoadInDialog {
		// wrapper = actionx.ResponseWrapperDialog.String()
		// }
		return qq.HxPost + "&wrapper=" + wrapper
	}
	return qq.HxPost
}

func (qq HTMXAttrs) GetHxTrigger() string {
	return qq.HxTrigger
}

func (qq HTMXAttrs) GetHxTarget() string {
	if qq.LoadInPopover && qq.HxTarget == "" {
		return "#popovers"
	}
	// if qq.LoadInDialog && qq.HxTarget == "" {
	// return "#dialogs"
	// }
	if qq.HxTarget == "" {
		// return "#content"
	}
	return qq.HxTarget
}

func (qq HTMXAttrs) GetHxSelect() string {
	return qq.HxSelect
}

func (qq HTMXAttrs) GetHxVals() template.JS {
	return qq.HxVals
}

// TODO is it a good idea to always use morph?
//
//	it is expensive and often brings no advantage
func (qq HTMXAttrs) GetHxSwap() string {
	/*
		// breaks navigation and some boosted links
		if qq.GetHxTarget() == "" && qq.HxSwap == "" && !qq.IsLink() {
			// TODO PageLink or Link()?
			return "none"
		}
	*/

	if qq.LoadInPopover && qq.HxSwap == "" {
		return "afterbegin"
	}
	// may need more cases added; makes only sense with mechanism
	// that override existing HTML, afterbegin for example doesn't need
	// nor supports morphing (support may be fixed as of 2024.11.05)
	//
	// was == instead of prefix before 2024.12.18
	if strings.HasPrefix(qq.HxSwap, "innerHTML") || strings.HasPrefix(qq.HxSwap, "outerHTML") {
		return "morph:" + qq.HxSwap
	}

	// HxGet check is necessary to only use for boosted links
	if qq.HxSwap == "" && qq.HxGet != "" { // && qq.HxBoost != "false" {
		return "morph:innerHTML" // innerHTML is default htmx setting, outerHTML is default for morph
	}

	return qq.HxSwap
}

func (qq HTMXAttrs) GetHxPushURL() string {
	if qq.HxPushURL != "" {
		return qq.HxPushURL
	}
	return qq.HxGet
}

func (qq HTMXAttrs) GetHxReplaceURL() string {
	return qq.HxReplaceURL
}

func (qq HTMXAttrs) GetHxIndicator() string {
	return qq.HxIndicator
}

func (qq HTMXAttrs) SetHxHeaders(headers template.JS) HTMXAttrs {
	qq.HxHeaders = headers
	return qq
}
