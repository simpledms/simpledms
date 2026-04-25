package timex

import (
	"database/sql/driver"
	"time"
)

// only time without date (always 0001-01-01)
// TODO add timezone support?
type Time struct {
	time.Time
}

func NewTimeZero() Time {
	return Time{}
}

func NewTimeNow() Time {
	return Time{
		time.Now(),
	}
}

func NewTime(timex time.Time) Time {
	return Time{
		Time: time.Date(1, 1, 1, timex.Hour(), timex.Minute(), timex.Second(), timex.Nanosecond(), timex.Location()),
	}
}

func NewTimeFromString(str string) (Time, error) {
	layout := "15:04:05"

	if len(str) == 5 {
		layout = "15:04"
	}

	timex, err := time.Parse(layout, str)
	if err != nil {
		return Time{}, err
	}

	return NewTime(timex), nil
}

func NewTimeBeginOfDay() Time {
	return Time{
		time.Date(1, 1, 1, 0, 0, 0, 0, time.Local),
	}
}

// local is also used in time.Now()
func NewTimeEndOfDay() Time {
	return Time{
		time.Date(1, 1, 1, 23, 59, 59, 999999999, time.Local),
	}
}

// necessary because postgresql doesn't handle nanoseconds correctly
func NewTimeEndOfDayWithoutNanoSeconds() Time {
	return Time{
		time.Date(1, 1, 1, 23, 59, 59, 0, time.Local),
	}
}

// used by structs.Map() with value tag
func (qd Time) StructValue() interface{} {
	return qd.Time
}

// TODO identical for Date, DateTime and Time
func (qd *Time) Scan(value interface{}) error {
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
func (qd Time) Value() (driver.Value, error) {
	t := qd.Time // just for convenience
	tt := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	return tt, nil
}

func (qd Time) FormatForDatabaseQuery() string {
	return qd.Time.Format("15:04:05")
}
