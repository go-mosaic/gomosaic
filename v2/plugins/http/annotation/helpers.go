package annotation

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

// WrapStruct оборачивает jen-код во вложенные структуры.
func WrapStruct(names []string, wrappedCode jen.Code) jen.Code {
	code := wrappedCode
	for i := len(names) - 1; i >= 0; i-- {
		code = jen.Id(strcase.ToCamel(names[i])).Struct(code).Tag(map[string]string{"json": names[i]})
	}
	return code
}

// MakeStructFieldsFromParams создаёт поля структуры из параметров.
func MakeStructFieldsFromParams(params []*MethodParamOpt, qual gomosaic.QualFunc) jen.Code {
	structFields := jen.NewFile("")
	for _, param := range params {
		jsonTag := param.Name
		fld := structFields.Id(strcase.ToCamel(param.Var.Name))
		if !param.Required {
			jsonTag += ",omitempty"
		}
		fld.Add(jenutils.TypeInfoQual(param.Var.Type, qual)).Tag(map[string]string{"json": jsonTag})
	}
	return structFields
}

// MakeStructFieldsFromResults создаёт поля структуры из результатов.
func MakeStructFieldsFromResults(results []*MethodResultOpt, qual gomosaic.QualFunc) jen.Code {
	structFields := jen.NewFile("")
	for _, r := range results {
		jsonTag := r.Name
		fld := structFields.Id(strcase.ToCamel(r.Var.Name))
		fld.Add(jenutils.TypeInfoQual(r.Var.Type, qual)).Tag(map[string]string{"json": jsonTag})
	}
	return structFields
}

// MakeEmptyResults возвращает нулевые значения для результатов.
func MakeEmptyResults(results []*MethodResultOpt, qualFunc gomosaic.QualFunc, addinCodes ...jen.Code) []jen.Code {
	var codes []jen.Code
	for _, r := range results {
		codes = append(codes, jenutils.ZeroValue(r.Var.Type, qualFunc))
	}
	return append(codes, addinCodes...)
}

// IsObjectType проверяет, является ли тип объектным (структура/интерфейс/мапа).
func IsObjectType(typeInfo *gomosaic.TypeInfo) bool {
	if typeInfo.IsPtr {
		typeInfo = typeInfo.ElemType
	}
	if typeInfo.IsNamed {
		typeInfo = typeInfo.ElemType
	}
	return typeInfo.Struct != nil || typeInfo.Interface != nil || typeInfo.IsMap
}

// Dot создаёт цепочку .Field1.Field2... из строк.
func Dot(parts ...string) jen.Code {
	group := jen.Null()
	for _, p := range parts {
		group.Dot(strcase.ToCamel(p))
	}
	return group
}
