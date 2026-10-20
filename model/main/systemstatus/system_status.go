package systemstatus

import (
	"time"
)

type SystemStatus struct {
	components []*ComponentStatus
	checkedAt  time.Time
}

func NewSystemStatus(components []*ComponentStatus, checkedAt time.Time) *SystemStatus {
	return &SystemStatus{
		components: components,
		checkedAt:  checkedAt,
	}
}

func (qq *SystemStatus) Components() []*ComponentStatus {
	return qq.components
}

func (qq *SystemStatus) CheckedAt() time.Time {
	return qq.checkedAt
}

func (qq *SystemStatus) Level() StatusLevel {
	level := StatusLevelOK
	for _, component := range qq.components {
		if component.Level() > level {
			level = component.Level()
		}
	}
	return level
}

func (qq *SystemStatus) ProblemCount() int {
	count := 0
	for _, component := range qq.components {
		count += component.ProblemCount()
	}
	return count
}
