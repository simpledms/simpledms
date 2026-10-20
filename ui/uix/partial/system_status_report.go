package partial

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/model/main/systemstatus"
)

// SystemStatusReport shows a summary of all problems first and the settings of each component
// below. A dedicated template is used because the settings are dense label-value pairs that
// read better as a table than as two-line list items.
type SystemStatusReport struct {
	widget.Widget[SystemStatusReport]
	Status    *systemstatus.SystemStatus
	CheckedAt string
}

// TemplateName keeps application components independent of the core widget package name.
func (qq *SystemStatusReport) TemplateName() string {
	return "SystemStatusReport"
}

// Name supplies the same template name to the nested render template function.
func (qq *SystemStatusReport) Name() string {
	return qq.TemplateName()
}

func (qq *SystemStatusReport) Headline() *widget.Text {
	switch problemCount := qq.Status.ProblemCount(); problemCount {
	case 0:
		return widget.T("No problems found")
	case 1:
		return widget.T("1 problem needs attention")
	default:
		return widget.Tf("%d problems need attention", problemCount)
	}
}

// ProblemLevels orders the problems in the summary, most severe first.
func (qq *SystemStatusReport) ProblemLevels() []systemstatus.StatusLevel {
	return []systemstatus.StatusLevel{
		systemstatus.StatusLevelError,
		systemstatus.StatusLevelWarning,
	}
}

func (qq *SystemStatusReport) SupportingText() *widget.Text {
	return widget.Tf(
		"Last checked: %s. Configuration changes take effect after a restart.",
		qq.CheckedAt,
	)
}

// SummaryClass reserves the error container for errors. The theme containers are strongly
// colored, and Material 3 has no warning role, thus warnings use a neutral surface.
func (qq *SystemStatusReport) SummaryClass() string {
	if qq.Status.Level() == systemstatus.StatusLevelError {
		return "bg-error-container text-on-error-container"
	}
	return "bg-surface-container-high text-on-surface"
}

func (qq *SystemStatusReport) SummaryIcon() *widget.Icon {
	level := qq.Status.Level()
	if !level.IsProblem() {
		// not configured optional components are not a problem
		level = systemstatus.StatusLevelOK
	}
	icon := qq.LevelIcon(level)
	icon.Size = widget.IconSizeLarge
	if level == systemstatus.StatusLevelOK {
		icon.Color(widget.ColorPrimary)
	}
	return icon
}

func (qq *SystemStatusReport) LevelClass(level systemstatus.StatusLevel) string {
	switch level {
	case systemstatus.StatusLevelError:
		return "bg-error-container text-on-error-container"
	case systemstatus.StatusLevelWarning:
		return "border border-outline text-on-surface"
	case systemstatus.StatusLevelDisabled:
		return "bg-surface-container-highest text-on-surface-variant"
	default:
		return "border border-outline-variant text-primary"
	}
}

func (qq *SystemStatusReport) LevelLabel(level systemstatus.StatusLevel) *widget.Text {
	switch level {
	case systemstatus.StatusLevelError:
		return widget.T("Error")
	case systemstatus.StatusLevelWarning:
		return widget.T("Warning")
	case systemstatus.StatusLevelDisabled:
		return widget.T("Not configured")
	default:
		return widget.T("OK")
	}
}

func (qq *SystemStatusReport) LevelIcon(level systemstatus.StatusLevel) *widget.Icon {
	name := "check_circle"
	switch level {
	case systemstatus.StatusLevelError:
		name = "error"
	case systemstatus.StatusLevelWarning:
		name = "warning"
	case systemstatus.StatusLevelDisabled:
		name = "block"
	}
	return &widget.Icon{
		Name: name,
		Size: widget.IconSizeSmall,
	}
}
