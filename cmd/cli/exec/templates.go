package exec

import (
	"fmt"
	"strings"
)

var templateMap map[string]string = nil

func init() {
	templateMap = make(map[string]string)
	templateMap["python"] = pythonTemplate()
	templateMap["duckdb"] = duckdbTemplate()
}

func ErrUnknownTemplate(name string) error {
	return fmt.Errorf("unknown template specified: %s", name)
}

// Template returns a string template (with yaml format) for the specified
// job type. Initially these templates are encoded in this file, but in future
// the first use of a name should load and cache a versioned template from the
// server.
func Template(name string) (string, error) {
	tpl, found := templateMap[strings.ToLower(name)]
	if !found {
		return "", ErrUnknownTemplate(name)
	}

	return tpl, nil
}

func pythonTemplate() string {
	t := `
{"Name":"Python",
 "Namespace":"default",
 "Type":"batch",
 "Count":1,
 "Tasks":[{
    "Name":"execute",
	"Engine": {
	    "Type":"python",
		"Params":{
			"Version": "{{or (index . "version") "3.11"}}"
		}
	}
 }]
}`

	return strings.TrimSpace(t)
}

func duckdbTemplate() string {
	t := `
{"Name":"DuckDB",
 "Namespace":"default",
 "Type":"batch",
 "Count":1,
 "Tasks":[{
    "Name":"execute",
	"Engine": {
	    "Type":"duckdb",
		"Params":{}
	}
 }]
}`

	return strings.TrimSpace(t)
}
