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

const bodyMaxSize = 10485760 // 10MB

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
		middlewareType := g.strategy.MiddlewareType()
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
		jen.Return(jen.Id("o")),
	).Line()

	for _, m := range methods {
		group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("Middleware" + m.Func.Name).Params(jen.Id("middleware").Op("...").Add(middlewareType)).Op("*").Id(optionsName).Block(
			jen.Return(jen.Id("o")),
		).Line()
	}

	return group
}

func (g *ServerGenerator) genOptionLoader(ifaceName string) jen.Code {
	return jen.Id("o").Op(":=").Op("&").Id(ifaceName + "Options").Values()
}

func (g *ServerGenerator) genBodyDataRead(ifaceOpt *service.IfaceOpt) jen.Code {
	group := jen.NewFile("")
	group.Var().Id("bodyData").Op("=").Make(jen.Index().Byte(), jen.Lit(0), jen.Lit(bodyMaxSize))
	group.Id("buf").Op(":=").Qual("bytes", "NewBuffer").Call(jen.Id("bodyData"))
	group.List(jen.Id("written"), jen.Id("err")).Op(":=").Qual("io", "Copy").Call(jen.Id("buf"), g.strategy.BodyPathParam())

	group.If(
		jen.Err().Op("!=").Nil(),
	).Block(
		g.genErrorEncoderCall(ifaceOpt),
		jen.Return(),
	)

	return group
}

func (g *ServerGenerator) genErrorEncoderCall(ifaceOpt *service.IfaceOpt) jen.Code {
	return jen.Id("errorEncoder"+ifaceOpt.NameTypeInfo.Name).Call(jen.Id(g.strategy.RespArgName()), jen.Err())
}

