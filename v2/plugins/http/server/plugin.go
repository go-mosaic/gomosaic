package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// Plugin — HTTP-серверный плагин v2.
// Конфигурируется через Strategy (Chi, Echo, и т.д.).
type Plugin struct {
	strategy Strategy
}

// NewPlugin создает новый серверный плагин.
func NewPlugin(strategy Strategy) *Plugin {
	return &Plugin{strategy: strategy}
}

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return p.strategy.Name() }

// Generate генерирует код HTTP-сервера.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &ServerGenerator{
		module:   module,
		strategy: p.strategy,
		qual:     f.Qual,
	}

	code, err := gen.Generate(annotations)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		f.Add(code)
	}

	filename := fmt.Sprintf("server_%s_gen.go", strings.TrimPrefix(p.strategy.Name(), "http-server-"))
	return map[string]gomosaic.File{filename: f}, errs
}

// ServerGenerator генерирует код HTTP-сервера.
type ServerGenerator struct {
	module   *gomosaic.ModuleInfo
	strategy Strategy
	qual     gomosaic.QualFunc
}

// Generate генерирует весь код сервера.
func (g *ServerGenerator) Generate(services []*annotation.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")
	group.Add(g.genServiceOptions(services))
	for _, s := range services {
		group.Add(g.genRegisterHandlers(s))
	}
	return group, nil
}

func (g *ServerGenerator) genServiceOptions(services []*annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")
	for _, s := range services {
		middlewareType := jen.Qual(gomosaic.TransportPkg, "Middleware")
		optionsName := s.NameTypeInfo.Name + "Options"
		group.Add(g.genTypeOptions(optionsName, middlewareType, s.Methods))
	}
	return group
}

func (g *ServerGenerator) genTypeOptions(optionsName string, middlewareType jen.Code, methods []*annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")
	transportOptions := jen.Do(g.qual(gomosaic.TransportPkg, "TransportOption"))

	group.Type().Id(optionsName).StructFunc(func(group *jen.Group) {
		group.Id("transportOptions").Index().Add(transportOptions)
		group.Id("middleware").Index().Add(middlewareType)
		for _, m := range methods {
			group.Id("middleware" + m.Func.Name).Index().Add(middlewareType)
		}
	})

	group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("TransportOptions").Params(jen.Id("opts").Op("...").Add(transportOptions)).Op("*").Id(optionsName).Block(
		jen.Id("o").Dot("transportOptions").Op("=").Append(jen.Id("o").Dot("transportOptions"), jen.Id("opts").Op("...")),
		jen.Return(jen.Id("o")),
	).Line()

	group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("Middleware").Params(jen.Id("middleware").Op("...").Add(middlewareType)).Op("*").Id(optionsName).Block(
		jen.Id("o").Dot("middleware").Op("=").Append(jen.Id("o").Dot("middleware"), jen.Id("middleware").Op("...")),
		jen.Return(jen.Id("o")),
	).Line()

	for _, m := range methods {
		group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("Middleware"+m.Func.Name).Params(jen.Id("middleware").Op("...").Add(middlewareType)).Op("*").Id(optionsName).Block(
			jen.Id("o").Dot("middleware"+m.Func.Name).Op("=").Append(jen.Id("o").Dot("middleware"+m.Func.Name), jen.Id("middleware").Op("...")),
			jen.Return(jen.Id("o")),
		).Line()
	}

	return group
}

