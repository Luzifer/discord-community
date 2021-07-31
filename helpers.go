package main

import "time"

var (
	ptrBoolFalse   = ptrBool(false)
	ptrBoolTrue    = ptrBool(true)
	ptrInt64Zero   = ptrInt64(0)
	ptrStringEmpty = ptrString("")
)

func ptrBool(v bool) *bool                       { return &v }
func ptrDuration(v time.Duration) *time.Duration { return &v }
func ptrInt64(v int64) *int64                    { return &v }
func ptrString(v string) *string                 { return &v }
func ptrTime(v time.Time) *time.Time             { return &v }
