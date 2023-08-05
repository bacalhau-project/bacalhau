package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
)

func main() {
	config := config_v2.BacalhauConfig{}

	// Open a file for writing
	file, err := os.Create("paths.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write the package declaration
	fmt.Fprintln(file, "package config_v2")

	generateConstants(reflect.TypeOf(config), "Node", file)
}

func generateConstants(t reflect.Type, prefix string, file *os.File) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("config")
		newPrefix := prefix

		if tag != "" {
			newPrefix = tag
		} else if field.Anonymous {
			newPrefix = prefix // Keep the existing prefix
		} else {
			// Special handling for "Node" within "Node"
			if prefix == "Node" && field.Name == "Node" {
				newPrefix = prefix
			} else {
				newPrefix = prefix + "." + field.Name
			}
		}

		if field.Type.Kind() == reflect.Struct {
			generateConstants(field.Type, newPrefix, file)
		} else {
			constantName := strings.ReplaceAll(newPrefix, ".", "")
			constantValue := newPrefix
			fmt.Fprintf(file, "const %s = \"%s\"\n", constantName, constantValue)
		}
	}
}
