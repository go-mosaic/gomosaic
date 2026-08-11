// Package annotation предоставляет парсер аннотаций.
package annotation

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type Annotation struct {
	Key         string
	Params      []string
	NamedParams map[string]string
}

// Parse парсит строку аннотации.
func Parse(s string) (*Annotation, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "@") {
		return nil, fmt.Errorf("аннотация должна начинаться с @")
	}

	parts := splitAnnotation(s)
	if len(parts) == 0 {
		return nil, fmt.Errorf("пустая аннотация")
	}

	key := strings.TrimPrefix(parts[0], "@")
	ann := &Annotation{
		Key:         key,
		NamedParams: make(map[string]string),
	}

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			ann.NamedParams[kv[0]] = unquote(kv[1])
		} else {
			ann.Params = append(ann.Params, unquote(part))
		}
	}

	return ann, nil
}

// unquote удаляет обрамляющие кавычки из строки.
func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
		value = value[1 : len(value)-1]
	}

	return value
}

// splitAnnotation разбивает строку аннотации на части, учитывая кавычки и экранирование.
func splitAnnotation(s string) []string {
	var parts []string
	var buffer []rune
	var inQuotes rune
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
			switch inQuotes {
			case 0:
				inQuotes = r
			case r:
				inQuotes = 0
			}
		}

		if unicode.IsSpace(r) && inQuotes == 0 {
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

func (a *Annotation) HasParam(param string) bool {
	return slices.Contains(a.Params, param)
}

func (a *Annotation) GetNamed(name string) (string, bool) {
	v, ok := a.NamedParams[name]
	return v, ok
}
