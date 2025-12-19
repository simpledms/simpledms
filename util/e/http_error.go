package e

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	wx "github.com/simpledms/simpledms/ui/widget"
)

var (
	// TODO rename to Forbidden to match http status code for simplicity?
	ErrNotAllowed = NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access the requested resource.")
)

type HTTPError struct {
	err        error
	message    string
	snackbar   *wx.Snackbar
	statusCode int
}

// f suffix is necessary for IDE to highlight placeholders in message when function
// is called
func NewHTTPErrorf(statusCode int, messageStr string, args ...any) *HTTPError {
	if len(args) > 0 {
		messageStr = fmt.Sprintf(messageStr, args...)
	}

	// for gotext, see comment in widget.Text; gets translated in qq.Snackbar() call
	_ = message.NewPrinter(language.English).Sprintf(messageStr, args...)

	// message to lower, remove dots at end replace . with ,
	details := strings.ToLower(messageStr)
	details = strings.Trim(details, ".")
	details = strings.Replace(details, ".", ",", -1)

	return &HTTPError{
		err:        errors.New(details),
		message:    messageStr,
		statusCode: statusCode,
	}
}

func NewHTTPErrorWithSnackbar(statusCode int, snackbar *wx.Snackbar) *HTTPError {
	return &HTTPError{
		err:        errors.New(snackbar.SupportingText.StringUntranslated()),
		snackbar:   snackbar,
		statusCode: statusCode,
	}
}

func (qq *HTTPError) Snackbar() *wx.Snackbar {
	if qq.snackbar != nil {
		return qq.snackbar.SetIsError(true)
	}
	return wx.NewSnackbarf(qq.Message()).SetIsError(true)
}

func (qq *HTTPError) Error() string {
	return qq.err.Error()
}

func (qq *HTTPError) Message() string {
	if qq.message != "" {
		return qq.message
	}
	return http.StatusText(qq.statusCode)
}

func (qq *HTTPError) StatusCode() int {
	return qq.statusCode
}