func (g *ServerGenerator) genNonBodyParamsfunc(methodOpt *service.MethodOpt, params []*service.MethodParamOpt, valueFn func(name string) jen.Code) jen.Code {
	group := jen.NewFile("")

	var transformCodes []jen.Code

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)

		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualifier.Qual))

		transformCodes = append(transformCodes, typetransform.For(p.Var.Type).
			SetAssignID(jen.Id(name)).
			SetValueID(valueFn(strcase.ToLowerCamel(p.Name))).
			SetErrStatements(
				g.genErrorEncoderCall(methodOpt.Iface),
				jen.Return(),
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

	varContentType := "contentType"

	switch httpMethod {
	case "POST", "PUT", "PATCH", "DELETE":

		group.Id(varContentType).Op(":=").Add(g.strategy.HeaderParamValue("content-type"))

		group.If(jen.Id(varContentType).Op("==").Lit("")).Block(
			jen.Id(varContentType).Op("=").Lit(opt.Iface.DefaultContentType),
		)

		group.Id("parts").Op(":=").Qual("strings", "Split").Call(jen.Id(varContentType), jen.Lit(";"))
		group.If(jen.Len(jen.Id("parts")).Op(">").Lit(0)).Block(
			jen.Id(varContentType).Op("=").Id("parts").Index(jen.Lit(0)),
		)

		group.Switch(jen.Id(varContentType)).BlockFunc(func(group *jen.Group) {
			group.Case(jen.Lit("application/json")).BlockFunc(func(group *jen.Group) {
				reqName := "reqBody"

				if len(params) == 1 && opt.Single.Req {
					group.Var().Id(reqName).Add(jenutils.TypeInfoQual(params[0].Var.Type, g.qualifier.Qual))
				} else {
					structFields := service.MakeStructFieldsFromParams(params, g.qualifier.Qual)

					if len(opt.WrapReq.PathParts) > 0 {
						structFields = service.WrapStruct(opt.WrapReq.PathParts, structFields)
					}

					group.Var().Id(reqName).Struct(structFields)
				}

				group.Add(g.genBodyDataRead(opt.Iface))
				group.Err().Op("=").Qual("encoding/json", "Unmarshal").Call(jen.Id("bodyData").Index(jen.Op(":").Id("written")), jen.Op("&").Id(reqName))

				group.If(jen.Err().Op("!=").Nil()).Block(
					g.genErrorEncoderCall(opt.Iface),
					jen.Return(),
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
					group.Add(
						g.strategy.FormParams(
							g.genErrorEncoderCall(opt.Iface),
							jen.Return(),
						),
					)

					for _, p := range params {
						stFldName := strcase.ToLowerCamel(p.Var.Name)
						valueID := g.strategy.FormParam(p.Name)

						assignID := jen.Id(stFldName)

						code := typetransform.For(p.Var.Type).
							SetAssignID(assignID).
							SetValueID(valueID).SetQualFunc(g.qualifier.Qual).
							SetErrStatements(
								g.genErrorEncoderCall(opt.Iface),
								jen.Return(),
							).Parse()

						group.Add(code)
					}
				})

				group.Case(jen.Lit("multipart/form-data")).BlockFunc(func(group *jen.Group) {
					group.Add(g.strategy.MultipartFormParams(opt.MultipartMaxMemory,
						g.genErrorEncoderCall(opt.Iface),
						jen.Return(),
					))

					for _, p := range params {
						stFldName := strcase.ToLowerCamel(p.Var.Name)
						valueID := g.strategy.MultipartFormParam(p.Name)
						code := typetransform.For(p.Var.Type).
							SetAssignID(jen.Id(stFldName)).
							SetValueID(valueID).
							SetErrStatements(
								g.genErrorEncoderCall(opt.Iface),
								jen.Return(),
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
		group.Add(g.strategy.Context())
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
		g.genErrorEncoderCall(m.Iface),
		jen.Return(),
	)

	return group
}

func (g *ServerGenerator) genRegisterHandlers(s *service.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	group.Func().Id(s.NameTypeInfo.Name+"RegisterHandlers").Params(
		jen.Id(g.strategy.LibArgName()).Add(g.strategy.LibType()),
		jen.Id("svc").Do(g.qualifier.Qual(s.NameTypeInfo.Package.Path, s.NameTypeInfo.Name)),
		jen.Id("opt").Op("*").Id(s.NameTypeInfo.Name+"Options"),
	).BlockFunc(func(group *jen.Group) {
		group.Add(g.genOptionLoader(s.NameTypeInfo.Name))
		for _, m := range s.Methods {
			middlewares := jen.Append(jen.Id("o").Dot("middleware"), jen.Id("o").Dot("middleware"+m.Func.Name).Op("...")).Op("...")

			pathParts := strings.Split(m.Path, "/")
			for _, pp := range m.PathParams {
				pathParts[pp.PathParamIndex] = g.strategy.PathParamWrap(pp.PathParamName)
			}

			group.Add(g.strategy.HandlerFunc(m.Method, strings.Join(pathParts, "/"), middlewares, func(group *jen.Group) {
				if len(m.Params) > 0 {
					if len(m.BodyParams) > 0 {
						group.Add(g.genHandlerDecodeBodyParams(m, m.BodyParams))
					}
					if len(m.HeaderParams) > 0 {
						group.Add(g.genNonBodyParamsfunc(m, m.HeaderParams, g.strategy.HeaderParamValue))
					}
					// if len(cookieParams) > 0 {
					// group.Add(g.genNonBodyParamsfunc(cookieParams, strategy.QueryParamValue, strategy))
					// }
					if len(m.QueryParams) > 0 {
						group.Add(g.strategy.QueryParams())

						group.Add(g.genNonBodyParamsfunc(m, m.QueryParams, g.strategy.QueryParamValue))
					}
					if len(m.PathParams) > 0 {
						group.Add(g.genNonBodyParamsfunc(m, m.PathParams, g.strategy.PathParamValue))
					}
				}

				group.Add(g.genCallServiceMethod(m))

				respName := "resp"

				// if len(m.BodyResults) > 0 {
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

				group.Add(g.genBodyResultWrite(m, respName))
				// }
			}))
		}
	})

	return group
}

func (g *ServerGenerator) genBodyResultWrite(m *service.MethodOpt, respName string) jen.Code {
	group := jen.NewFile("")

	group.Var().Id("dataBytes").Index().Byte()

	group.Id("acceptHeader").Op(":=").Add(g.strategy.HeaderParamValue("accept"))
	group.Id("ah").Op(":=").Qual(service.MimeheaderPkg, "ParseAcceptHeader").Call(jen.Id("acceptHeader"))

	group.List(jen.Id("_"), jen.Id("mtype"), jen.Id("_")).Op(":=").Id("ah").Dot("Negotiate").Call(
		jen.Index().String().Values(
			jen.Lit("text/html"),
			jen.Lit("application/json"),
		),
		jen.Lit("application/json"),
	)

	group.Switch(jen.Id("mtype")).BlockFunc(func(group *jen.Group) {
		group.Default().BlockFunc(func(group *jen.Group) {
			group.If(
				jen.List(jen.Id("t"), jen.Id("ok")).Op(":=").Any().Call(jen.Id(respName)).Assert(
					jen.Interface(jen.Id("Bytes").Params(jen.Id("string")).Params(jen.Index().Id("byte"), jen.Id("bool"))),
				), jen.Id("ok"),
			).Block(
				jen.If(
					jen.List(jen.Id("bytes"), jen.Id("ok")).Op(":=").Id("t").Dot("Bytes").Call(jen.Id("mtype")), jen.Id("ok"),
				).BlockFunc(func(group *jen.Group) {
					group.Add(g.strategy.SetHeader(jen.Lit("content-type"), jen.Id("mtype")))
					group.Add(g.strategy.WriteBody(jen.Id("bytes"), jen.Qual(service.HTTPPkg, "StatusOK")))
					group.Return()
				}),
			)
			group.Add(g.strategy.WriteBody(nil, jen.Qual(service.HTTPPkg, "StatusNotAcceptable")))
			group.Return()
		})
		group.Case(jen.Lit("application/json")).Block(
			jen.Id("w").Dot("Header").Call().Dot("Set").Call(jen.Lit("content-type"), jen.Lit("application/json")),
			jen.List(jen.Id("dataBytes"), jen.Id("err")).Op("=").Qual("encoding/json", "Marshal").Call(jen.Id(respName)),
			jen.If(jen.Id("err").Op("!=").Id("nil")).Block(
				g.genErrorEncoderCall(m.Iface),
				jen.Return(),
			),
			jen.Add(g.strategy.WriteBody(jen.Id("dataBytes"), jen.Lit(200))), //nolint: mnd
			jen.Return(),
		)

		if m.Templ.Path != "" {
			group.Case(jen.Lit("text/html")).BlockFunc(func(group *jen.Group) {
				var callParams []jen.Code

				if len(m.BodyResults) == 1 && m.Single.Resp {
					callParams = append(callParams, jen.Id(respName))
				} else {
					for _, p := range m.Templ.Params {
						callParams = append(callParams, jen.Id("resp").Dot(strcase.ToCamel(p)))
					}
				}

				group.Qual(service.TemplPkg, "Handler").Call(
					jen.Do(g.qualifier.Qual(m.Templ.PkgPath, m.Templ.FuncName)).Call(callParams...),
				).Dot("ServeHTTP").Call(
					jen.Id("w"),
					jen.Id("r"),
				)

				group.Return()
			})
		}
	})

	return group
}

func (g *ServerGenerator) genErrorEncoder(services []*service.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	group.Func().Id("errorEncoder").Params(
		jen.Id(g.strategy.RespArgName()).Add(g.strategy.RespType()),
		jen.Err().Error(),
	).BlockFunc(func(group *jen.Group) {
		group.Var().Id("statusCode").Int().Op("=").Qual("net/http", "StatusInternalServerError")
		group.If(jen.List(jen.Id("e"), jen.Id("ok")).Op(":=").Err().Assert(jen.Interface(jen.Id("StatusCode").Params().Int())), jen.Id("ok")).Block(
			jen.Id("statusCode").Op("=").Id("e").Dot("StatusCode").Call(),
		)
		group.If(jen.List(jen.Id("headerer"), jen.Id("ok")).Op(":=").Err().Assert(jen.Interface(jen.Id("Headers").Params().Qual("net/http", "Header"))), jen.Id("ok")).Block(
			jen.For(jen.List(jen.Id("k"), jen.Id("values"))).Op(":=").Range().Id("headerer").Dot("Headers").Call().Block(
				jen.For(jen.List(jen.Id("_"), jen.Id("v"))).Op(":=").Range().Id("values").Block(
					jen.Add(g.strategy.SetHeader(jen.Id("k"), jen.Id("v"))),
				),
			),
		)

		group.Add(g.strategy.WriteBody(nil, jen.Id("statusCode")))
	})

	group.Line()

	for _, s := range services {
		group.Func().Id("errorEncoder"+s.NameTypeInfo.Name).Params(
			jen.Id(g.strategy.RespArgName()).Add(g.strategy.RespType()),
			jen.Err().Error(),
		).BlockFunc(func(group *jen.Group) {
			var (
				ifaceMethods []jen.Code
				structFields []jen.Code
			)

			for _, e := range s.Errors {
				if e.StatusCode {
					ifaceMethods = append(ifaceMethods, jen.If(jen.List(jen.Id("t"), jen.Id("ok")).Op(":=").Err().Assert(jen.Interface(jen.Id(e.MethodName).Params().Params(jen.Id(e.Type)))), jen.Id("ok")).Block(
						jen.Id("w").Dot("WriteHeader").Call(jen.Id("t").Dot(e.MethodName).Call()),
					).Line())
				} else {
					structFields = append(structFields, jen.Id(e.FldName).Id(e.Type).Tag(map[string]string{"json": e.TagName}))
					ifaceMethods = append(ifaceMethods, jen.If(jen.List(jen.Id("t"), jen.Id("ok")).Op(":=").Err().Assert(jen.Interface(jen.Id(e.MethodName).Params().Params(jen.Id(e.Type)))), jen.Id("ok")).Block(
						jen.Id("body").Dot(e.FldName).Op("=").Id("t").Dot(e.MethodName).Call(),
					).Line())
				}
			}

			group.Id("errorEncoder").Call(jen.Id(g.strategy.RespArgName()), jen.Err())

			group.Var().Id("body").Struct(structFields...)

			group.Add(ifaceMethods...)

			group.Id("_").Op("=").Qual(service.JSONPkg, "NewEncoder").Call(jen.Id("w")).Dot("Encode").Call(jen.Id("body"))
		})
	}

	return group
}

func (g *ServerGenerator) Generate(services []*service.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")
	group.Add(g.genServiceOptions(services))
	for _, s := range services {
		group.Add(g.genRegisterHandlers(s))
	}
	group.Add(g.genErrorEncoder(services))

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
