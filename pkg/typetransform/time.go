package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &TimeTypeParse{}

type TimeTypeParse struct{}

func (s *TimeTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	switch typeInfo.Name {
	default:
		panic("unknown time pkg type")
	case "Time":
		code = jen.Do(qualFn(ggRuntimePkg, "ParseTime")).Call(
			jen.Do(qualFn("time", "RFC3339")),
			valueID,
			jen.Op("&").Add(assignID),
		)
	case "Duration":
		code = jen.Do(qualFn(ggRuntimePkg, "ParseDuration")).Call(
			valueID,
			jen.Op("&").Add(assignID),
		)
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *TimeTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.Add(valueID).Dot("Format").Call(jen.Do(qualFn("time", "RFC3339")))
}

func (s *TimeTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.Package == "time"
}
