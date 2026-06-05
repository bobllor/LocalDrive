package utils

import (
	"time"
)

// NowUTC retrieves the current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatTime formats the Time argument to a UTC string with the
// following format by default:
// 2006-01-02T15:04:05.000Z+-0700
func FormatTime(date time.Time) string {
	format := "2006-01-02T15:04:05.000Z-0700"
	return date.UTC().Format(format)
}
