package widget

// Snackbars appear without warning, but they don’t block users from interacting with page content.
//
// Snackbars without actions can auto-dismiss after 4–10 seconds, depending on platform. Avoid
// using auto-dismissing snackbars on web unless there's also inline feedback.
//
// Snackbars with actions must remain on the screen until the user takes an action on the snackbar,
// or dismisses it.
//
// https://m3.material.io/components/snackbar/guidelines
//
// snackbars are usually rendered in place because they should appear above
// popover, which is not the case if they are not a child of the popover
//
// just one snackbar can be shown at the time
//
// TODO Icon support
type Snackbar struct {
	Widget[Snackbar]
	SupportingText *Text
	// Child   IWidget // TODO or Content?
	Action  *Link // TODO Link or IWidget?
	IsError bool
	// TODO Action
	// TODO error, primary, secondary, tertiary
	// 		in material guidelines it's always default?
	// Type    string
}

// f is necessary for IDE support
func NewSnackbarf(text string, a ...any) *Snackbar {
	return &Snackbar{
		SupportingText: Tf(text, a...),
	}
}

func (qq *Snackbar) IsOOB() bool {
	return true
}

func (qq *Snackbar) WithAction(action *Link) *Snackbar {
	qq.Action = action
	return qq
}

func (qq *Snackbar) GetAction() *Link {
	if qq.Action == nil {
		return nil
	}
	qq.Action.IsNoColor = true
	return qq.Action
}

func (qq *Snackbar) GetClass() string {
	return ""
	/*
		classes := []string{"snackbar", "active"}
		if qq.IsError {
			classes = append(classes, "error")
		}
		return strings.Join(classes, " ")
	*/
}

func (qq *Snackbar) GetAutoDismissTimeout() int64 {
	if qq.Action != nil {
		return 10000
	}
	// TODO calculate based on content length
	return 5000
}

func (qq *Snackbar) SetIsError(isError bool) *Snackbar {
	qq.IsError = isError
	return qq
}
