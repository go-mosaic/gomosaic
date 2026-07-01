// Package openapi предоставляет плагин для генерации OpenAPI 3.0 документации.
//
// На основе HTTP-аннотаций генерирует openapi.json со spec-endpoint'ами.
//
// Пример:
//
//	// @openapi-tags users
//	// @openapi-title Пользователи
//	type UserService interface {
//	    // @http-method GET
//	    // @http-path /users
//	    // @openapi-summary Получить список пользователей
//	    // @openapi-descr Возвращает список всех пользователей
//	    ListUsers(ctx context.Context) ([]User, error)
//
//	    // @http-method POST
//	    // @http-path /users
//	    // @openapi-summary Создать пользователя
//	    CreateUser(ctx context.Context, user *User) (error)
//	}
//
// Генерирует файл openapi_gen.go с Handler для отдачи JSON-спецификации.
package openapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// Plugin — плагин генерации OpenAPI-документации.
type Plugin struct{}

// NewPlugin создает новый экземпляр плагина.
func NewPlugin() *Plugin { return &Plugin{} }

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return "openapi" }

// Generate генерирует OpenAPI-спецификацию.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, err error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	if len(annotations) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := &generator{qual: f.Qual, module: module}
	f.Add(gen.generateSpec(annotations))

	return map[string]gomosaic.File{"openapi_gen.go": f}, nil
}

type generator struct {
	qual   gomosaic.QualFunc
	module *gomosaic.ModuleInfo
}

func (g *generator) generateSpec(services []*annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	// OpenAPI Handler
	group.Comment("OpenAPIHandler возвращает HTTP-хендлер для OpenAPI-спецификации.")
	group.Func().Id("OpenAPIHandler").Params().Qual("net/http", "Handler").BlockFunc(func(body *jen.Group) {
		body.Var().Id("spec").Op("=").Map(jen.String()).Interface().ValuesFunc(func(dict *jen.Group) {
			dict.Add(jen.Lit("openapi")).Op(":").Add(jen.Lit("3.0.3"))
			dict.Add(jen.Lit("info")).Op(":").Add(jen.Map(jen.String()).Interface().Values(jen.Dict{
				jen.Lit("title"):   jen.Lit(g.module.Path),
				jen.Lit("version"): jen.Lit("1.0.0"),
			}))

			// Paths
			for _, svc := range services {
				for _, m := range svc.Methods {
					pathItem := g.generatePathItem(svc, m)
					if pathItem != nil {
						dict.Add(jen.Lit(m.Path)).Op(":").Add(pathItem)
					}
				}
			}
		})

		body.Return(
			jen.Qual("net/http", "HandlerFunc").Call(
				jen.Func().Params(
					jen.Id("w").Qual("net/http", "ResponseWriter"),
					jen.Id("r").Op("*").Qual("net/http", "Request"),
				).Block(
					jen.Id("w").Dot("Header").Call().Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json")),
					jen.List(jen.Id("data"), jen.Id("_")).Op(":=").Qual("encoding/json", "MarshalIndent").Call(jen.Id("spec"), jen.Lit(""), jen.Lit("  ")),
					jen.Id("w").Dot("Write").Call(jen.Id("data")),
				),
			),
		)
	})

	return group
}

func (g *generator) generatePathItem(svc *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	if m.Path == "" || m.Method == "" {
		return nil
	}

	methodName := strcase.ToLowerCamel(m.Func.Name)
	tags := []jen.Code{}
	if ann, ok := svc.NameTypeInfo.Annotations.Get("openapi-tags"); ok {
		for _, t := range ann.Params {
			tags = append(tags, jen.Lit(t))
		}
	}
	if tags == nil {
		tags = append(tags, jen.Lit(svc.NameTypeInfo.Name))
	}

	summary := fmt.Sprintf("%s %s", m.Func.Name, svc.NameTypeInfo.Name)

	return jen.Map(jen.String()).Interface().Values(jen.DictFunc(func(d jen.Dict) {
		d[jen.Lit(strings.ToLower(m.Method))] = jen.Map(jen.String()).Interface().Values(jen.Dict{
			jen.Lit("summary"):     jen.Lit(summary),
			jen.Lit("operationId"): jen.Lit(methodName),
			jen.Lit("tags"):        jen.Index().String().Values(tags...),
			jen.Lit("parameters"):  g.generateParameters(m),
			jen.Lit("responses"):   g.generateResponses(m),
		})
	}))
}

func (g *generator) generateParameters(m *annotation.MethodOpt) jen.Code {
	params := jen.Index().Interface()
	for _, p := range m.PathParams {
		params.Values(jen.Map(jen.String()).Interface().Values(jen.Dict{
			jen.Lit("name"):     jen.Lit(p.PathParamName),
			jen.Lit("in"):       jen.Lit("path"),
			jen.Lit("required"): jen.Lit(true),
			jen.Lit("schema"):   g.typeToSchema(p.Var.Type),
		}))
	}
	for _, p := range m.QueryParams {
		params.Values(jen.Map(jen.String()).Interface().Values(jen.Dict{
			jen.Lit("name"):     jen.Lit(p.Name),
			jen.Lit("in"):       jen.Lit("query"),
			jen.Lit("required"): jen.Lit(p.Required),
			jen.Lit("schema"):   g.typeToSchema(p.Var.Type),
		}))
	}
	return params
}

func (g *generator) generateResponses(m *annotation.MethodOpt) jen.Code {
	responses := jen.Map(jen.String()).Interface().Values(jen.Dict{
		jen.Lit("200"): jen.Map(jen.String()).Interface().Values(jen.Dict{
			jen.Lit("description"): jen.Lit("Успешный ответ"),
		}),
	})
	return responses
}

func (g *generator) typeToSchema(t *gomosaic.TypeInfo) jen.Code {
	schema := jen.Map(jen.String()).Interface().Values(jen.Dict{})

	if t.IsBasic {
		switch t.BasicInfo {
		case gomosaic.IsString:
			return jen.Map(jen.String()).Interface().Values(jen.Dict{
				jen.Lit("type"): jen.Lit("string"),
			})
		case gomosaic.IsInteger:
			return jen.Map(jen.String()).Interface().Values(jen.Dict{
				jen.Lit("type"):   jen.Lit("integer"),
				jen.Lit("format"): jen.Lit("int32"),
			})
		case gomosaic.IsFloat:
			return jen.Map(jen.String()).Interface().Values(jen.Dict{
				jen.Lit("type"):   jen.Lit("number"),
				jen.Lit("format"): jen.Lit("float"),
			})
		case gomosaic.IsBoolean:
			return jen.Map(jen.String()).Interface().Values(jen.Dict{
				jen.Lit("type"): jen.Lit("boolean"),
			})
		}
	}
	return schema
}

var _ gomosaic.Generator = (*Plugin)(nil)