func (g *ServerGenerator) genRegisterHandlers(s *annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	group.Func().Id(s.NameTypeInfo.Name+"RegisterHandlers").Params(
		jen.Id("router").Do(func(s *jen.Statement) {
			if g.strategy.UsePtrType() {
				s.Op("*")
			}
		}).Qual(g.strategy.RouterPkg(), g.strategy.RouterType()),
		jen.Id("svc").Do(g.qual(s.NameTypeInfo.Package.Path, s.NameTypeInfo.Name)),
		jen.Id("opt").Op("*").Id(s.NameTypeInfo.Name+"Options"),
	).BlockFunc(func(group *jen.Group) {
		group.If(jen.Id("opt").Op("==").Nil()).Block(
			jen.Id("opt").Op("=").Op("&").Id(s.NameTypeInfo.Name + "Options").Values(),
		)

		group.Id("transportFactory").Op(":=").Qual(gomosaic.TransportFactoryPkg, "NewFactory").Call(
			jen.Id("opt").Dot("transportOptions").Op("..."),
		)

		group.List(jen.Id("tr"), jen.Err()).Op(":=").Id("transportFactory").Dot("Create").Call(
			jen.Qual(gomosaic.TransportFactoryPkg, g.strategy.TransportFactoryType()),
			jen.Id("router"),
		)
		group.If(jen.Err().Op("!=").Nil()).Block(jen.Panic(jen.Err()))

		for _, m := range s.Methods {
			pathParts := strings.Split(m.Path, "/")
			for _, pp := range m.PathParams {
				pathParts[pp.PathParamIndex] = g.strategy.PathParamWrap(pp.PathParamName)
			}

			group.Id("tr").Dot("AddRoute").Call(
				jen.Lit(m.Method),
				jen.Lit(strings.Join(pathParts, "/")),
				jen.Func().Params(
					jen.Id("req").Qual(gomosaic.TransportPkg, "Request"),
					jen.Id("resp").Qual(gomosaic.TransportPkg, "Response"),
				).Error().BlockFunc(func(group *jen.Group) {
					if len(m.BodyParams) > 0 {
						group.Add(g.genDecodeBodyParams(m, m.BodyParams))
					}
					if len(m.HeaderParams) > 0 {
						group.Add(g.genNonBodyParams(m.HeaderParams, func(name string) jen.Code {
							return jen.Id("req").Dot("Header").Call(jen.Lit(name))
						}))
					}
					if len(m.QueryParams) > 0 {
						group.Add(g.genQueryParams(m.QueryParams))
					}
					if len(m.PathParams) > 0 {
						group.Add(g.genNonBodyParams(m.PathParams, func(name string) jen.Code {
							return jen.Id("req").Dot("PathValue").Call(jen.Lit(name))
						}))
					}

					group.Add(g.genCallServiceMethod(m))

					respName := "respData"
					if len(m.BodyResults) == 1 && m.Single.Resp {
						respName = m.BodyResults[0].Var.Name
					} else {
						structFields := annotation.MakeStructFieldsFromResults(m.BodyResults, g.qual)
						if len(m.WrapResp.PathParts) > 0 {
							structFields = annotation.WrapStruct(m.WrapResp.PathParts, structFields)
						}
						group.Var().Id(respName).Struct(structFields)
						for _, result := range m.BodyResults {
							group.Id(respName).Do(func(s *jen.Statement) {
								for _, name := range m.WrapResp.PathParts {
									s.Dot(strcase.ToCamel(name))
								}
							}).Dot(strcase.ToCamel(result.Var.Name)).Op("=").Id(result.Var.Name)
						}
					}

					group.Id("resp").Dot("WriteData").Call(jen.Id("req"), jen.Id(respName))
					group.Return(jen.Nil())
				}),
				jen.Append(jen.Id("opt").Dot("middleware"), jen.Id("opt").Dot("middleware"+m.Func.Name).Op("...")).Op("..."),
			)
		}
	})

	return group
}

