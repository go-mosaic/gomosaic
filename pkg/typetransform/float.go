package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &FloatTypeParse{}

type FloatTypeParse struct{}

func (s *FloatTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	g := jen.NewFile("")

	g.If(
		jen.Err().Op(":=").Do(qualFn(ggRuntimePkg, "ParseFloat")).Call(valueID, jen.Lit(10), jen.Lit(typeInfo.BitSize), jen.Op("&").Add(assignID)), //nolint: mnd
		jen.Err().Op("!=").Nil(),
	).Block(errorStatements...)

	return g
}

func (s *FloatTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.Qual("strconv", "FormatFloat").CallFunc(func(g *jen.Group) {
		if typeInfo.BitSize == 64 { //nolint: mnd
			g.Add(valueID)
		} else {
			g.Id("float64").Call(valueID)
		}
		g.LitRune('g')
		g.Lit(2) //nolint: mnd
		g.Lit(typeInfo.BitSize)
	})
}

func (s *FloatTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == gomosaic.IsFloat
}
