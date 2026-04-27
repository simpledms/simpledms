package timex

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

var TimeNow = time.Now // makes current time manipulate-able for testing

type Date struct {
	time.Time
}

func NewDate(timex time.Time) Date {
	dateWithNullifiedTime := time.Date(timex.Year(), timex.Month(), timex.Day(), 0, 0, 0, 0, timex.Location())
	return Date{
		dateWithNullifiedTime,
	}
}

func ParseDate(str string) (Date, error) {
	timex, err := time.Parse("2006-01-02", str)
	if err != nil {
		return Date{}, err
	}
	return NewDate(timex), nil
}

func NewDateToday() Date {
	return NewDate(TimeNow())
}

func (qd Date) MarshalJSON() ([]byte, error) {
	if qd.Time.IsZero() {
		return []byte("\"\""), nil
	}

	year, month, day := qd.Time.Date()
	return []byte(fmt.Sprintf("\"%04d%02d%02d\"", year, month, day)), nil
}

func (qd *Date) UnmarshalJSON(data []byte) error {
	dataStr := string(data)
	if dataStr == "\"\"" {
		return nil
	}

	dataStr = strings.Replace(dataStr, "-", "", -1) // remove dashs to get uniform format 20160102

	qt, err := time.Parse(`"`+"20060102"+`"`, dataStr)
	if err != nil {
		return err
	}

	qd.Time = qt
	return nil
}

// TODO identical for Date, DateTime and Time
func (qd *Date) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	qd.Time = value.(time.Time)

	// IsZero check because if we set a timezone for zero values, the next IsZero check wouldn't return true
	// IsZero needs a zero value in UTC to return true and if we set the timezone, but keep the time of day at 0 o'clock, it's not a zero value in UTC anymore
	// so be careful when working with zero values, AddDate might not work as expected as the timezone would still be UTC
	if !qd.Time.IsZero() && qd.Time.Location().String() == "" {
		// set local time zone because for example DATE type in db has no timezone and is handled as UTC when the data comes from the db
		// but we handle everything with Local time in this package

		// do not use qd.Time.In(time.Local) because then we would get the date with time 2 o'clock and not 0 o'clock as we like
		t := qd.Time // just for convenience
		qd.Time = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	}

	return nil
}

// TODO identical for Date, DateTime and Time
func (qd Date) Value() (driver.Value, error) {
	t := qd.Time // just for convenience
	tt := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	return tt, nil
}

// used by structs.Map() with value tag
func (qd Date) StructValue() interface{} {
	return qd.Time
}

func (qd *Date) String(languageBCP47 string) string {
	year, month, day := qd.Time.Date()
	switch languageBCP47 {
	case "de", "fr", "it":
		return fmt.Sprintf("%02d.%02d.%d", day, month, year)
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}
