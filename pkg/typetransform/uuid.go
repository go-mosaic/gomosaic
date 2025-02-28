package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

const (
	googleUUIDPkg = "github.com/google/uuid"
	satoriUUIDPkg = "github.com/satori/go.uuid"
)

var _ Parser = &UUIDTypeParse{}

type UUIDTypeParse struct{}

func (s *UUIDTypeParse) Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code) {
	var parseFuncName jen.Code
	switch typeInfo.Package {
	case googleUUIDPkg:
		parseFuncName = jen.Qual(typeInfo.Package, "Parse")
	case satoriUUIDPkg:
		parseFuncName = jen.Qual(typeInfo.Package, "FromString")
	}

	return jen.If(jen.Err().Op(":=").Qual(ggRuntimePkg, "ParseUUID").Call(valueID, parseFuncName, jen.Op("&").Add(assignID)), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *UUIDTypeParse) Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code) {
	return jen.Add(valueID).Dot("String").Call()
}

func (s *UUIDTypeParse) Support(typeInfo *gomosaic.TypeInfo) bool {
	return (typeInfo.Package == satoriUUIDPkg || typeInfo.Package == googleUUIDPkg) && typeInfo.Name == "UUID"
}
