package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
)

// findAndProcessEnums searches for all enum types in the codebase and adds their values to the uniqueFields map
func findAndProcessEnums(uniqueFields map[string]bool) error {
	// Root directory to search for enum files
	rootDir := "../../../"

	// Regular expression to match go:generate enumer command
	enumRegex := regexp.MustCompile(`//go:generate go tool enumer -type=(\w+)`)

	// Walk through all files in the project
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return nil // Skip files we can't open
		}
		defer file.Close()

		// Read the first line to check for go:generate command
		scanner := bufio.NewScanner(file)
		if !scanner.Scan() {
			return nil // Skip empty files
		}

		firstLine := scanner.Text()
		matches := enumRegex.FindStringSubmatch(firstLine)
		if len(matches) < 2 {
			return nil // Not an enum file
		}

		// Extract enum type name
		enumType := matches[1]

		// Parse the file to get enum values
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Printf("Error parsing enum file %s: %v\n", path, err)
			return nil
		}

		// Find the enum constants
		var enumValues []string
		ast.Inspect(node, func(n ast.Node) bool {
			// Look for const blocks
			genDecl, ok := n.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				return true
			}

			// Process each constant in the block
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				// Skip the first value (usually Unknown)
				for i, name := range valueSpec.Names {
					if i == 0 && name.Name == "Unknown" {
						continue
					}
					enumValues = append(enumValues, name.Name)
				}
			}

			return true
		})

		// Add enum values to uniqueFields
		for _, value := range enumValues {
			uniqueFields[value] = true
		}

		fmt.Printf("Processed enum type %s with values: %v\n", enumType, enumValues)
		return nil
	})
}

func main() {
	// Path to the action package
	actionPath := "../action"

	// Map to keep track of unique field names
	uniqueFields := make(map[string]bool)

	// Walk through all files in the action package and its subdirectories
	err := filepath.Walk(actionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Printf("Error parsing file %s: %v\n", path, err)
			return nil
		}

		// Inspect the AST to find struct declarations
		ast.Inspect(node, func(n ast.Node) bool {
			// Check if this is a type declaration
			typeDecl, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			// Get the type name
			typeName := typeDecl.Name.Name

			// Check if the type name has "Data" or "FormData" suffix
			if !strings.HasSuffix(typeName, "Data") && !strings.HasSuffix(typeName, "FormData") {
				return true
			}

			// Check if this is a struct type
			structType, ok := typeDecl.Type.(*ast.StructType)
			if !ok {
				return true
			}

			// Extract field names from the struct
			for _, field := range structType.Fields.List {
				// Skip fields with form_attr_type:"hidden" tag
				if field.Tag != nil {
					tagValue := field.Tag.Value
					if strings.Contains(tagValue, `form_attr_type:"hidden"`) {
						continue
					}
				}

				for _, name := range field.Names {
					fieldName := name.Name

					// Skip fields with suffix "ID"
					if strings.HasSuffix(fieldName, "ID") {
						continue
					}

					// Add to unique fields map
					uniqueFields[fieldName] = true
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", actionPath, err)
		os.Exit(1)
	}

	// Add enum values from all enum types
	err = findAndProcessEnums(uniqueFields)

	// Convert map keys to a sorted slice
	var allFields []string
	for field := range uniqueFields {
		allFields = append(allFields, field)
	}
	sort.Strings(allFields)

	// Create output file
	outputFile, err := os.Create("form_fields.gen.go")
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	fmt.Fprintf(outputFile, "package i18n\n\n")

	fmt.Fprintf(outputFile, "import (\n")
	fmt.Fprintf(outputFile, "\"golang.org/x/text/language\"\n")
	fmt.Fprintf(outputFile, "\"golang.org/x/text/message\"\n")
	fmt.Fprintf(outputFile, ")\n\n")

	fmt.Fprintf(outputFile, "// gotext helper for translation extraction for form fields;\n")
	fmt.Fprintf(outputFile, "// necessary because they are not auto detected by `gotext update`\n")
	fmt.Fprintf(outputFile, "func formFieldsGotextHelper() {\n")
	fmt.Fprintf(outputFile, "pp := message.NewPrinter(language.English)\n")

	// Write results to the output file as a flat list
	for _, field := range allFields {
		// IMPORTANT some in newElementsFromFields
		label := strcase.ToDelimited(field, ' ')
		labelRunes := []rune(label)
		labelRunes[0] = unicode.ToUpper(labelRunes[0])
		label = string(labelRunes)
		fmt.Fprintf(outputFile, "pp.Sprintf(\"%s\")\n", label)
	}

	fmt.Fprintf(outputFile, "}")

	fmt.Printf("Extracted %d unique field names. Results written to form_fields.gen.go\n", len(allFields))
}
