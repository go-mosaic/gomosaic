// Package strcase переводит строки между стилями именования: CamelCase,
// snake_case, kebab-case и их вариантами в верхнем регистре.
package strcase

import (
	"strings"
)

func toCamelInitCase(s string, initCase bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	if a, ok := uppercaseAcronym[s]; ok {
		s = a
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := initCase
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

// ToCamel переводит s в CamelCase с учётом акронимов.
func ToCamel(s string) string {
	return toCamelInitCase(s, true)
}

// ToLowerCamel переводит s в lowerCamelCase с учётом акронимов.
func ToLowerCamel(s string) string {
	return toCamelInitCase(s, false)
}
