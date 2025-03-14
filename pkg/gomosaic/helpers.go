package gomosaic

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Регулярное выражение для поиска имени функции и параметров
	funcCallRegexp = regexp.MustCompile(`^(\w+)\(([^)]*)\)$`)
)

func IsTime(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Time"
}

func IsDuration(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Duration"
}

func HasError(vars []*VarInfo) bool {
	for _, v := range vars {
		if v.IsError {
			return true
		}
	}

	return false
}

// ParseFunctionCall парсит строку вызова функции и возвращает имя функции и список параметров
func ParseFunctionCall(input string) (funcName string, params []string, err error) {
	matches := funcCallRegexp.FindStringSubmatch(input)

	if len(matches) != 3 { //nolint: mnd
		return "", nil, fmt.Errorf("неверный формат строки")
	}

	funcName = matches[1]
	paramsStr := matches[2]

	if paramsStr != "" {
		params = strings.Split(paramsStr, ",")
		for i := range params {
			params[i] = strings.TrimSpace(params[i])
		}
	}

	return funcName, params, nil
}
