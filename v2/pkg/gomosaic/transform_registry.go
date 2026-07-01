package gomosaic

import (
	"github.com/dave/jennifer/jen"
)

const runtimePkg = "github.com/go-mosaic/runtime"

// DefaultTransformRegistry возвращает реестр со всеми встроенными трансформерами.
func DefaultTransformRegistry() *TransformRegistry {
	r := NewTransformRegistry()

	r.AddTransformer(newStringTypeParse)
	r.AddTransformer(newIntTypeParse)
	r.AddTransformer(newFloatTypeParse)
	r.AddTransformer(newBoolTypeParse)
	r.AddTransformer(newTimeTypeParse)
	r.AddTransformer(newSliceTypeParse)
	r.AddTransformer(newMapTypeParse)
	r.AddTransformer(newURLTypeParse)
	r.AddTransformer(newUUIDTypeParse)

	return r
}

func newStringTypeParse() Transformer { return &stringTypeParse{} }
func newIntTypeParse() Transformer    { return &intTypeParse{} }
func newFloatTypeParse() Transformer  { return &floatTypeParse{} }
func newBoolTypeParse() Transformer   { return &boolTypeParse{} }
func newTimeTypeParse() Transformer   { return &timeTypeParse{} }
func newSliceTypeParse() Transformer  { return &sliceTypeParse{} }
func newMapTypeParse() Transformer    { return &mapTypeParse{} }
func newURLTypeParse() Transformer    { return &urlTypeParse{} }
func newUUIDTypeParse() Transformer   { return &uuidTypeParse{} }

type stringTypeParse struct{}
type intTypeParse struct{}
type floatTypeParse struct{}
type boolTypeParse struct{}
type timeTypeParse struct{}
type sliceTypeParse struct{}
type mapTypeParse struct{}
type urlTypeParse struct{}
type uuidTypeParse struct{}

func (s *stringTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == IsString
}

func (s *stringTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	return jen.Add(assignID).Op("=").Add(valueID)
}

func (s *stringTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return valueID
}

func (i *intTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsBasic && (typeInfo.BasicInfo == IsInteger || typeInfo.BasicInfo == IsInteger|IsUnsigned)
}

