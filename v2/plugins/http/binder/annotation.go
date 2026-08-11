// Package binder предоставляет генератор хелперов связывания параметров HTTP-запроса.
//
// Позволяет описать структуру с аннотациями, указав источники параметров
// (query, path, header, cookie, form, file), имена, значения по умолчанию и т.д..
// Генератор создаёт метод Bind(r *http.Request) error, заполняющий поля структуры.
package binder

import (
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/option"
)

const (
	SourceQuery  = "query"
	SourcePath   = "path"
	SourceHeader = "header"
	SourceCookie = "cookie"
	SourceForm   = "form"
	SourceFile   = "file"
)

const defaultFormMaxMemory = 32 << 20 // 32 MB

type StructOpt struct {
	NameTypeInfo  *gomosaic.NameTypeInfo
	FormMaxMemory int `option:"form-max-memory"`
	Fields        []*FieldOpt

	hasFormFields bool
}

type FieldOpt struct {
	Source   string `option:"source"`
	Name     string `option:"name"`
	Default  string `option:"default"`
	Required bool   `option:"required,asFlag"`

	Var          *gomosaic.VarInfo
	NameOverride string
}

func Load(module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (structs []*StructOpt, errs error) {
	for _, nameTypeInfo := range types {
		if nameTypeInfo.Type.Struct == nil {
			continue
		}

		st := &StructOpt{NameTypeInfo: nameTypeInfo}

		if err := option.Unmarshal("http", nameTypeInfo.Annotations, st); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("структура %s: %w", nameTypeInfo.Name, err))
			continue
		}

		if st.FormMaxMemory == 0 {
			st.FormMaxMemory = defaultFormMaxMemory
		}

		for _, field := range nameTypeInfo.Type.Struct.Fields {
			fo := &FieldOpt{Var: field, Source: SourceQuery}

			if err := option.Unmarshal("http", field.Annotations, fo); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("поле %s.%s: %w", nameTypeInfo.Name, field.Name, err))
				continue
			}

			if err := validateFieldOpt(fo); err != nil {
				errs = multierror.Append(errs, fmt.Errorf("поле %s.%s: %w", nameTypeInfo.Name, field.Name, err))
				continue
			}

			if fo.Source == SourceForm || fo.Source == SourceFile {
				st.hasFormFields = true
			}

			fo.NameOverride = resolveParamName(fo, field)

			st.Fields = append(st.Fields, fo)
		}

		structs = append(structs, st)
	}

	if errs != nil {
		return nil, errs
	}

	return structs, nil
}

func validateFieldOpt(fo *FieldOpt) error {
	switch fo.Source {
	case SourceQuery, SourcePath, SourceHeader, SourceCookie, SourceForm, SourceFile, "":
		if fo.Source == "" {
			fo.Source = SourceQuery
		}
	default:
		return fmt.Errorf("неподдерживаемый источник параметра: %q (допустимые: query, path, header, cookie, form, file)", fo.Source)
	}
	return nil
}

func resolveParamName(fo *FieldOpt, field *gomosaic.VarInfo) string {
	if fo.Name != "" {
		return fo.Name
	}
	return field.Name
}
