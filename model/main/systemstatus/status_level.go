package systemstatus

// StatusLevel is ordered by severity, so that the most severe level of a component wins.
type StatusLevel int

const (
	StatusLevelOK StatusLevel = iota
	// StatusLevelDisabled marks optional components that are not configured.
	StatusLevelDisabled
	StatusLevelWarning
	StatusLevelError
)

func (qq StatusLevel) IsProblem() bool {
	return qq == StatusLevelWarning || qq == StatusLevelError
}
