package utils

import (
	"bytes"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

func ProcessYAMLTemplate(inputFilePath string, outputFilePath string, data map[string]interface{}) error {
	content, err := os.ReadFile(inputFilePath)
	if err != nil {
		return err
	}

	tmpl := template.New("yaml").Funcs(template.FuncMap{
		"toYaml":   toYAML,
		"fromYaml": fromYAML,
	})

	// ParseToDynamicJSON the template
	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return err
	}

	// Apply the template
	var processed bytes.Buffer
	if err := tmpl.Execute(&processed, data); err != nil {
		return err
	}

	// ParseToDynamicJSON the processed YAML to ensure it's still valid YAML
	var parsedYAML interface{}
	if err := yaml.Unmarshal(processed.Bytes(), &parsedYAML); err != nil {
		return err
	}

	// Marshal back to YAML
	processedYAML, err := yaml.Marshal(parsedYAML)
	if err != nil {
		return err
	}

	// Write the processed YAML to the output file
	err = os.WriteFile(outputFilePath, processedYAML, 0644)
	if err != nil {
		return err
	}

	return nil
}

// toYAML converts a value to YAML
func toYAML(v interface{}) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// fromYAML converts YAML to a Go value
func fromYAML(str string) interface{} {
	var out interface{}
	err := yaml.Unmarshal([]byte(str), &out)
	if err != nil {
		return nil
	}
	return out
}
