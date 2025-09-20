package helpers

import (
	"time"
)

var (
	// DefaultStreamScheduleEntries defines how many entries to show by default
	DefaultStreamScheduleEntries = Ptr(int64(5)) //nolint:mnd // This is already the "constant"

	// DefaultStreamSchedulePastTime defines how long after the stream to keep it
	DefaultStreamSchedulePastTime = Ptr(15 * time.Minute) //nolint:mnd // This is already the "constant"

	// StreamScheduleDefaultColor defines the default color for the schedule
	StreamScheduleDefaultColor = Ptr(int64(0x2ECC71)) //nolint:mnd // This is already the "constant"
)
