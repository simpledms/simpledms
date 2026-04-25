package web

import (
	"html/template"

	"github.com/simpledms/simpledms/action"
	wx "github.com/simpledms/simpledms/core/ui/widget"
)

type UI struct {
	actions   *action.Actions
	partials  []wx.IWidget // TODO or widgets?
	templates *template.Template
}
