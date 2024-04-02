package template

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type ParserParams struct {
	Replacements map[string]string
	EnvPattern   string
}

type DefaultParser struct {
	replacements map[string]string
	template     *template.Template
}

func NewParser(params ParserParams) (*DefaultParser, error) {
	if params.Replacements == nil {
		params.Replacements = make(map[string]string)
	}
	if params.EnvPattern != "" {
		// If the pattern is "*", we want to match all environment variables
		if params.EnvPattern == "*" {
			params.EnvPattern = ".*"
		}
		pattern, err := regexp.Compile(params.EnvPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile environment variable pattern: %w", err)
		}

		for _, envVar := range os.Environ() {
			parts := strings.SplitN(envVar, "=", 2)
			if pattern != nil && pattern.MatchString(parts[0]) {
				if _, ok := params.Replacements[parts[0]]; !ok {
					params.Replacements[parts[0]] = parts[1]
				}
			}
		}
	}
	return &DefaultParser{
		replacements: params.Replacements,
		template:     template.New("").Option("missingkey=error"),
	}, nil
}

func (p *DefaultParser) parse(content string) (*bytes.Buffer, error) {
	tmpl, err := p.template.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	tpl := new(bytes.Buffer)
	if err := tmpl.Execute(tpl, p.replacements); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return tpl, nil
}

// Parse parses the template and replaces the placeholders with the values from the replacements map.
func (p *DefaultParser) Parse(content string) (string, error) {
	tpl, err := p.parse(content)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

// ParseBytes parses the template and replaces the placeholders with the values from the replacements map.
func (p *DefaultParser) ParseBytes(content []byte) ([]byte, error) {
	tpl, err := p.parse(string(content))
	if err != nil {
		return nil, err
	}
	return tpl.Bytes(), nil
}

// compile time check if the interface is implemented
var _ Parser = &DefaultParser{}
