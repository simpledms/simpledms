package timex

import (
	"database/sql/driver"
	"strings"
	"time"
)

// unmodified time.Time
// TODO impl as combination of Date and Time?
type DateTime struct {
	// don't use timex.Time because it's just time without date
	time.Time
}

func NewDateTime(time time.Time) DateTime {
	return DateTime{
		time,
	}
}

func NewDateTimeFromString(str string) (DateTime, error) {
	timex, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return DateTime{}, err
	}
	return NewDateTime(timex), nil
}

func NewDateTimeZero() DateTime {
	return DateTime{}
}

func NewDateTimeNow() DateTime {
	return DateTime{
		time.Now(),
	}
}

// local is also used in time.Now()
func NewDateTimeMax() DateTime {
	return DateTime{
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.Local),
	}
}

// necessary because postgresql doesn't handle nanoseconds correctly
func NewDateTimeMaxWithoutNanoSeconds() DateTime {
	return DateTime{
		time.Date(9999, 12, 31, 23, 59, 59, 0, time.Local),
	}
}

func NewDateTimeEndOfDay(date Date) DateTime {
	return DateTime{
		time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location()),
	}
}

func NewDateTimeEndOfToday() DateTime {
	return NewDateTimeEndOfDay(NewDateToday())
}

func NewDateTimeBeginOfToday() DateTime {
	now := time.Now()
	return DateTime{
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
	}

}

func (qd DateTime) Date() Date {
	return NewDate(qd.Time)
}

func (qd DateTime) AfterOrEqual(datetime DateTime) bool {
	return qd.After(datetime.Time) || qd.Equal(datetime.Time)
}

func (qd DateTime) BeforeOrEqual(datetime DateTime) bool {
	return qd.Before(datetime.Time) || qd.Equal(datetime.Time)
}

// used by structs.Map() with value tag
func (qd DateTime) StructValue() interface{} {
	return qd.Time
}

// TODO identical for Date, DateTime and Time
func (qd *DateTime) Scan(value interface{}) error {
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
func (qd DateTime) Value() (driver.Value, error) {
	t := qd.Time // just for convenience
	tt := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	return tt, nil
}

func (qd DateTime) FormatForDatabaseQuery(usedInNamedQuery bool) string {
	str := qd.Time.Format(time.RFC3339)
	if usedInNamedQuery {
		// in named queries, argument names are prefixed with : and there is currently no way
		// to enable sensitive parsing where : in quotes are handled literally
		// see https://github.com/jmoiron/sqlx/pull/85 for details
		//
		// sqlx allows escaping with :: in named queries, but non named queries
		// would fail with ::
		// see https://github.com/jmoiron/sqlx/issues/91 for details
		return strings.ReplaceAll(str, ":", "::")
	}
	return str
}

func (qd DateTime) String(languageBCP47 string) string {
	switch languageBCP47 {
	case "de":
		return qd.Time.Format("02.01.2006, 15:04 Uhr")
	case "fr", "it":
		return qd.Time.Format("02.01.2006, 15:04")
	}

	return qd.Time.Format("2006-01-02, 15:04 o'clock")
}
