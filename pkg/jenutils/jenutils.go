package jenutils

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type QualFunc func(pkgPath, name string) func(s *jen.Statement)

func ZeroValue(typeInfo *gomosaic.TypeInfo, qualFunc QualFunc) jen.Code {
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

func TypeInfoQual(typeInfo *gomosaic.TypeInfo, qual QualFunc) (s *jen.Statement) {
	s = new(jen.Statement)
	packagePath, name := typeInfo.Package, typeInfo.Name
	if typeInfo.IsPtr {
		s.Op("*")
		packagePath, name = typeInfo.ElemType.Package, typeInfo.ElemType.Name
	}

	s.Do(qual(packagePath, name))
	return s
}
