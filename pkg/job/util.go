package job

import (
	"fmt"
	"regexp"
)


func SafeStringStripper(s string) string {
	rChars := SafeAnnotationRegex()
	return rChars.ReplaceAllString(s, "")
}

func IsSafeAnnotation(s string) bool {
	matches := SafeAnnotationRegex().FindIndex([]byte(s))
	return matches == nil
}

func SafeAnnotationRegex() *regexp.Regexp {
	regexString := "A-Za-z0-9._~!:@,;+-"

	r := regexp.MustCompile(fmt.Sprintf("[^%s|^%s]", returnAllEmojiString(), regexString))
	return r
}

