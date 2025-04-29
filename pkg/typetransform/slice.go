package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &SliceTypeParse{}

type SliceTypeParse struct{}

func (s *SliceTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	switch typeInfo.BasicInfo {
	default:
		panic("unknown slice basic type")
	case gomosaic.IsString:
		code = jen.Do(qualFn(ggRuntimePkg, "Split")).Call(valueID, jen.Lit(";"), jen.Op("&").Add(assignID))
	case gomosaic.IsNumeric:
		code = jen.Do(qualFn(ggRuntimePkg, "SplitInt")).Call(valueID, jen.Lit(";"), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID)) //nolint: mnd
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *SliceTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	switch typeInfo.BasicInfo { //nolint: exhaustive
	case gomosaic.IsInteger:
		return jen.Do(qualFn(ggRuntimePkg, "JoinInt")).Call(valueID, jen.Lit(","), jen.Lit(10)) //nolint: mnd
	case gomosaic.IsFloat:
		return jen.Do(qualFn(ggRuntimePkg, "JoinFloat")).Call(valueID, jen.Lit(","), jen.Lit('f'), jen.Lit(2), jen.Lit(64)) //nolint: mnd
	case gomosaic.IsString:
		return jen.Do(qualFn("strings", "Join")).Call(valueID, jen.Lit(","))
	}

	return
}

func (s *SliceTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsSlice && typeInfo.ElemType.IsBasic
}
