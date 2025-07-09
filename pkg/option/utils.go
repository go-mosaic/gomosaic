package option

import (
	"reflect"
	"slices"
	"strings"

	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

// hasInlineOption checks if the "inline" option is present in the given options.
func hasInlineOption(options []string) bool {
	return slices.Contains(options, "inline")
}

// parseTag parses the "option" tag from a struct field.
// Returns the name, options, and a boolean indicating if the tag was found.
func parseTag(fieldType reflect.StructField) (name string, options []string, ok bool) {
	tagValue, ok := fieldType.Tag.Lookup("option")
	if !ok {
		return "", nil, false
	}
	name, options = parseTagValue(fieldType.Name, tagValue)
	return name, options, true
}

// parseTagValue parses the value of the "option" tag.
// If the name is empty, it converts the field name to kebab-case.
func parseTagValue(fieldName string, tagValue string) (name string, options []string) {
	tagParts := strings.Split(tagValue, ",")
	if len(tagParts) > 0 {
		name = tagParts[0]
		options = tagParts[1:]
		if name == "" {
			name = strcase.ToKebab(fieldName)
		}
		return name, options
	}
	return "", nil
}
