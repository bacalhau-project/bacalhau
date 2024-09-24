package output

import (
	"fmt"
	"strings"
	"time"
	"unicode"
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

func WrapSoftPreserveNewlines(s string, width int) string {
	lines := strings.Split(s, "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(wrapLine(line, width))
	}

	return result.String()
}

func wrapLine(line string, width int) string {
	var wrapped strings.Builder
	var current strings.Builder
	var styling strings.Builder
	inEscape := false
	lineLength := 0
	escapeStartRune := []rune(escapeStart)[0]
	escapeEndRune := []rune(escapeEnd)[0]

	for _, r := range line {
		if r == escapeStartRune {
			inEscape = true
			styling.WriteRune(r)
		} else if inEscape {
			styling.WriteRune(r)
			if r == escapeEndRune {
				inEscape = false
				current.WriteString(styling.String())
				if styling.String() == escapeStart+"0"+escapeEnd {
					styling.Reset()
				}
			}
		} else {
			current.WriteRune(r)
			lineLength++

			if lineLength >= width && unicode.IsSpace(r) {
				wrapped.WriteString(current.String())
				wrapped.WriteString("\n")
				current.Reset()
				current.WriteString(styling.String())
				lineLength = 0
			}
		}
	}

	wrapped.WriteString(current.String())
	return wrapped.String()
}
