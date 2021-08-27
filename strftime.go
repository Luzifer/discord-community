package main

import (
	"strings"
	"time"

	"github.com/goodsign/monday"
)

var strftimeReplaces = []string{
	"%a", "Mon", // Weekday as locale’s abbreviated name.
	"%A", "Monday", // Weekday as locale’s full name.
	"%d", "02", // Day of the month as a zero-padded decimal number.
	"%b", "Jan", // Month as locale’s abbreviated name.
	"%B", "January", // Month as locale’s full name.
	"%m", "01", // Month as a zero-padded decimal number.
	"%y", "06", // Year without century as a zero-padded decimal number.
	"%Y", "2006", // Year with century as a decimal number.
	"%H", "15", // Hour (24-hour clock) as a zero-padded decimal number.
	"%I", "03", // Hour (12-hour clock) as a zero-padded decimal number.
	"%p", "PM", // Locale’s equivalent of either AM or PM.
	"%M", "04", // Minute as a zero-padded decimal number.
	"%S", "05", // Second as a zero-padded decimal number.
	"%f", "000000", // Microsecond as a decimal number, zero-padded on the left.
	"%z", "-0700", // UTC offset in the form ±HHMM[SS[.ffffff]] (empty string if the object is naive).
	"%Z", "MST", // Time zone name (empty string if the object is naive).
}

func localeStrftime(t time.Time, format, locale string) string {
	return monday.Format(
		t,
		strings.NewReplacer(strftimeReplaces...).Replace(format),
		monday.Locale(locale),
	)
}