func (i *intTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	var parseFunc string

	switch typeInfo.BasicInfo {
	default:
		panic("unknown basic number type")
	case IsInteger:
		parseFunc = "ParseInt"
	case IsInteger | IsUnsigned:
		parseFunc = "ParseUint"
	}

	return jen.If(jen.Err().Op(":=").Do(qualFn(runtimePkg, parseFunc)).Call(
		valueID,
		jen.Lit(10),
		jen.Lit(typeInfo.BitSize),
		jen.Op("&").Add(assignID),
	), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (i *intTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	switch typeInfo.BasicInfo {
	case IsInteger:
		return jen.Qual("strconv", "FormatInt").CallFunc(func(g *jen.Group) {
			if typeInfo.BitSize == 64 && typeInfo.BasicKind != Int {
				g.Add(valueID)
			} else {
				g.Id("int64").Call(valueID)
			}

			g.Lit(10)
		})
	case IsInteger | IsUnsigned:
		return jen.Qual("strconv", "FormatUint").CallFunc(func(g *jen.Group) {
			if typeInfo.BitSize == 64 && typeInfo.BasicKind != Uint {
				g.Add(valueID)
			} else {
				g.Id("uint64").Call(valueID)
			}

			g.Lit(10)
		})
	}

	return jen.Null()
}

func (f *floatTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == IsFloat
}

func (f *floatTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	const runtimePkg = "github.com/go-mosaic/runtime"

	return jen.If(
		jen.Err().Op(":=").Do(qualFn(runtimePkg, "ParseFloat")).Call(valueID, jen.Lit(10), jen.Lit(typeInfo.BitSize), jen.Op("&").Add(assignID)),
		jen.Err().Op("!=").Nil(),
	).Block(errorStatements...)
}

func (f *floatTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return jen.Qual("strconv", "FormatFloat").CallFunc(func(g *jen.Group) {
		if typeInfo.BitSize == 64 {
			g.Add(valueID)
		} else {
			g.Id("float64").Call(valueID)
		}

		g.LitRune('g')
		g.Lit(2)
		g.Lit(typeInfo.BitSize)
	})
}

func (b *boolTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsBasic && typeInfo.BasicInfo == IsBoolean
}

func (b *boolTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	return jen.If(
		jen.Err().Op(":=").Do(qualFn(runtimePkg, "ParseBool")).Call(valueID, jen.Op("&").Add(assignID)),
		jen.Err().Op("!=").Nil(),
	).Block(errorStatements...)
}

func (b *boolTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return jen.Qual("strconv", "FormatBool").Call(valueID)
}

func (t *timeTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time"
}

func (t *timeTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	var code jen.Code

	switch typeInfo.Name {
	case "Time":
		code = jen.Do(qualFn(runtimePkg, "ParseTime")).Call(
			jen.Do(qualFn("time", "RFC3339")),
			valueID,
			jen.Op("&").Add(assignID),
		)
	case "Duration":
		code = jen.Do(qualFn(runtimePkg, "ParseDuration")).Call(
			valueID,
			jen.Op("&").Add(assignID),
		)
	default:
		panic("unknown time pkg type")
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (t *timeTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return jen.Add(valueID).Dot("Format").Call(jen.Do(qualFn("time", "RFC3339")))
}

func (s *sliceTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsSlice && typeInfo.ElemType.IsBasic
}

func (s *sliceTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	var code jen.Code

	switch typeInfo.ElemType.BasicInfo {
	case IsString:
		code = jen.Do(qualFn(runtimePkg, "Split")).Call(valueID, jen.Lit(";"), jen.Op("&").Add(assignID))
	case IsNumeric:
		code = jen.Do(qualFn(runtimePkg, "SplitInt")).Call(valueID, jen.Lit(";"), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID))
	default:
		panic("unknown slice basic type")
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (s *sliceTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	switch typeInfo.BasicInfo {
	case IsInteger:
		return jen.Do(qualFn(runtimePkg, "JoinInt")).Call(valueID, jen.Lit(","), jen.Lit(10))
	case IsFloat:
		return jen.Do(qualFn(runtimePkg, "JoinFloat")).Call(valueID, jen.Lit(","), jen.Lit('f'), jen.Lit(2), jen.Lit(64))
	case IsString:
		return jen.Do(qualFn("strings", "Join")).Call(valueID, jen.Lit(","))
	}

	return nil
}

func (m *mapTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.IsMap && typeInfo.ElemType.IsBasic
}

func (m *mapTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	var code jen.Code

	switch typeInfo.ElemType.BasicInfo {
	case IsInteger:
		code = jen.Qual(runtimePkg, "SplitKeyValInt").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID))
	case IsInteger | IsUnsigned:
		code = jen.Qual(runtimePkg, "SplitKeyValUint").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(10), jen.Lit(64), jen.Op("&").Add(assignID))
	case IsFloat:
		code = jen.Qual(runtimePkg, "SplitKeyValFloat").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Lit(64), jen.Op("&").Add(assignID))
	case IsString:
		code = jen.Qual(runtimePkg, "SplitKeyValString").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Op("&").Add(assignID))
	case IsBoolean:
		code = jen.Qual(runtimePkg, "SplitKeyValBool").Call(valueID, jen.Lit(","), jen.Lit("="), jen.Op("&").Add(assignID))
	default:
		panic("unknown map basic type")
	}

	return jen.If(jen.Err().Op(":=").Add(code), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (m *mapTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	switch typeInfo.BasicInfo {
	case IsInteger:
		return jen.Do(qualFn(runtimePkg, "JoinKeyValInt")).Call(valueID, jen.Lit(";"), jen.Lit("="), jen.Lit(10))
	case IsFloat:
		return jen.Do(qualFn(runtimePkg, "JoinKeyValFloat")).Call(valueID, jen.Lit(";"), jen.Lit("="), jen.Lit('f'), jen.Lit(2), jen.Lit(64))
	case IsString:
		return jen.Do(qualFn(runtimePkg, "JoinKeyValString")).Call(valueID, jen.Lit(";"), jen.Lit("="))
	}

	return nil
}

func (u *urlTypeParse) Support(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "net/url" && typeInfo.Name == "URL"
}

func (u *urlTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	return jen.If(jen.Err().Op(":=").Do(qualFn(runtimePkg, "ParseURL")).Call(
		valueID,
		jen.Op("&").Add(assignID),
	), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (u *urlTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return jen.Add(valueID).Dot("String").Call()
}

func (u *uuidTypeParse) Support(typeInfo *TypeInfo) bool {
	const (
		googleUUIDPkg = "github.com/google/uuid"
		satoriUUIDPkg = "github.com/satori/go.uuid"
	)
	return (typeInfo.Package == satoriUUIDPkg || typeInfo.Package == googleUUIDPkg) && typeInfo.Name == "UUID"
}

func (u *uuidTypeParse) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	var parseFuncName jen.Code

	switch typeInfo.Package {
	case "github.com/google/uuid":
		parseFuncName = jen.Qual(typeInfo.Package, "Parse")
	case "github.com/satori/go.uuid":
		parseFuncName = jen.Qual(typeInfo.Package, "FromString")
	}

	return jen.If(jen.Err().Op(":=").Qual(runtimePkg, "ParseUUID").Call(valueID, parseFuncName, jen.Op("&").Add(assignID)), jen.Err().Op("!=").Nil()).Block(errorStatements...)
}

func (u *uuidTypeParse) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return jen.Add(valueID).Dot("String").Call()
}
