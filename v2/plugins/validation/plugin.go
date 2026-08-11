// Package validation предоставляет плагин для генерации методов валидации структур.
//
// Поддерживаемые аннотации:
//
//	@validate required         — обязательное поле
//	@validate min=N            — минимальное значение (числа)
//	@validate max=N            — максимальное значение (числа)
//	@validate min-len=N        — минимальная длина строки
//	@validate max-len=N        — максимальная длина строки
//	@validate pattern=regex    — регулярное выражение
//	@validate email            — валидный email
//	@validate url              — валидный URL
//
// Пример:
//
//	type User struct {
//	    // @validate required min-len=2 max-len=50
//	    Name string
//
//	    // @validate required email
//	    Email string
//
//	    // @validate required min=18 max=150
//	    Age int
//	}
//
// Генерирует метод:
//
//	func (u *User) Validate() error { ... }
package validation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Rule содержит правила валидации для поля.
type Rule struct {
	Required bool
	Min      *int64
	Max      *int64
	MinLen   *int
	MaxLen   *int
	Pattern  string
	IsEmail  bool
	IsURL    bool
}

// FieldRules — правила для конкретного поля.
type FieldRules struct {
	Var  *gomosaic.VarInfo
	Rule Rule
}

// StructRules — правила для структуры.
type StructRules struct {
	NameTypeInfo *gomosaic.NameTypeInfo
	Fields       []*FieldRules
}

// Plugin — плагин генерации валидации.
type Plugin struct{}

// NewPlugin создает новый экземпляр плагина.
func NewPlugin() *Plugin { return &Plugin{} }

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return "validation" }

// Generate генерирует методы Validate().
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	structs := loadValidationRules(types)
	if len(structs) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := &generator{qual: f.Qual}

	for _, st := range structs {
		f.Add(gen.generateStruct(st))
	}

	return map[string]gomosaic.File{"validation_gen.go": f}, nil
}

// loadValidationRules загружает правила валидации из аннотаций.
func loadValidationRules(types []*gomosaic.NameTypeInfo) []*StructRules {
	var result []*StructRules
	for _, nt := range types {
		if nt.Type.Struct == nil {
			continue
		}
		rules := &StructRules{NameTypeInfo: nt}
		for _, field := range nt.Type.Struct.Fields {
			ann, ok := field.Annotations.Get("validate")
			if !ok {
				continue
			}
			rule := parseRule(strings.Join(ann.Params, " "))
			rules.Fields = append(rules.Fields, &FieldRules{Var: field, Rule: rule})
		}
		if len(rules.Fields) > 0 {
			result = append(result, rules)
		}
	}
	return result
}

func parseRule(s string) Rule {
	r := Rule{}
	parts := strings.FieldsSeq(s)
	for p := range parts {
		switch {
		case p == "required":
			r.Required = true
		case p == "email":
			r.IsEmail = true
		case p == "url":
			r.IsURL = true
		case strings.HasPrefix(p, "min="):
			if v, err := strconv.ParseInt(strings.TrimPrefix(p, "min="), 10, 64); err == nil {
				r.Min = &v
			}
		case strings.HasPrefix(p, "max="):
			if v, err := strconv.ParseInt(strings.TrimPrefix(p, "max="), 10, 64); err == nil {
				r.Max = &v
			}
		case strings.HasPrefix(p, "min-len="):
			if v, err := strconv.Atoi(strings.TrimPrefix(p, "min-len=")); err == nil {
				r.MinLen = &v
			}
		case strings.HasPrefix(p, "max-len="):
			if v, err := strconv.Atoi(strings.TrimPrefix(p, "max-len=")); err == nil {
				r.MaxLen = &v
			}
		case strings.HasPrefix(p, "pattern="):
			r.Pattern = strings.TrimPrefix(p, "pattern=")
		}
	}
	return r
}

type generator struct {
	qual gomosaic.QualFunc
}

