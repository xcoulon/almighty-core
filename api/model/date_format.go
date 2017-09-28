package model

import "time"

// format formats the given time to the RFC3339 format: "2006-01-02T15:04:05Z07:00"
func formatRFC3339(t time.Time) *string {
	result := t.Format(time.RFC3339)
	return &result
}
