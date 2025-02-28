package typetransform

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
)

const (
	ggRuntimePkg = "github.com/go-mosaic/runtime"
)

func init() {
	AddTransformer(func() Transformer {
		return new(StringTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(FloatTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(IntTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(BoolTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(TimeTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(URLTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(SliceTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(MapTypeParse)
	})
	AddTransformer(func() Transformer {
		return new(UUIDTypeParse)
	})
}

type Transformer interface {
	Parser
	Formatter
}

type Parser interface {
	Parse(valueID, assignID jen.Code, typeInfo *gomosaic.TypeInfo, errorStatements []jen.Code, qualFn jenutils.QualFunc) (code jen.Code)
	Support(typeInfo *gomosaic.TypeInfo) bool
}

type Formatter interface {
	Format(valueID jen.Code, typeInfo *gomosaic.TypeInfo, qualFn jenutils.QualFunc) (code jen.Code)
	Support(typeInfo *gomosaic.TypeInfo) bool
}

func AddTransformer(f func() Transformer) {
	parseFactories = append(parseFactories, parserFactory{
		factory: func() Parser {
			return f()
		},
		support: f().Support,
	})
	formatFactories = append(formatFactories, formatFactory{
		factory: func() Formatter {
			return f()
		},
		support: f().Support,
	})
}

func AddParse(f func() Parser) {
	parseFactories = append(parseFactories, parserFactory{
		factory: f,
		support: f().Support,
	})
}

func AddFormat(typeName string, f func() Formatter) {
	formatFactories = append(formatFactories, formatFactory{
		factory: f,
		support: f().Support,
	})
}

type Transform struct {
	valueID, assignID jen.Code
	typeInfo          *gomosaic.TypeInfo
	qualFn            jenutils.QualFunc
	errorStatements   []jen.Code
}

func (tr *Transform) parse(typeInfo *gomosaic.TypeInfo) (code jen.Code) {
	for _, pf := range parseFactories {
		if pf.support(typeInfo) {
			return pf.factory().Parse(tr.valueID, tr.assignID, tr.typeInfo, tr.errorStatements, tr.qualFn)
		}
	}

	if typeInfo.Named != nil && typeInfo.Named.IsBasic {
		if typeInfo.Named.BasicInfo == gomosaic.IsString {
			return jen.Add(tr.assignID).Op("=").Do(tr.qualFn(typeInfo.Named.Package, typeInfo.Named.Name)).Call(tr.valueID)
		}
		return For(typeInfo.Named).
			SetAssignID(tr.assignID).
			SetValueID(tr.valueID).
			SetQualFunc(tr.qualFn).
			SetErrStatements(tr.errorStatements...).
			Parse()
	}

	return jen.Null()
}

func (tr *Transform) format(typeInfo *gomosaic.TypeInfo) (code jen.Code) {
	for _, pf := range formatFactories {
		if pf.support(typeInfo) {
			return pf.factory().Format(tr.valueID, tr.typeInfo, tr.qualFn)
		}
	}
	return
}

func (tr *Transform) Parse() (result jen.Code) {
	if tr.assignID == nil {
		panic("assignID is not set")
	}
	if tr.valueID == nil {
		panic("valueID is not set")
	}
	code := tr.parse(tr.typeInfo)

	return code
}

func (tr *Transform) Format() (code jen.Code) {
	if tr.valueID == nil {
		panic("valueID is not set")
	}
	code = tr.format(tr.typeInfo)

	return code
}

func (tr *Transform) SetAssignID(id jen.Code) *Transform {
	tr.assignID = id
	return tr
}

func (tr *Transform) SetValueID(id jen.Code) *Transform {
	tr.valueID = id
	return tr
}

func (tr *Transform) SetQualFunc(qualFn jenutils.QualFunc) *Transform {
	tr.qualFn = qualFn
	return tr
}

func (tr *Transform) SetErrStatements(errStatements ...jen.Code) *Transform {
	tr.errorStatements = errStatements
	return tr
}

func For(typeInfo *gomosaic.TypeInfo) *Transform {
	return &Transform{
		typeInfo: typeInfo,
		qualFn: func(pkgPath, name string) func(s *jen.Statement) {
			return func(s *jen.Statement) {
				s.Qual(pkgPath, name)
			}
		},
	}
}

var formatFactories []formatFactory
var parseFactories []parserFactory

type parserFactory struct {
	factory func() Parser
	support func(typeInfo *gomosaic.TypeInfo) bool
}

type formatFactory struct {
	factory func() Formatter
	support func(typeInfo *gomosaic.TypeInfo) bool
}
