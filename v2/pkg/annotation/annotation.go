// Package annotation предоставляет парсер аннотаций.
package annotation

import (
	"fmt"
	"slices"
	"strings"
)

type Annotation struct {
	Key         string
	Params      []string
	NamedParams map[string]string
}

// Parse парсит строку аннотации вида "@key param1 param2".
func Parse(s string) (*Annotation, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "@") {
		return nil, fmt.Errorf("аннотация должна начинаться с @")
	}

	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil, fmt.Errorf("пустая аннотация")
	}

	key := strings.TrimPrefix(parts[0], "@")
	ann := &Annotation{
		Key:         key,
		NamedParams: make(map[string]string),
	}

	for _, p := range parts[1:] {
		if strings.Contains(p, "=") {
			kv := strings.SplitN(p, "=", 2)
			ann.NamedParams[kv[0]] = kv[1]
		} else {
			ann.Params = append(ann.Params, p)
		}
	}

	return ann, nil
}

// HasParam проверяет наличие позиционного параметра.
func (a *Annotation) HasParam(param string) bool {
	return slices.Contains(a.Params, param)
}

// GetNamed возвращает именованный параметр.
func (a *Annotation) GetNamed(name string) (string, bool) {
	v, ok := a.NamedParams[name]
	return v, ok
}
