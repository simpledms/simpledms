package systemstatus

import (
	"github.com/simpledms/simpledms/core/ui/widget"
)

// ComponentStatus groups the settings and findings of one part of the system, for example
// the storage or the mail server.
type ComponentStatus struct {
	title    *widget.Text
	settings []*Setting
	findings []*Finding
}

func NewComponentStatus(title *widget.Text) *ComponentStatus {
	return &ComponentStatus{
		title:    title,
		settings: []*Setting{},
		findings: []*Finding{},
	}
}

func (qq *ComponentStatus) AddSetting(label *widget.Text, value *widget.Text) {
	qq.settings = append(qq.settings, NewSetting(label, value))
}

func (qq *ComponentStatus) AddFinding(level StatusLevel, message *widget.Text) {
	qq.findings = append(qq.findings, NewFinding(level, message))
}

func (qq *ComponentStatus) Title() *widget.Text {
	return qq.title
}

func (qq *ComponentStatus) Settings() []*Setting {
	return qq.settings
}

func (qq *ComponentStatus) Findings() []*Finding {
	return qq.findings
}

// Level is the most severe level of all findings, or OK if there are none.
func (qq *ComponentStatus) Level() StatusLevel {
	level := StatusLevelOK
	for _, finding := range qq.findings {
		if finding.Level() > level {
			level = finding.Level()
		}
	}
	return level
}

func (qq *ComponentStatus) ProblemCount() int {
	count := 0
	for _, finding := range qq.findings {
		if finding.Level().IsProblem() {
			count++
		}
	}
	return count
}
