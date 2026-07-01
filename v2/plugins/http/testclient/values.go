package testclient

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
)

// typeToValue генерирует тестовое значение через faker.
func (g *generator) typeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	var isPtr bool
	if typeInfo.IsPtr {
		isPtr = true
		typeInfo = typeInfo.ElemType
	}

	switch {
	case typeInfo.IsBasic:
		c := g.basicTypeToValue(typeInfo)
		if isPtr {
			c = jen.Id("ptr").Call(c)
		}
		return c
	case typeInfo.IsNamed:
		var s jen.Statement
		if isPtr {
			s.Op("&")
		}
		if typeInfo.ElemType.IsBasic {
			var value any
			if typeInfo.ElemType.BasicInfo == gomosaic.IsString {
				value = "1"
			}
			return s.Do(g.qual(typeInfo.Package, typeInfo.Name)).Call(jen.Lit(value))
		}
		return s.Do(g.qual(typeInfo.Package, typeInfo.Name)).Values()
	case typeInfo.IsMap:
		if typeInfo.ElemType.IsBasic {
			return jenutils.TypeInfoQual(typeInfo, g.qual).Values(
				jen.Add(g.typeToValue(typeInfo.KeyType)).Op(":").Add(g.typeToValue(typeInfo.ElemType)),
			)
		}
		return jenutils.TypeInfoQual(typeInfo, g.qual).Values()
	case typeInfo.IsSlice:
		if typeInfo.ElemType.Struct != nil {
			return jen.Nil()
		}
		return jen.Index().Add(jenutils.TypeInfoQual(typeInfo.ElemType, g.qual)).Values()
	case typeInfo.IsArray:
		return jen.Index(jen.Lit(typeInfo.ArrayLen)).Add(jenutils.TypeInfoQual(typeInfo.ElemType, g.qual)).Values()
	case typeInfo.Struct != nil:
		return jenutils.TypeInfoQual(typeInfo, g.qual).Values()
	case typeInfo.Interface != nil:
		return jen.Nil()
	}
	return jen.Qual(FakerPkg, "New").Call().Dot("Lorem").Call().Dot("Sentence").Call(jen.Lit(10))
}

func (g *generator) basicTypeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	value := g.basicTypeToRawValue(typeInfo)
	if kind := g.basicKindToString(typeInfo.BasicKind); kind != "" {
		return jen.Id(kind).Call(value)
	}
	return value
}

// basicTypeToRawValue возвращает вызов faker без приведения типа.
func (g *generator) basicTypeToRawValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	switch typeInfo.BasicInfo {
	case gomosaic.IsBoolean:
		return jen.Lit(true)
	case gomosaic.IsInteger, gomosaic.IsInteger | gomosaic.IsUnsigned:
		return jen.Qual(FakerPkg, "New").Call().Dot("RandomNumber").Call(jen.Lit(5))
	case gomosaic.IsFloat:
		return jen.Qual(FakerPkg, "New").Call().Dot("Float64").Call(jen.Lit(2), jen.Lit(1), jen.Lit(100))
	default:
		return jen.Qual(FakerPkg, "New").Call().Dot("Lorem").Call().Dot("Sentence").Call(jen.Lit(10))
	}
}

// basicKindToString возвращает имя типа Go для приведения, либо "" если приведение не нужно.
func (g *generator) basicKindToString(kind gomosaic.BasicKind) string {
	switch kind {
	case gomosaic.Int, gomosaic.Float64, gomosaic.String:
		return ""
	case gomosaic.Int8:
		return "int8"
	case gomosaic.Int16:
		return "int16"
	case gomosaic.Int32:
		return "int32"
	case gomosaic.Int64:
		return "int64"
	case gomosaic.Uint:
		return "uint"
	case gomosaic.Uint8:
		return "uint8"
	case gomosaic.Uint16:
		return "uint16"
	case gomosaic.Uint32:
		return "uint32"
	case gomosaic.Uint64:
		return "uint64"
	case gomosaic.Uintptr:
		return "uintptr"
	case gomosaic.Float32:
		return "float32"
	case gomosaic.Complex64:
		return "complex64"
	case gomosaic.Complex128:
		return "complex128"
	default:
		return ""
	}
}
