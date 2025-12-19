package uix

import (
	"html/template"

	"github.com/simpledms/simpledms/action"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type UI struct {
	actions   *action.Actions
	partials  []wx.IWidget // TODO or widgets?
	templates *template.Template
}
