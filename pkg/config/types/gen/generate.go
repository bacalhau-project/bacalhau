package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Please provide the path to the config directory.")
		os.Exit(1)
	}
	pkgPath := os.Args[1]

	fieldInfos := ConfigFieldMap(pkgPath)
	// Generate constants file
	constantsFile, err := os.Create(filepath.Join(pkgPath, "generated_constants.go"))
	if err != nil {
		panic(err)
	}
	defer constantsFile.Close()

	if err := WriteConstants(fieldInfos, constantsFile); err != nil {
		panic(err)
	}

	commentsFile, err := os.Create(filepath.Join(pkgPath, "generated_descriptions.go"))
	if err != nil {
		panic(err)
	}
	defer commentsFile.Close()

	if err := WriteComments(fieldInfos, commentsFile); err != nil {
		panic(err)
	}
}

func WriteComments(fieldInfos map[string]FieldInfo, w io.Writer) error {
	var builder strings.Builder
	// write method declaration
	builder.WriteString("// CODE GENERATED BY pkg/config/types/gen/generate.go DO NOT EDIT\n\n")
	builder.WriteString("package types\n\n")
	builder.WriteString("// ConfigDescriptions maps configuration paths to their descriptions\n")
	builder.WriteString("var ConfigDescriptions = map[string]string{\n")

	// Collect keys and sort them
	var keys []string
	for k := range fieldInfos {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Generate constant declarations
	for _, key := range keys {
		info := fieldInfos[key]
		constName := generateConstantName(info.CapitalizedPath)
		description := info.Comment
		if description == "" {
			description = "No description available"
		} else {
			// Clean up the comment: remove leading "//" and newlines
			description = strings.TrimPrefix(description, "//")
			description = strings.ReplaceAll(description, "\n", " ")
			description = strings.TrimSpace(description)
			description = escapeString(description)
		}
		builder.WriteString(fmt.Sprintf("\t%s: \"%s\",\n", constName, description))
	}
	builder.WriteString("}\n")
	// Write the content to the specified file
	if _, err := io.WriteString(w, builder.String()); err != nil {
		return err
	}

	return nil
}

func WriteConstants(fieldInfos map[string]FieldInfo, w io.Writer) error {
	var builder strings.Builder
	// write method declaration
	builder.WriteString("// CODE GENERATED BY pkg/config/types/gen/generate.go DO NOT EDIT\n\n")
	builder.WriteString("package types\n\n")

	// Collect keys and sort them
	var keys []string
	for k := range fieldInfos {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Generate constant declarations
	for _, key := range keys {
		info := fieldInfos[key]
		constName := generateConstantName(info.CapitalizedPath)
		builder.WriteString(fmt.Sprintf("const %s = \"%s\"\n", constName, key))
	}
	// Write the content to the specified file
	if _, err := io.WriteString(w, builder.String()); err != nil {
		return err
	}

	return nil
}

func ConfigFieldMap(dir string) map[string]FieldInfo {
	// Parse the package directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// Map from lowercased field path to FieldInfo
	fieldInfos := make(map[string]FieldInfo)
	typeMap := make(map[string]*ast.StructType)

	// Build a map of type names to *ast.StructType
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				// Look for type declarations
				genDecl, ok := n.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					return true
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					structName := typeSpec.Name.Name
					typeMap[structName] = structType
				}
				return false
			})
		}
	}

	// Start processing from the Bacalhau struct
	if bacalhauStruct, ok := typeMap["Bacalhau"]; ok {
		processStruct("", "", bacalhauStruct, fieldInfos, typeMap)
	} else {
		log.Fatal("Could not find Bacalhau struct")
	}

	return fieldInfos
}

// FieldInfo stores information about a field
type FieldInfo struct {
	Comment         string
	CapitalizedPath string
}

func processStruct(prefix string, capPrefix string, structType *ast.StructType, fieldInfos map[string]FieldInfo, typeMap map[string]*ast.StructType) {
	for _, field := range structType.Fields.List {
		// Get field names
		var fieldNames []string
		var fieldNamesOriginal []string
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				fieldNames = append(fieldNames, name.Name)
				fieldNamesOriginal = append(fieldNamesOriginal, name.Name)
			}
		} else {
			// Embedded field
			switch t := field.Type.(type) {
			case *ast.Ident:
				fieldNames = append(fieldNames, t.Name)
				fieldNamesOriginal = append(fieldNamesOriginal, t.Name)
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					fieldNames = append(fieldNames, ident.Name)
					fieldNamesOriginal = append(fieldNamesOriginal, ident.Name)
				}
			}
		}

		// Extract comment
		comment := ""
		if field.Doc != nil {
			comment = strings.TrimSpace(field.Doc.Text())
		} else if field.Comment != nil {
			comment = strings.TrimSpace(field.Comment.Text())
		}

		// Extract YAML tag
		tag := ""
		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			structTag := reflect.StructTag(tagValue)
			yamlTag := structTag.Get("yaml")
			if yamlTag != "" {
				tag = strings.Split(yamlTag, ",")[0]
			}
		}

		for idx, name := range fieldNames {
			// Use YAML tag or field name
			tagOrName := tag
			if tagOrName == "" {
				tagOrName = name
			}

			// Original capitalization for the constant name
			origName := fieldNamesOriginal[idx]
			if tag != "" {
				origName = tag
			}

			// Build field path
			var fieldPath string
			var capFieldPath string
			if prefix != "" {
				fieldPath = prefix + "." + tagOrName
				capFieldPath = capPrefix + "." + origName
			} else {
				fieldPath = tagOrName
				capFieldPath = origName
			}

			// Determine if the field is a leaf (non-struct) field
			isLeaf := true
			switch ft := field.Type.(type) {
			case *ast.Ident:
				if _, ok := typeMap[ft.Name]; ok {
					// Field is a named struct type
					isLeaf = false
				}
			case *ast.StarExpr:
				if ident, ok := ft.X.(*ast.Ident); ok {
					if _, ok := typeMap[ident.Name]; ok {
						// Field is a pointer to a named struct type
						isLeaf = false
					}
				}
			case *ast.StructType:
				// Field is an anonymous struct
				isLeaf = false
			}

			// If it's a leaf field, store the comment and capitalized path
			if isLeaf {
				fieldInfos[fieldPath] = FieldInfo{
					Comment:         comment,
					CapitalizedPath: capFieldPath,
				}
			} else {
				// Recursively process nested structs
				switch ft := field.Type.(type) {
				case *ast.Ident:
					if nestedStruct, ok := typeMap[ft.Name]; ok {
						processStruct(fieldPath, capFieldPath, nestedStruct, fieldInfos, typeMap)
					}
				case *ast.StarExpr:
					if ident, ok := ft.X.(*ast.Ident); ok {
						if nestedStruct, ok := typeMap[ident.Name]; ok {
							processStruct(fieldPath, capFieldPath, nestedStruct, fieldInfos, typeMap)
						}
					}
				case *ast.StructType:
					processStruct(fieldPath, capFieldPath, ft, fieldInfos, typeMap)
				}
			}
		}
	}
}

// generateConstantName converts a field path to a constant name
func generateConstantName(fieldPath string) string {
	// Split the field path by "."
	parts := strings.Split(fieldPath, ".")
	// Capitalize each part
	for i, part := range parts {
		parts[i] = capitalize(part)
	}
	// Join the parts and append "Key"
	constName := strings.Join(parts, "") + "Key"
	return constName
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// escapeString escapes double quotes in the string for safe inclusion in Go source code.
func escapeString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
