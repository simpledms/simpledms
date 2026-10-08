package systemstatus

import (
	"github.com/simpledms/simpledms/core/ui/widget"
)

// Finding describes the result of one check, for example a failed connection or a setting
// that disables a feature.
type Finding struct {
	level   StatusLevel
	message *widget.Text
}

func NewFinding(level StatusLevel, message *widget.Text) *Finding {
	return &Finding{
		level:   level,
		message: message,
	}
}

func (qq *Finding) Level() StatusLevel {
	return qq.level
}

func (qq *Finding) Message() *widget.Text {
	return qq.message
}
