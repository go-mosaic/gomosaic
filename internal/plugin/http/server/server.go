package server

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
	"github.com/go-mosaic/gomosaic/pkg/typetransform"
)

type Qualifier interface {
	Qual(pkgPath, name string) func(s *jen.Statement)
}

type ServerGenerator struct {
	module    *gomosaic.ModuleInfo
	strategy  Strategy
	qualifier Qualifier
}

func (g *ServerGenerator) genServiceOptions(services []*service.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	for _, s := range services {
		middlewareType := jen.Qual(service.RuntimeTransport, "Middleware")
		optionsName := s.NameTypeInfo.Name + "Options"
		group.Add(g.genTypeOptions(optionsName, middlewareType, s.Methods))
	}

	return group
}

func (g *ServerGenerator) genTypeOptions(optionsName string, middlewareType jen.Code, methods []*service.MethodOpt) jen.Code {
	group := jen.NewFile("")

	group.Type().Id(optionsName).StructFunc(func(group *jen.Group) {
		group.Id("middleware").Index().Add(middlewareType)
		for _, m := range methods {
			group.Id("middleware" + m.Func.Name).Index().Add(middlewareType)
		}
	})

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

func (g *ServerGenerator) genOptionLoader(ifaceName string) jen.Code {
	return jen.Id("o").Op(":=").Op("&").Id(ifaceName + "Options").Values()
}

func (g *ServerGenerator) genNonBodyParamsFunc(
	params []*service.MethodParamOpt,
	valueFn func(name string) jen.Code,
) jen.Code {
	group := jen.NewFile("")

	var transformCodes []jen.Code

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)

		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualifier.Qual))

		transformCodes = append(transformCodes, typetransform.For(p.Var.Type).
			SetAssignID(jen.Id(name)).
			SetValueID(valueFn(strcase.ToLowerCamel(p.Name))).
			SetErrStatements(
				jen.Return(jen.Err()),
			).Parse(),
		)
	}

	for _, c := range transformCodes {
		group.Add(c)
	}

	return group
}

func (g *ServerGenerator) genHandlerDecodeBodyParams(
	opt *service.MethodOpt,
	params []*service.MethodParamOpt,
) jen.Code {
	group := jen.NewFile("")

	for _, p := range params {
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualifier.Qual))
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
					group.Var().Id(reqName).Add(jenutils.TypeInfoQual(params[0].Var.Type, g.qualifier.Qual))
				} else {
					structFields := service.MakeStructFieldsFromParams(params, g.qualifier.Qual)

					if len(opt.WrapReq.PathParts) > 0 {
						structFields = service.WrapStruct(opt.WrapReq.PathParts, structFields)
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
						group.Id(strcase.ToLowerCamel(p.Var.Name)).Op("=").Id(reqName).Add(service.Dot(append(opt.WrapReq.PathParts, strcase.ToCamel(p.Var.Name))...))
					}
				}
			})

			if !service.IsObjectType(opt.BodyParams[0].Var.Type) {
				group.Case(jen.Lit("application/x-www-form-urlencoded")).BlockFunc(func(group *jen.Group) {
					group.List(jen.Id("form"), jen.Err()).Op(":=").Id("req").Dot("URLEncodedForm").Call()
					group.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.Err()),
					)

					for _, p := range params {
						stFldName := strcase.ToLowerCamel(p.Var.Name)
						valueID := jen.Id("form").Dot("Get").Call(jen.Lit(p.Name))

						assignID := jen.Id(stFldName)

						code := typetransform.For(p.Var.Type).
							SetAssignID(assignID).
							SetValueID(valueID).SetQualFunc(g.qualifier.Qual).
							SetErrStatements(
								jen.Return(jen.Err()),
							).Parse()

						group.Add(code)
					}
				})

				group.Case(jen.Lit("multipart/form-data")).BlockFunc(func(group *jen.Group) {
					group.List(jen.Id("form"), jen.Err()).Op(":=").Id("req").Dot("MultipartForm").Call(jen.Lit(opt.FormMaxMemory))
					group.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.Err()),
					)

					for _, p := range params {
						stFldName := strcase.ToLowerCamel(p.Var.Name)
						valueID := jen.Id("form").Dot("FormValue").Call(jen.Lit(p.Name))
						code := typetransform.For(p.Var.Type).
							SetAssignID(jen.Id(stFldName)).
							SetValueID(valueID).
							SetErrStatements(
								jen.Return(jen.Err()),
							).Parse()

						group.Add(code)
					}
				})
			}
		})
	}

	return group
}

