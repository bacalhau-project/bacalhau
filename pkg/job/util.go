package job

import (
	"context"
	"fmt"
	"regexp"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

const RegexString = "A-Za-z0-9._~!:@,;+-"

func SafeStringStripper(s string) string {
	rChars := SafeAnnotationRegex()
	return rChars.ReplaceAllString(s, "")
}

func IsSafeAnnotation(s string) bool {
	matches := SafeAnnotationRegex().FindString(s)
	return matches == ""
}

func SafeAnnotationRegex() *regexp.Regexp {
	r := regexp.MustCompile(fmt.Sprintf("[^%s|^%s]", returnAllEmojiString(), RegexString))
	return r
}

func NewNoopJobLoader() JobLoader {
	jobLoader := func(ctx context.Context, id string) (executor.Job, error) {
		return executor.Job{}, nil
	}
	return jobLoader
}

func NewNoopStateLoader() StateLoader {
	stateLoader := func(ctx context.Context, id string) (executor.JobState, error) {
		return executor.JobState{}, nil
	}
	return stateLoader
}
