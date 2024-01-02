package output

import "time"

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
