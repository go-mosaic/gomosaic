// Package envconfig предоставляет плагин для генерации кода загрузки
// переменных окружения в поля структуры на основе аннотаций.
//
// Поддерживаемые аннотации:
//
//	@env-name VAR_NAME      — имя переменной окружения
//	@env-validate required  — обязательное поле
//	@env-validate max-len=N — максимальная длина строки
//	@env-validate min-len=N — минимальная длина строки
//	@env-default value      — значение по умолчанию
//
// Пример использования:
//
//	@gomosaic
//	type Config struct {
//	    // @env-name HOST
//	    // @env-validate required max-len=100
//	    Host string
//
//	    // @env-name PORT
//	    // @env-default 8080
//	    Port int
//
//	    // @env-name DEBUG
//	    Debug bool
//
//	    Database DatabaseConfig
//	}
//
//	type DatabaseConfig struct {
//	    // @env-name DB_ADDR
//	    // @env-validate required
//	    Addr string
//
//	    // @env-name DB_PORT
//	    // @env-default 5432
//	    Port int
//	}
package envconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

// FieldOpt содержит аннотации поля структуры.
type FieldOpt struct {
	EnvName  string `option:"name,fromValue"` // @env-name
	Validate string `option:"validate"`       // @env-validate
	Default  string `option:"default"`        // @env-default

	// Вычисляемые поля
	Var      *gomosaic.VarInfo
	Required bool
	MaxLen   int
	MinLen   int

	// Для вложенных структур — рекурсивная загрузка
	Children []*FieldOpt
}

// StructOpt содержит аннотации структуры.
type StructOpt struct {
	NameTypeInfo *gomosaic.NameTypeInfo
	Fields       []*FieldOpt
}

const (
	annotationPrefix = "env"
)

// ParseValidation парсит строку валидации "required max-len=100 min-len=3".
func ParseValidation(validate string) (required bool, maxLen, minLen int, err error) {
	parts := strings.Fields(validate)
	for _, p := range parts {
		switch {
		case p == "required":
			required = true
		case strings.HasPrefix(p, "max-len="):
			v := strings.TrimPrefix(p, "max-len=")
			maxLen, err = strconv.Atoi(v)
			if err != nil {
				return false, 0, 0, fmt.Errorf("неверное значение max-len: %s", v)
			}
		case strings.HasPrefix(p, "min-len="):
			v := strings.TrimPrefix(p, "min-len=")
			minLen, err = strconv.Atoi(v)
			if err != nil {
				return false, 0, 0, fmt.Errorf("неверное значение min-len: %s", v)
			}
		}
	}
	return
}

// Load загружает аннотации конфигурации из типов.
func Load(types []*gomosaic.NameTypeInfo) (structs []*StructOpt, errs error) {
	for _, nameTypeInfo := range types {
		// Обрабатываем только структуры с @gomosaic
		if nameTypeInfo.Type.Struct == nil {
			continue
		}
		if !nameTypeInfo.Annotations.Has("gomosaic") {
			continue
		}

		st := &StructOpt{NameTypeInfo: nameTypeInfo}
		st.Fields = loadFields(nameTypeInfo.Type.Struct.Fields)
		structs = append(structs, st)
	}

	if errs != nil {
		return nil, errs
	}
	return structs, nil
}

// loadFields загружает аннотации полей.
func loadFields(fields []*gomosaic.VarInfo) []*FieldOpt {
	var result []*FieldOpt

	for _, f := range fields {
		// Пропускаем неэкспортируемые поля
		if !isExported(f.Name) {
			continue
		}

		opt := &FieldOpt{Var: f}

		// Имя переменной окружения по умолчанию — SCREAMING_SNAKE_CASE
		opt.EnvName = strcase.ToScreamingSnake(f.Name)

		// Парсим @env-name
		if ann, ok := f.Annotations.Get("env-name"); ok && len(ann.Params) > 0 {
			opt.EnvName = ann.Params[0]
		}

		// Парсим @env-validate
		if ann, ok := f.Annotations.Get("env-validate"); ok {
			opt.Validate = strings.Join(ann.Params, " ")
			required, maxLen, minLen, err := ParseValidation(opt.Validate)
			if err == nil {
				opt.Required = required
				opt.MaxLen = maxLen
				opt.MinLen = minLen
			}
		}

		// Парсим @env-default
		if ann, ok := f.Annotations.Get("env-default"); ok && len(ann.Params) > 0 {
			opt.Default = ann.Params[0]
		}

		// Для структур — рекурсивно загружаем вложенные поля
		if isStructType(f.Type) {
			opt.Children = loadFields(f.Type.ElemType.Struct.Fields)
		}

		result = append(result, opt)
	}

	return result
}

// isExported проверяет, экспортируется ли имя поля.
func isExported(name string) bool {
	if name == "" {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// isStructType проверяет, является ли тип структурой.
func isStructType(t *gomosaic.TypeInfo) bool {
	if t == nil {
		return false
	}
	if t.IsPtr {
		t = t.ElemType
	}
	if t.IsNamed {
		t = t.ElemType
	}
	return t != nil && t.Struct != nil
}

// Ensure package compiles.
var _ = multierror.Append
