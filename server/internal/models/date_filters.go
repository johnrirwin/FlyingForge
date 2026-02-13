package models

import (
	"strings"
	"time"
)

// ParseDateFilter parses a date filter value commonly used by the API.
// Supported formats:
// - YYYY-MM-DD
// - MM/DD/YYYY
func ParseDateFilter(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, true
	}
	if t, err := time.Parse("01/02/2006", value); err == nil {
		return t, true
	}

	return time.Time{}, false
}
