package envconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Plugin — плагин для генерации загрузки env-переменных.
type Plugin struct{}

// NewPlugin создает новый экземпляр плагина.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return "env-config" }

// Generate генерирует код загрузки переменных окружения.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	structs, err := Load(types)
	if err != nil {
		return nil, err
	}

	if len(structs) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &Generator{qual: f.Qual}
	for _, st := range structs {
		f.Add(gen.GenerateStruct(st))
	}

	return map[string]gomosaic.File{"env_config_gen.go": f}, nil
}

// Generator генерирует код для структур.
type Generator struct {
	qual gomosaic.QualFunc
}

// GenerateStruct генерирует метод LoadFromEnv() для структуры.
func (g *Generator) GenerateStruct(st *StructOpt) jen.Code {
	group := jen.NewFile("")
	structVar := "c"

	group.Func().Params(
		jen.Id(structVar).Op("*").Id(st.NameTypeInfo.Name),
	).Id("LoadFromEnv").Params().Error().BlockFunc(func(body *jen.Group) {
		g.generateFieldLoaders(body, structVar, st.Fields, "")
		body.Return(jen.Nil())
	})

	return group
}

// generateFieldLoaders генерирует код загрузки для списка полей.
func (g *Generator) generateFieldLoaders(body *jen.Group, structVar string, fields []*FieldOpt, prefix string) {
	for _, f := range fields {
		envName := f.EnvName
		if prefix != "" {
			envName = prefix + "_" + envName
		}

		// Вложенная структура
		if len(f.Children) > 0 {
			g.generateStructLoader(body, structVar, f, envName)
			continue
		}

		// Простое поле
		g.generateFieldLoader(body, structVar, f, envName)
	}
}

// generateFieldLoader генерирует загрузку одного поля.
func (g *Generator) generateFieldLoader(body *jen.Group, structVar string, f *FieldOpt, envName string) {
	fieldPath := jen.Id(structVar).Dot(f.Var.Name)
	goType := f.Var.Type

	// Основная логика: os.LookupEnv → парсинг → валидация
	body.List(jen.Id("val"), jen.Id("ok")).Op(":=").Qual("os", "LookupEnv").Call(jen.Lit(envName))

	switch {
	case f.Required && f.Default == "":
		// Обязательное поле без default
		body.If(jen.Op("!").Id("ok")).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения " + envName + " не установлена"),
			)),
		)
		g.generateAssignment(body, fieldPath, goType, "val", envName)
	case f.Default != "":
		// Поле с default-значением
		body.If(jen.Op("!").Id("ok")).Block(
			jen.Id("val").Op("=").Lit(f.Default),
		)
		g.generateAssignment(body, fieldPath, goType, "val", envName)
	default:
		// Необязательное поле
		body.If(jen.Id("ok")).BlockFunc(func(ifBody *jen.Group) {
			g.generateAssignment(ifBody, fieldPath, goType, "val", envName)
		})
	}

	// Валидация после присваивания
	g.generateValidation(body, fieldPath, f, envName)
}

// generateAssignment генерирует присваивание значения полю.
func (g *Generator) generateAssignment(body *jen.Group, fieldPath *jen.Statement, goType *gomosaic.TypeInfo, valueVar, envName string) {
	// Определяем базовый тип (снимаем Named-обёртку)
	elemType := goType
	if elemType.IsNamed && elemType.ElemType != nil {
		elemType = elemType.ElemType
	}

	switch {
	case elemType.IsBasic && elemType.BasicInfo == gomosaic.IsString:
		body.Add(fieldPath.Clone().Op("=").Id(valueVar))

	case elemType.IsBasic && elemType.BasicInfo == gomosaic.IsInteger:
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("strconv", "Atoi").Call(jen.Id(valueVar))
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Int().Call(jen.Id("v")))

	case elemType.IsBasic && elemType.BasicKind == gomosaic.Int64:
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("strconv", "ParseInt").Call(
			jen.Id(valueVar), jen.Lit(10), jen.Lit(64),
		)
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Id("v"))

	case elemType.IsBasic && gomosaic.IsFloat == elemType.BasicInfo:
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("strconv", "ParseFloat").Call(
			jen.Id(valueVar), jen.Lit(64),
		)
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Float64().Call(jen.Id("v")))

	case elemType.IsBasic && elemType.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned:
		// unsigned int type
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("strconv", "ParseUint").Call(
			jen.Id(valueVar), jen.Lit(10), jen.Lit(64),
		)
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Uint64().Call(jen.Id("v")))

	case elemType.IsBasic && elemType.BasicInfo == gomosaic.IsBoolean:
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("strconv", "ParseBool").Call(jen.Id(valueVar))
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Id("v"))

	case goType.Package == "time" && goType.Name == "Duration":
		body.List(jen.Id("v"), jen.Err()).Op(":=").Qual("time", "ParseDuration").Call(jen.Id(valueVar))
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)
		body.Add(fieldPath.Clone().Op("=").Id("v"))

	default:
		// Неподдерживаемый тип — просто присваиваем строку
		body.Add(fieldPath.Clone().Op("=").Id(valueVar))
	}
}

// generateValidation генерирует код валидации поля.
func (g *Generator) generateValidation(body *jen.Group, fieldPath *jen.Statement, f *FieldOpt, envName string) {
	if f.MaxLen > 0 {
		body.If(jen.Len(fieldPath.Clone()).Op(">").Lit(f.MaxLen)).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("переменная окружения %s: максимальная длина %d", envName, f.MaxLen)),
			)),
		)
	}

	if f.MinLen > 0 {
		body.If(jen.Len(fieldPath.Clone()).Op("<").Lit(f.MinLen)).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("переменная окружения %s: минимальная длина %d", envName, f.MinLen)),
			)),
		)
	}
}

// generateStructLoader генерирует загрузку вложенной структуры.
func (g *Generator) generateStructLoader(body *jen.Group, structVar string, f *FieldOpt, envName string) {
	// Проверяем, является ли поле указателем
	fieldPath := jen.Id(structVar).Dot(f.Var.Name)

	isPtr := f.Var.Type.IsPtr

	body.Commentf("Загрузка вложенной конфигурации %s", f.Var.Name)

	if isPtr {
		// Если указатель — создаём экземпляр
		body.If(fieldPath.Clone().Op("==").Nil()).Block(
			fieldPath.Clone().Op("=").Op("&").Add(g.qualType(f.Var.Type.ElemType)).Values(),
		)
	}

	// Рекурсивный вызов LoadFromEnv()
	body.If(
		jen.Err().Op(":=").Add(fieldPath.Clone()).Dot("LoadFromEnv").Call(),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Qual("fmt", "Errorf").Call(
			jen.Lit(strings.ToLower(f.Var.Name)+": %w"),
			jen.Err(),
		)),
	)
}

// qualType возвращает jen-код для квалификации типа.
func (g *Generator) qualType(t *gomosaic.TypeInfo) *jen.Statement {
	if t.Package != "" {
		return jen.Qual(t.Package, t.Name)
	}
	return jen.Id(t.Name)
}

// Ensure Plugin implements gomosaic.Generator.
var _ gomosaic.Generator = (*Plugin)(nil)
