package gomosaic

import (
	"github.com/dave/jennifer/jen"
)

// QualFunc функция квалификации имени пакета для jen.
type QualFunc func(pkgPath, name string) func(s *jen.Statement)

// Transformer объединяет парсер и форматер для типа.
type Transformer interface {
	Parser
	Formatter
}

// Parser преобразует строковое представление в Go-значение.
type Parser interface {
	Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) (code jen.Code)
	Support(typeInfo *TypeInfo) bool
}

// Formatter преобразует Go-значение в строковое представление.
type Formatter interface {
	Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) (code jen.Code)
	Support(typeInfo *TypeInfo) bool
}

// TransformRegistry управляет регистрацией трансформеров типов.
type TransformRegistry struct {
	parseFactories  []parserFactory
	formatFactories []formatFactory
}

func NewTransformRegistry() *TransformRegistry {
	return &TransformRegistry{}
}

// AddTransformer добавляет трансформер.
func (r *TransformRegistry) AddTransformer(factory func() Transformer) {
	t := factory()

	r.AddParser(func() Parser { return t })
	r.AddFormatter(func() Formatter { return t })
}

// AddParser добавляет только парсер.
func (r *TransformRegistry) AddParser(factory func() Parser) {
	p := factory()

	r.parseFactories = append(r.parseFactories, parserFactory{
		factory: factory,
		support: p.Support,
	})
}

// AddFormatter добавляет только форматер.
func (r *TransformRegistry) AddFormatter(factory func() Formatter) {
	f := factory()

	r.formatFactories = append(r.formatFactories, formatFactory{
		factory: factory,
		support: f.Support,
	})
}

// For создает новый Transform для заданного типа, используя этот реестр.
func (r *TransformRegistry) For(typeInfo *TypeInfo) *Transform {
	return &Transform{
		typeInfo: typeInfo,
		registry: r,
		qualFn: func(pkgPath, name string) func(s *jen.Statement) {
			return func(s *jen.Statement) {
				s.Qual(pkgPath, name)
			}
		},
	}
}

// Merge добавляет все трансформеры из другого реестра.
func (r *TransformRegistry) Merge(other *TransformRegistry) {
	r.parseFactories = append(r.parseFactories, other.parseFactories...)
	r.formatFactories = append(r.formatFactories, other.formatFactories...)
}

type parserFactory struct {
	factory func() Parser
	support func(typeInfo *TypeInfo) bool
}

type formatFactory struct {
	factory func() Formatter
	support func(typeInfo *TypeInfo) bool
}

// Transform представляет операцию трансформации для конкретного типа.
type Transform struct {
	valueID, assignID jen.Code
	typeInfo          *TypeInfo
	qualFn            QualFunc
	errorStatements   []jen.Code
	registry          *TransformRegistry
}

// SetAssignID устанавливает идентификатор для присваивания.
func (tr *Transform) SetAssignID(id jen.Code) *Transform {
	tr.assignID = id
	return tr
}

// SetValueID устанавливает идентификатор исходного значения.
func (tr *Transform) SetValueID(id jen.Code) *Transform {
	tr.valueID = id
	return tr
}

// SetQualFunc устанавливает функцию квалификации имён пакетов.
func (tr *Transform) SetQualFunc(qualFn QualFunc) *Transform {
	tr.qualFn = qualFn
	return tr
}

// SetErrStatements устанавливает операторы для обработки ошибок.
func (tr *Transform) SetErrStatements(errStatements ...jen.Code) *Transform {
	tr.errorStatements = errStatements
	return tr
}

// Parse выполняет разбор значения для присваивания.
func (tr *Transform) Parse() jen.Code {
	if tr.assignID == nil {
		panic("assignID is not set")
	}

	if tr.valueID == nil {
		panic("valueID is not set")
	}

	return tr.parse(tr.typeInfo)
}

// Format выполняет форматирование значения в строку.
func (tr *Transform) Format() jen.Code {
	if tr.valueID == nil {
		panic("valueID is not set")
	}

	return tr.format(tr.typeInfo)
}

func (tr *Transform) parse(typeInfo *TypeInfo) jen.Code {
	for _, pf := range tr.registry.parseFactories {
		if pf.support(typeInfo) {
			return pf.factory().Parse(tr.valueID, tr.assignID, tr.typeInfo, tr.errorStatements, tr.qualFn)
		}
	}

	if typeInfo.IsNamed && typeInfo.ElemType.IsBasic {
		if typeInfo.ElemType.BasicInfo == IsString {
			return jen.Add(tr.assignID).Op("=").Do(tr.qualFn(typeInfo.ElemType.Package, typeInfo.ElemType.Name)).Call(tr.valueID)
		}

		return tr.registry.For(typeInfo.ElemType).
			SetAssignID(tr.assignID).
			SetValueID(tr.valueID).
			SetQualFunc(tr.qualFn).
			SetErrStatements(tr.errorStatements...).
			Parse()
	}

	return jen.Null()
}

func (tr *Transform) format(typeInfo *TypeInfo) jen.Code {
	for _, ff := range tr.registry.formatFactories {
		if ff.support(typeInfo) {
			return ff.factory().Format(tr.valueID, tr.typeInfo, tr.qualFn)
		}
	}

	return nil
}
