package jenutils

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// ZeroValue возвращает нулевое значение для типа.
func ZeroValue(typeInfo *gomosaic.TypeInfo, qualFunc gomosaic.QualFunc) jen.Code {
	switch {
	case typeInfo.BasicInfo == gomosaic.IsBoolean:
		return jen.Lit(false)
	case typeInfo.BasicInfo == gomosaic.IsString:
		return jen.Lit("")
	case typeInfo.BasicInfo == gomosaic.IsInteger:
		return jen.Lit(0)
	case typeInfo.BasicInfo == gomosaic.IsFloat:
		return jen.Lit(0.0)
	case typeInfo.Package != "" && typeInfo.Name != "":
		return jen.Do(qualFunc(typeInfo.Package, typeInfo.Name)).Values()
	}

	return jen.Nil()
}

// TypeInfoQual возвращает jen-выражение для квалификации типа.
func TypeInfoQual(typeInfo *gomosaic.TypeInfo, qual gomosaic.QualFunc) (s *jen.Statement) {
	s = new(jen.Statement)
	switch {
	case typeInfo.IsBasic:
		s.Id(typeInfo.Name)
		return s
	case typeInfo.IsAlias:
		s.Id(typeInfo.Name)
		return s
	case typeInfo.IsPtr:
		s.Op("*").Add(TypeInfoQual(typeInfo.ElemType, qual))
		return s
	case typeInfo.IsMap:
		s.Map(TypeInfoQual(typeInfo.KeyType, qual)).Add(TypeInfoQual(typeInfo.ElemType, qual))
		return s
	case typeInfo.IsArray:
		return s.Index(jen.Lit(typeInfo.ArrayLen)).Add(TypeInfoQual(typeInfo.ElemType, qual))
	case typeInfo.IsSlice:
		return s.Index().Add(TypeInfoQual(typeInfo.ElemType, qual))
	case typeInfo.IsNamed:
		s.Do(qual(typeInfo.Package, typeInfo.Name))

		if typeInfo.IsInstantiated {
			s.IndexFunc(func(g *jen.Group) {
				for _, typeParam := range typeInfo.TypeParams {
					g.Add(TypeInfoQual(typeParam, qual))
				}
			})
		}
		return s
	}

	panic(fmt.Sprintf("unknown TypeInfo: %+v", *typeInfo))
}
