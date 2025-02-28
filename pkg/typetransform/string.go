package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

var _ Parser = &StringTypeParse{}

type StringTypeParse struct{}

func (s *StringTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFunc jenutils.QualFunc) (code jen.Code) {
	return jen.Add(assignID).Op("=").Add(valueID)
}

func (s *StringTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return valueID
}

func (s *StringTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == gomosaic.IsString
}
