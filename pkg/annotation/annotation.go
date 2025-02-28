package annotation

import (
	"fmt"
	"strings"
	"unicode"
)

type Annotation struct {
	Key     string
	Options []string
	Params  map[string]string
}

func (a *Annotation) Value() string {
	if len(a.Options) > 0 {
		return a.Options[0]
	}

	return ""
}

// parseAnnotation парсит аннотацию.
func parseAnnotation(s string) (*Annotation, error) {
	s = strings.TrimSpace(s)
	parts := splitAnnotation(s)

	if len(parts) == 0 {
		return nil, fmt.Errorf("annotation is empty")
	}

	if !strings.HasPrefix(parts[0], "@") {
		return nil, fmt.Errorf("annotation nod found")
	}

	a := &Annotation{
		Key:    parts[0][1:],
		Params: make(map[string]string),
	}

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.Contains(part, "=") {
			keyValue := strings.SplitN(part, "=", 2) //nolint: mnd
			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])
			a.Params[key] = unquote(value)
		} else {
			a.Options = append(a.Options, unquote(part))
		}
	}

	return a, nil
}

func unquote(value string) string {
	if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
		value = value[1 : len(value)-1]
	}

	return value
}

func splitAnnotation(s string) []string {
	var parts []string
	var buffer []rune
	inQuotes := false
	escape := false

	for _, r := range s {
		if escape {
			buffer = append(buffer, r)
			escape = false
			continue
		}

		if r == '\\' {
			escape = true
			continue
		}

		if r == '"' || r == '\'' {
			inQuotes = !inQuotes
		}

		if unicode.IsSpace(r) && !inQuotes {
			if len(buffer) > 0 {
				parts = append(parts, string(buffer))
				buffer = nil
			}
		} else {
			buffer = append(buffer, r)
		}
	}

	if len(buffer) > 0 {
		parts = append(parts, string(buffer))
	}

	return parts
}

func Parse(s string) (*Annotation, error) {
	return parseAnnotation(s)
}
