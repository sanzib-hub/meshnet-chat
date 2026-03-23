package utils

import (
	"fmt"
	"strings"
	"time"
)

// TruncateString truncates s to maxLen characters, appending "…" if trimmed.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

// FormatDuration returns a human-readable representation of d (e.g. "2h5m").
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute

	var b strings.Builder
	if h > 0 {
		b.WriteString(fmt.Sprintf("%dh", h))
	}
	b.WriteString(fmt.Sprintf("%dm", m))
	return b.String()
}

// Contains reports whether slice contains item.
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
