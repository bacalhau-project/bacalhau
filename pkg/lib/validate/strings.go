package validate

import "unicode"

func IsBlank(s string) bool {
	return len(s) == 0
}

func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

func ContainsSpaces(s string) bool {
	for _, c := range s {
		if unicode.IsSpace(c) {
			return true
		}
	}
	return false
}

func ContainsNull(s string) bool {
	for _, c := range s {
		if c == 0 {
			return true
		}
	}
	return false
}
