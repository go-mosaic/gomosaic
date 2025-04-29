package typetransform

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &IntTypeParse{}

type IntTypeParse struct{}

func (s *IntTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	var parseFunc string
	switch typeInfo.BasicInfo {
	default:
		panic(fmt.Sprintf("unknown basic number type: %v", typeInfo.BasicInfo))
	case gomosaic.IsInteger:
		parseFunc = "ParseInt"

	case gomosaic.IsInteger | gomosaic.IsUnsigned:
		parseFunc = "ParseUint"
	}
	return jen.If(jen.Err().Op(":=").Do(qualFn(ggRuntimePkg, parseFunc)).Call(
		valueID,
		jen.Lit(10), //nolint: mnd
		jen.Lit(typeInfo.BitSize),
		jen.Op("&").Add(assignID),
	), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *IntTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	//nolint: exhaustive
	switch typeInfo.BasicInfo {
	case gomosaic.IsInteger:
		return jen.Qual("strconv", "FormatInt").CallFunc(func(g *jen.Group) {
			if typeInfo.BitSize == 64 && typeInfo.BasicInfo == gomosaic.IsInteger && typeInfo.BasicKind != gomosaic.Int {
				g.Add(valueID)
			} else {
				g.Id("int64").Call(valueID)
			}
			g.Lit(10) //nolint: mnd
		})
	case gomosaic.IsInteger | gomosaic.IsUnsigned:
		return jen.Qual("strconv", "FormatUint").CallFunc(func(g *jen.Group) {
			if typeInfo.BitSize == 64 && typeInfo.BasicInfo == gomosaic.IsInteger && typeInfo.BasicKind != gomosaic.Uint {
				g.Add(valueID)
			} else {
				g.Id("uint64").Call(valueID)
			}
			g.Lit(10) //nolint: mnd
		})
	}

	return jen.Null()
}

func (s *IntTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsBasic && (typeInfo.BasicInfo == gomosaic.IsInteger || typeInfo.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned)
}
