package strcase

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// ToSnake переводит s в snake_case, разбивая по границам слов.
func ToSnake(s string) string {
	snake := matchFirstCap.ReplaceAllString(s, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ToKebab переводит s в kebab-case.
func ToKebab(s string) string {
	return strings.ReplaceAll(ToSnake(s), "_", "-")
}

// ToScreamingSnake переводит s в SCREAMING_SNAKE_CASE.
func ToScreamingSnake(s string) string {
	return strings.ToUpper(ToSnake(s))
}

// ToScreamingKebab переводит s в SCREAMING-KEBAB-CASE.
func ToScreamingKebab(s string) string {
	return strings.ToUpper(ToKebab(s))
}
