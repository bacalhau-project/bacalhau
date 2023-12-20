package template

type Parser interface {
	// Parse parses the template and replaces the placeholders with the values a replacements map.
	Parse(content string) (string, error)
	// ParseBytes parses the template and replaces the placeholders with the values a replacements map.
	ParseBytes(content []byte) ([]byte, error)
}
