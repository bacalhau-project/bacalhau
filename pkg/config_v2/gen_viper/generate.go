package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
)

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
			fmt.Fprintf(writer, "viper.SetDefault(%s, Default.%s)\n", constantName, constantPath)
		}
	}
}

func main() {
	file, err := os.Create("viper_defaults.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// You would replace this with an actual instance of BacalhauConfig with the default values set
	defaultConfig := config_v2.Default

	// Adding the package name
	fmt.Fprintf(file, "package config_v2\n\n")
	fmt.Fprintf(file, "import \"github.com/spf13/viper\"\n\n")
	fmt.Fprintf(file, "func setDefaults() {\n")

	generateSetDefaults(reflect.TypeOf(defaultConfig), "Node", []string{}, reflect.ValueOf(defaultConfig), file)

	fmt.Fprintf(file, "}\n")
}
