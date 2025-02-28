package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &BoolTypeParse{}

type BoolTypeParse struct{}

func (s *BoolTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	g := jen.NewFile("")

	g.If(
		jen.Err().Op(":=").Do(qualFn(ggRuntimePkg, "ParseBool")).Call(valueID, jen.Op("&").Add(assignID)),
		jen.Err().Op("!=").Nil(),
	).Block(errorStatements...)

	return g
}

func (s *BoolTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.Qual("strconv", "FormatBool").Call(valueID)
}

func (s *BoolTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == gomosaic.IsBoolean
}