func (g *generator) generateStruct(st *StructRules) jen.Code {
	group := jen.NewFile("")
	structVar := "v"
	errVar := "errs"

	group.Func().Params(
		jen.Id(structVar).Op("*").Id(st.NameTypeInfo.Name),
	).Id("Validate").Params().Error().BlockFunc(func(body *jen.Group) {
		body.Var().Id(errVar).Error()

		for _, fr := range st.Fields {
			fieldName := fr.Var.Name
			fieldPath := jen.Id(structVar).Dot(fieldName)

			if fr.Rule.Required {
				g.genRequiredCheck(body, fieldPath, fr.Var.Type, fieldName)
			}

			if fr.Rule.IsEmail && fr.Var.Type.IsBasic && fr.Var.Type.BasicInfo == gomosaic.IsString {
				g.genPatternCheck(body, fieldPath, fieldName, `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "невалидный email")
			}

			if fr.Rule.IsURL && fr.Var.Type.IsBasic && fr.Var.Type.BasicInfo == gomosaic.IsString {
				g.genPatternCheck(body, fieldPath, fieldName, `^https?://`, "невалидный URL")
			}

			if fr.Rule.Pattern != "" && fr.Var.Type.IsBasic && fr.Var.Type.BasicInfo == gomosaic.IsString {
				g.genPatternCheck(body, fieldPath, fieldName, fr.Rule.Pattern, "не соответствует шаблону")
			}

			if fr.Rule.MinLen != nil && fr.Var.Type.IsBasic && fr.Var.Type.BasicInfo == gomosaic.IsString {
				body.If(jen.Len(fieldPath.Clone()).Op("<").Lit(*fr.Rule.MinLen)).Block(
					jen.Id(errVar).Op("=").Qual("github.com/hashicorp/go-multierror", "Append").Call(
						jen.Id(errVar),
						jen.Qual("fmt", "Errorf").Call(jen.Lit(fmt.Sprintf("%s: минимальная длина %d", fieldName, *fr.Rule.MinLen))),
					),
				)
			}

			if fr.Rule.MaxLen != nil && fr.Var.Type.IsBasic && fr.Var.Type.BasicInfo == gomosaic.IsString {
				body.If(jen.Len(fieldPath.Clone()).Op(">").Lit(*fr.Rule.MaxLen)).Block(
					jen.Id(errVar).Op("=").Qual("github.com/hashicorp/go-multierror", "Append").Call(
						jen.Id(errVar),
						jen.Qual("fmt", "Errorf").Call(jen.Lit(fmt.Sprintf("%s: максимальная длина %d", fieldName, *fr.Rule.MaxLen))),
					),
				)
			}

			if fr.Rule.Min != nil && isNumericType(fr.Var.Type) {
				body.If(fieldPath.Clone().Op("<").Lit(int(*fr.Rule.Min))).Block(
					jen.Id(errVar).Op("=").Qual("github.com/hashicorp/go-multierror", "Append").Call(
						jen.Id(errVar),
						jen.Qual("fmt", "Errorf").Call(jen.Lit(fmt.Sprintf("%s: минимум %d", fieldName, *fr.Rule.Min))),
					),
				)
			}

			if fr.Rule.Max != nil && isNumericType(fr.Var.Type) {
				body.If(fieldPath.Clone().Op(">").Lit(int(*fr.Rule.Max))).Block(
					jen.Id(errVar).Op("=").Qual("github.com/hashicorp/go-multierror", "Append").Call(
						jen.Id(errVar),
						jen.Qual("fmt", "Errorf").Call(jen.Lit(fmt.Sprintf("%s: максимум %d", fieldName, *fr.Rule.Max))),
					),
				)
			}
		}

		body.Return(jen.Id(errVar))
	})

	return group
}

func (g *generator) genRequiredCheck(body *jen.Group, fieldPath *jen.Statement, t *gomosaic.TypeInfo, fieldName string) {
	if t.IsBasic {
		switch t.BasicInfo {
		case gomosaic.IsString:
			body.If(fieldPath.Clone().Op("==").Lit("")).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit(fieldName + ": обязательное поле"))),
			)
		case gomosaic.IsInteger, gomosaic.IsFloat:
			body.If(fieldPath.Clone().Op("==").Lit(0)).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit(fieldName + ": обязательное поле"))),
			)
		}
	} else if t.IsPtr || t.IsSlice || t.IsMap {
		body.If(fieldPath.Clone().Op("==").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit(fieldName + ": обязательное поле"))),
		)
	}
}

func (g *generator) genPatternCheck(body *jen.Group, fieldPath *jen.Statement, fieldName, pattern, msg string) {
	body.If(
		jen.Op("!").Qual("regexp", "MustCompile").Call(jen.Lit(pattern)).Dot("MatchString").Call(fieldPath.Clone()),
	).Block(
		jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit(fieldName + ": " + msg))),
	)
}

func isNumericType(t *gomosaic.TypeInfo) bool {
	if t.IsBasic && (t.BasicInfo == gomosaic.IsInteger || t.BasicInfo == gomosaic.IsFloat || t.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned) {
		return true
	}
	return false
}

// Ensure compilation
var _ = multierror.Append
var _ gomosaic.Generator = (*Plugin)(nil)
