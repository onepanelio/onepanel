package model

import "time"

type LogEntry struct {
	Timestamp time.Time
	Content   string
}
