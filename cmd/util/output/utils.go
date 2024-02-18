package output

import (
	"fmt"
	"time"
)

const (
	second = 1
	minute = 60 * second
	hour   = 60 * minute
	day    = 24 * hour
)

func ShortenTime(formattedTime string, maxLen int) string {
	if len(formattedTime) > maxLen {
		t, err := time.Parse(time.DateTime, formattedTime)
		if err != nil {
			panic(err)
		}
		formattedTime = t.Format(time.TimeOnly)
	}

	return formattedTime
}

// Elapsed returns a human-readable string representing the time elapsed since t
// e.g. "3d" for 3 days, "2h" for 2 hours, "5m" for 5 minutes, "10s" for 10 seconds
func Elapsed(t time.Time) string {
	d := time.Since(t)
	totalSeconds := int(d.Seconds())

	days := totalSeconds / day
	hours := (totalSeconds % day) / hour
	minutes := (totalSeconds % hour) / minute
	seconds := totalSeconds % minute

	var result string
	if days > 0 {
		if hours > 0 {
			result = fmt.Sprintf("%dd%dh", days, hours)
		} else {
			result = fmt.Sprintf("%dd", days)
		}
	} else if hours > 0 {
		if minutes > 0 {
			result = fmt.Sprintf("%dh%dm", hours, minutes)
		} else {
			result = fmt.Sprintf("%dh", hours)
		}
	} else if minutes > 0 {
		if seconds > 0 {
			result = fmt.Sprintf("%dm%ds", minutes, seconds)
		} else {
			result = fmt.Sprintf("%dm", minutes)
		}
	} else {
		result = fmt.Sprintf("%ds", seconds)
	}

	return result + " ago"
}
