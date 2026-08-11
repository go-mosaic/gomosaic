package envconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string { return "env-config" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	structs := Load(types)
	if len(structs) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &Generator{
		qual:              f.Qual,
		transformRegistry: gomosaic.DefaultTransformRegistry(),
	}
	for _, st := range structs {
		f.Add(gen.GenerateStruct(st))
	}

	return map[string]gomosaic.File{"env_config_gen.go": f}, nil
}

type Generator struct {
	qual              gomosaic.QualFunc
	transformRegistry *gomosaic.TransformRegistry
}

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

func (g *Generator) generateFieldLoaders(body *jen.Group, structVar string, fields []*FieldOpt, prefix string) {
	for _, f := range fields {
		envName := f.EnvName
		if prefix != "" {
			envName = prefix + "_" + envName
		}

		if len(f.Children) > 0 {
			g.generateStructLoader(body, structVar, f, envName)
			continue
		}

		g.generateFieldLoader(body, structVar, f, envName)
	}
}

// generateFieldLoader генерирует загрузку одного поля.
func (g *Generator) generateFieldLoader(body *jen.Group, structVar string, f *FieldOpt, envName string) {
	fieldPath := jen.Id(structVar).Dot(f.Var.Name)
	goType := f.Var.Type

	body.List(jen.Id("val"), jen.Id("ok")).Op(":=").Qual("os", "LookupEnv").Call(jen.Lit(envName))

	switch {
	case f.Required && f.Default == "":
		body.If(jen.Op("!").Id("ok")).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения " + envName + " не установлена"),
			)),
		)
		g.generateAssignment(body, fieldPath, goType, "val", envName)
	case f.Default != "":
		body.If(jen.Op("!").Id("ok")).Block(
			jen.Id("val").Op("=").Lit(f.Default),
		)
		g.generateAssignment(body, fieldPath, goType, "val", envName)
	default:
		body.If(jen.Id("ok")).BlockFunc(func(ifBody *jen.Group) {
			g.generateAssignment(ifBody, fieldPath, goType, "val", envName)
		})
	}

	g.generateValidation(body, fieldPath, f, envName)
}

// generateAssignment генерирует присваивание значения полю через transform-реестр.
func (g *Generator) generateAssignment(body *jen.Group, fieldPath *jen.Statement, goType *gomosaic.TypeInfo, valueVar, envName string) {
	tr := g.transformRegistry.For(goType).
		SetValueID(jen.Id(valueVar)).
		SetAssignID(fieldPath.Clone()).
		SetQualFunc(g.qual).
		SetErrStatements(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("переменная окружения "+envName+": %w"),
				jen.Err(),
			)),
		)

	parseCode := tr.Parse()
	if parseCode != nil {
		body.Add(parseCode)
	} else {
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
	fieldPath := jen.Id(structVar).Dot(f.Var.Name)

	isPtr := f.Var.Type.IsPtr
	if isPtr {
		body.If(fieldPath.Clone().Op("==").Nil()).Block(
			fieldPath.Clone().Op("=").Op("&").Add(g.qualType(f.Var.Type.ElemType)).Values(),
		)
	}

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

var _ gomosaic.Generator = (*Plugin)(nil)
