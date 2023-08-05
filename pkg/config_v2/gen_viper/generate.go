package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
)

// generateSetDefaults is a recursive function that takes a reflect.Type representing the structure of the configuration,
// a prefix string to build up the constant name and path, a slice of strings representing the path within the nested structure,
// a reflect.Value containing the current value being inspected, and an io.Writer to write the output to.
//
// The function iterates through the fields of the given struct type, examining each one in turn:
// - If the field has a "config" tag, the prefix is replaced with the tag's value.
// - If the field is an anonymous field, the prefix remains unchanged.
// - If the field's name is "Node" and the prefix is already "Node", the prefix remains unchanged to avoid duplication.
// - In other cases, the field's name is appended to the prefix.
//
// If the field is itself a struct, the function calls itself recursively to handle the nested fields.
// Otherwise, the function generates a line of code to set a default value in Viper using viper.SetDefault.
// If the field's value has a method called "String", this method is called in the generated code to obtain the default value.
// THIS IS IMPORTANT FOR CUSTOM VALUE TYPES
//
// The output is written to the provided io.Writer, producing a series of Viper default value assignments suitable for including
// in a Go source file.
func generateSetDefaults(t reflect.Type, prefix string, path []string, value reflect.Value, writer io.Writer) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("config")
		newPrefix := prefix
		newPath := append(path, field.Name)

		if tag != "" {
			newPrefix = tag
		} else if field.Anonymous {
			newPrefix = prefix
		} else {
			// Special handling for "Node" within "Node"
			if prefix == "Node" && field.Name == "Node" {
				newPrefix = prefix
			} else {
				newPrefix = prefix + "." + field.Name
			}
		}

		fieldValue := value.Field(i)

		if field.Type.Kind() == reflect.Struct {
			generateSetDefaults(field.Type, newPrefix, newPath, fieldValue, writer)
		} else {
			constantName := strings.ReplaceAll(newPrefix, ".", "")
			constantPath := strings.Join(newPath, ".")
			defaultValue := "cfg." + constantPath

			// Check if the field implements String() method
			stringerMethod := fieldValue.MethodByName("String")
			if stringerMethod.IsValid() {
				defaultValue = "cfg." + constantPath + ".String()"
			}

			fmt.Fprintf(writer, "viper.SetDefault(%s, %s)\n", constantName, defaultValue)
		}
	}
}

func main() {
	file, err := os.Create("generated_viper_defaults.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// You would replace this with an actual instance of BacalhauConfig with the default values set
	defaultConfig := config_v2.Default

	// Adding the package name
	fmt.Fprintf(file, "package config_v2\n\n")
	fmt.Fprintf(file, "import \"github.com/spf13/viper\"\n\n")
	fmt.Fprintf(file, "func setDefaults(cfg BacalhauConfig) {\n")

	generateSetDefaults(reflect.TypeOf(defaultConfig), "Node", []string{}, reflect.ValueOf(defaultConfig), file)

	fmt.Fprintf(file, "}\n")
}
