package timex

import (
	"time"
)

// unmodified time.Time
type DateTime struct {
	time.Time
}

func NewDateTime(time time.Time) DateTime {
	return DateTime{
		time,
	}
}

func NewDateTimeZero() DateTime {
	return DateTime{}
}

func (qd DateTime) String(languageBCP47 string) string {
	switch languageBCP47 {
	case "de":
		return qd.Format("02.01.2006, 15:04 Uhr")
	case "fr", "it":
		return qd.Format("02.01.2006, 15:04")
	}

	return qd.Format("2006-01-02, 15:04 o'clock")
}
