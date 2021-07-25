package main

import "time"

var (
	ptrInt64Zero   = ptrInt64(0)
	ptrStringEmpty = ptrString("")
)

func ptrDuration(v time.Duration) *time.Duration { return &v }
func ptrInt64(v int64) *int64                    { return &v }
func ptrString(v string) *string                 { return &v }
