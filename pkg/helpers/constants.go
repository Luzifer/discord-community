// Package helpers contains shared helper functions and default values.
package helpers

import (
	"time"
)

var (
	// DefaultStreamScheduleEntries defines how many entries to show by default
	DefaultStreamScheduleEntries = new(int64(5))

	// DefaultStreamSchedulePastTime defines how long after the stream to keep it
	DefaultStreamSchedulePastTime = new(15 * time.Minute)

	// StreamScheduleDefaultColor defines the default color for the schedule
	StreamScheduleDefaultColor = new(int64(0x2ECC71))
)
