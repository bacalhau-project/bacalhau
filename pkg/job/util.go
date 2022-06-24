package job

import (
	"fmt"
	"os"
	"regexp"
	"strings"
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

	file, _ := os.ReadFile("../../pkg/config/all_emojis.txt")
	emojiArray := strings.Split(string(file), "\n")
	emojiString := strings.Join(emojiArray, "|")

	r := regexp.MustCompile(fmt.Sprintf("[^%s|^%s]", emojiString, regexString))
	return r
}

