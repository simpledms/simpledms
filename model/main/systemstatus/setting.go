package systemstatus

import (
	"github.com/simpledms/simpledms/core/ui/widget"
)

// Setting is an effective configuration value. Secrets must only be shown as set or not set.
type Setting struct {
	label *widget.Text
	value *widget.Text
}

func NewSetting(label *widget.Text, value *widget.Text) *Setting {
	return &Setting{
		label: label,
		value: value,
	}
}

func (qq *Setting) Label() *widget.Text {
	return qq.label
}

func (qq *Setting) Value() *widget.Text {
	return qq.value
}
