package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &MapTypeParse{}

type MapTypeParse struct{}

func (s *MapTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	switch {
	default:
		panic("unknown map basic type")
	case typeInfo.BasicInfo == gomosaic.IsInteger:
		code = jen.Qual(ggRuntimePkg, "SplitKeyValInt").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned:
		code = jen.Qual(ggRuntimePkg, "SplitKeyValUint").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsFloat:
		code = jen.Qual(ggRuntimePkg, "SplitKeyValFloat").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(64), jen.Op("&").Add(assignID)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsString:
		code = jen.Qual(ggRuntimePkg, "SplitKeyValString").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Op("&").Add(assignID))
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *MapTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	switch {
	case typeInfo.BasicInfo == gomosaic.IsInteger:
		return jen.Do(qualFn(ggRuntimePkg, "JoinKeyValInt")).Call(valueID, jen.Lit(";"), jen.Lit("="), jen.Lit(10)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsFloat:
		return jen.Do(qualFn(ggRuntimePkg, "JoinKeyValFloat")).Call(valueID, jen.Lit(";"), jen.Lit("="), jen.Lit('f'), jen.Lit(2), jen.Lit(64)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsString:
		return jen.Do(qualFn(ggRuntimePkg, "JoinKeyValString")).Call(valueID, jen.Lit(";"), jen.Lit("="))
	}

	return
}

func (s *MapTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsMap && typeInfo.ElemType.IsBasic
}
