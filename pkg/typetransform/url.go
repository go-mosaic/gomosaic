package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &URLTypeParse{}

type URLTypeParse struct{}

func (s *URLTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.If(jen.Err().Op(":=").Do(qualFn(ggRuntimePkg, "ParseURL")).Call(
		valueID,
		jen.Op("&").Add(assignID),
	), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *URLTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.Add(valueID).Dot("String").Call()
}

func (s *URLTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.Package == "net/url" && typeInfo.Name == "URL"
}