func (g *ServerGenerator) genDecodeBodyParams(opt *annotation.MethodOpt, params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("")

	for _, p := range params {
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
	}

	httpMethod := strings.ToUpper(opt.Method)
	reqName := "reqBody"
	varContentType := "contentType"

	switch httpMethod {
	case "POST", "PUT", "PATCH", "DELETE":
		group.Id(varContentType).Op(":=").Id("req").Dot("Header").Call(jen.Lit("Content-Type"))
		group.If(jen.Id(varContentType).Op("==").Lit("")).Block(
			jen.Id(varContentType).Op("=").Lit(opt.Default.ContentType),
		)
		group.Id("parts").Op(":=").Qual("strings", "Split").Call(jen.Id(varContentType), jen.Lit(";"))
		group.If(jen.Len(jen.Id("parts")).Op(">").Lit(0)).Block(
			jen.Id(varContentType).Op("=").Id("parts").Index(jen.Lit(0)),
		)

		group.Switch(jen.Id(varContentType)).BlockFunc(func(group *jen.Group) {
			group.Case(jen.Lit("application/json")).BlockFunc(func(group *jen.Group) {
				if len(params) == 1 && opt.Single.Req {
					group.Var().Id(reqName).Add(jenutils.TypeInfoQual(params[0].Var.Type, g.qual))
				} else {
					structFields := annotation.MakeStructFieldsFromParams(params, g.qual)
					if len(opt.WrapReq.PathParts) > 0 {
						structFields = annotation.WrapStruct(opt.WrapReq.PathParts, structFields)
					}
					group.Var().Id(reqName).Struct(structFields)
				}

				group.If(jen.Err().Op(":=").Id("req").Dot("ReadData").Call(jen.Op("&").Id(reqName)), jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Err()),
				)

				if len(params) == 1 && opt.Single.Req {
					group.Id(strcase.ToLowerCamel(params[0].Var.Name)).Op("=").Id(reqName)
				} else {
					for _, p := range params {
						group.Id(strcase.ToLowerCamel(p.Var.Name)).Op("=").Id(reqName).Add(annotation.Dot(append(opt.WrapReq.PathParts, strcase.ToCamel(p.Var.Name))...))
					}
				}
			})
		})
	}

	return group
}

func (g *ServerGenerator) genQueryParams(params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("")
	group.Id("q").Op(":=").Id("req").Dot("Queries").Call()

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)
		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

		queryName := p.Name
		if p.NameOpt.Value != "" {
			queryName = p.NameOpt.Value
		}

		valueID := jen.Id("q").Dot("Get").Call(jen.Lit(queryName))
		group.If(
			jen.Err().Op(":=").Do(g.qual("github.com/go-mosaic/runtime", "ParseQueryParam")).Call(valueID, jen.Op("&").Add(jen.Id(name))),
			jen.Err().Op("!=").Nil(),
		).Block(jen.Return(jen.Err()))
	}

	return group
}

func (g *ServerGenerator) genNonBodyParams(params []*annotation.MethodParamOpt, valueFn func(name string) jen.Code) jen.Code {
	group := jen.NewFile("")

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)
		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

		valueName := strcase.ToLowerCamel(p.Name)
		if p.NameOpt.Value != "" {
			valueName = p.NameOpt.Value
		}

		group.If(
			jen.Err().Op(":=").Do(g.qual("github.com/go-mosaic/runtime", "ParseHeaderParam")).Call(valueFn(valueName), jen.Op("&").Add(jen.Id(name))),
			jen.Err().Op("!=").Nil(),
		).Block(jen.Return(jen.Err()))
	}

	return group
}

func (g *ServerGenerator) genCallServiceMethod(m *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	svcCall := jen.Do(func(s *jen.Statement) {
		s.ListFunc(func(group *jen.Group) {
			for _, r := range m.Results {
				group.Id(strcase.ToLowerCamel(r.Var.Name))
			}
		})
		if len(m.Results) > 0 {
			s.Op(":=")
		} else {
			s.Op("=")
		}
	}).Id("svc").Dot(m.Func.Name).CallFunc(func(group *jen.Group) {
		group.Id("req").Dot("Context").Call()
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			switch p.HTTPType {
			case annotation.PathHTTPType, annotation.CookieHTTPType, annotation.QueryHTTPType, annotation.HeaderHTTPType:
				group.Id("param" + strcase.ToCamel(p.Var.Name))
			default:
				group.Id(strcase.ToLowerCamel(p.Var.Name))
			}
		}
	})

	group.Add(svcCall)
	group.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err()))

	return group
}

// Ensure Plugin implements gomosaic.Generator.
var _ gomosaic.Generator = (*Plugin)(nil)