func (g *ServerGenerator) genCallServiceMethod(m *service.MethodOpt) jen.Code {
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
			default:
				group.Id(strcase.ToLowerCamel(p.Var.Name))
			case service.PathHTTPType, service.CookieHTTPType, service.QueryHTTPType:
				group.Id("param" + strcase.ToCamel(p.Var.Name))
			}
		}
	})

	group.Add(svcCall)

	group.If(jen.Err().Op("!=").Nil()).Block(
		// g.genErrorEncoderCall(m),
		jen.Return(jen.Err()),
	)

	return group
}

func (g *ServerGenerator) genRegisterHandlers(s *service.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	group.Func().Id(s.NameTypeInfo.Name+"RegisterHandlers").Params(
		jen.Id("router").Do(func(s *jen.Statement) {
			if g.strategy.UsePtrType() {
				s.Op("*")
			}
		}).Qual(
			g.strategy.Pkg(), g.strategy.Type(),
		),
		jen.Id("svc").Do(g.qualifier.Qual(s.NameTypeInfo.Package.Path, s.NameTypeInfo.Name)),
		jen.Id("opt").Op("*").Id(s.NameTypeInfo.Name+"Options"),
	).BlockFunc(func(group *jen.Group) {
		group.Add(g.genOptionLoader(s.NameTypeInfo.Name))

		group.Id("tr").Op(":=").Qual(g.strategy.TransportPkg(), g.strategy.TransportConstruct()).Call(jen.Id("router"))
		group.Id("tr").Dot("Use").Call(jen.Id("o").Dot("middleware").Op("..."))

		for _, m := range s.Methods {
			pathParts := strings.Split(m.Path, "/")
			for _, pp := range m.PathParams {
				pathParts[pp.PathParamIndex] = g.strategy.PathParamWrap(pp.PathParamName)
			}

			group.Id("tr").Dot("AddRoute").Call(
				jen.Lit(m.Method),
				jen.Lit(strings.Join(pathParts, "/")),
				jen.Func().Params(
					jen.Id("req").Id("transport").Dot("Request"),
					jen.Id("resp").Id("transport").Dot("Response"),
				).Error().BlockFunc(func(group *jen.Group) {
					if len(m.Params) > 0 {
						if len(m.BodyParams) > 0 {
							group.Add(g.genHandlerDecodeBodyParams(m, m.BodyParams))
						}

						if len(m.HeaderParams) > 0 {
							group.Add(g.genNonBodyParamsFunc(m.HeaderParams, func(name string) jen.Code {
								return jen.Id("req").Dot("Header").Call(jen.Lit(name))
							}))
						}

						// TODO: cookie

						if len(m.QueryParams) > 0 {
							group.Id("q").Op(":=").Id("req").Dot("Queries").Call()

							group.Add(g.genNonBodyParamsFunc(m.QueryParams, func(name string) jen.Code {
								return jen.Id("q").Dot("Get").Call(jen.Lit(name))
							}))
						}

						if len(m.PathParams) > 0 {
							group.Add(g.genNonBodyParamsFunc(m.PathParams, func(name string) jen.Code {
								return jen.Id("req").Dot("PathValue").Call(jen.Lit(name))
							}))
						}
					}

					group.Add(g.genCallServiceMethod(m))

					respName := "respData"

					if len(m.BodyResults) == 1 && m.Single.Resp {
						respName = m.BodyResults[0].Var.Name
					} else {
						structFields := service.MakeStructFieldsFromResults(m.BodyResults, g.qualifier.Qual)

						if len(m.WrapResp.PathParts) > 0 {
							structFields = service.WrapStruct(m.WrapResp.PathParts, structFields)
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

					group.Id("resp").Dot("WriteData").Call(
						jen.Id("req"),
						jen.Id(respName),
					)

					group.Return(jen.Nil())
				}),
				jen.Id("o").Dot("middleware"+m.Func.Name).Op("..."),
			)
		}
	})

	return group
}

func (g *ServerGenerator) Generate(services []*service.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")
	group.Add(g.genServiceOptions(services))
	for _, s := range services {
		group.Add(g.genRegisterHandlers(s))
	}

	return group, nil
}

func NewServer(
	strategy Strategy,
	module *gomosaic.ModuleInfo,
	qualifier Qualifier,
) *ServerGenerator {
	return &ServerGenerator{
		strategy:  strategy,
		module:    module,
		qualifier: qualifier,
	}
}
