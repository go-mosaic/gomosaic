package client

import (
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/gen/structure"
	"github.com/go-mosaic/gomosaic/internal/plugin/http/annotation"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
	"github.com/go-mosaic/gomosaic/pkg/typetransform"
)

const httpStatusLastSuccessCode = 399

const (
	clientOptionName = "clientOptions"
	recvName         = "r"
)

type Qualifier interface {
	Qual(pkgPath, name string) func(s *jen.Statement)
}

type ClientGenerator struct {
	qualifier    Qualifier
	rewriteTypes map[string]bool
}

func NewClientGenerator(qualifier Qualifier) *ClientGenerator {
	return &ClientGenerator{
		qualifier:    qualifier,
		rewriteTypes: map[string]bool{},
	}
}

func (g *ClientGenerator) qual(pkgPath, name string) func(s *jen.Statement) {
	return func(s *jen.Statement) {
		typeName := pkgPath + "." + name
		if g.rewriteTypes[typeName] {
			s.Id(name)
		} else {
			g.qualifier.Qual(pkgPath, name)(s)
		}
	}
}

func (g *ClientGenerator) Generate(services []*annotation.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")

	group.Add(g.genTypes())

	structGen := structure.NewGenerator(group)

	for _, s := range services {
		if s.CopyDTOTypes {
			group.Add(g.genClientDTO(structGen, s))
		}
	}

	g.rewriteTypes = structGen.Processed()

	for _, s := range services {
		group.Add(g.genClientStruct(s))
		group.Add(g.genClientConstruct(s))
		group.Add(g.genClientEndpoints(s))
	}

	return group, nil
}

func (g *ClientGenerator) genReqBodyStruct(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if len(methodOpt.BodyParams) == 1 && methodOpt.Single.Req {
		fldName := strcase.ToLowerCamel(methodOpt.BodyParams[0].Var.Name)
		fldParam := jen.Id(recvName).Dot("params").Dot(fldName)

		group.Return(fldParam)
	} else {
		if len(methodOpt.WrapReq.PathParts) > 0 {
			group.Var().Id("body").Struct(annotation.WrapStruct(methodOpt.WrapReq.PathParts, annotation.MakeStructFieldsFromParams(methodOpt.BodyParams, g.qual))).Line()
		} else {
			group.Var().Id("body").Struct(annotation.MakeStructFieldsFromParams(methodOpt.BodyParams, g.qual)).Line()
		}

		for _, param := range methodOpt.BodyParams {
			if param.Var.IsContext {
				continue
			}
			fldName := strcase.ToLowerCamel(param.Var.Name)

			fldAssign := jen.Do(func(s *jen.Statement) {
				s.Id("body")
				for _, name := range methodOpt.WrapReq.PathParts {
					s.Dot(strcase.ToCamel(name))
				}

				fldParam := jen.Id(recvName).Dot("params").Dot(fldName)
				if !param.Required {
					fldParam = jen.Op("*").Add(fldParam)
				}

				s.Dot(strcase.ToCamel(param.Var.Name)).Op("=").Add(fldParam)
			}).Line()

			if !param.Required {
				group.If(jen.Id(recvName).Dot("params").Dot(fldName).Op("!=").Nil()).Block(fldAssign).Line()
			} else {
				group.Add(fldAssign)
			}
		}

		group.Return(jen.Id("body"))
	}

	return group
}

func (g *ClientGenerator) genJSONReqContent(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()
	group.Id("req").Dot("Header").Dot("Add").Call(jen.Lit("Content-Type"), jen.Lit("application/json")).Line()
	group.Var().Id("reqData").Qual("bytes", "Buffer").Line()

	group.If(
		jen.Err().Op(":=").Qual(annotation.JSONPkg, "NewEncoder").Call(jen.Op("&").Id("reqData")).Dot("Encode").Call(jen.Id(recvName).Dot("makeBodyRequest").Call()),
		jen.Err().Op("!=").Nil(),
	).Block(
		g.genTrace(
			g.genAddEventTrace(jen.Lit("JSON encode error"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call())),
			g.genSetStatusErrorTrace(jen.Lit("failed sent request")),
		),

		jen.Return(
			annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
		),
	).Line()
	group.Id("req").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Op("&").Id("reqData"))

	return group
}

func (g *ClientGenerator) genMakeBodyRequetsMethod(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	methodRequestName := methodRequestName(methodOpt)

	group.Func().Params(jen.Id(recvName).Op("*").Id(methodRequestName)).Id("makeBodyRequest").Params().Any().Block(
		g.genReqBodyStruct(methodOpt),
	).Line()

	return group
}

func (g *ClientGenerator) genQueryParams(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	group.Id("q").Op(":=").Id("req").Dot("URL").Dot("Query").Call()

	for _, param := range methodOpt.QueryParams {
		paramID := jen.Id(recvName).Dot("params").Dot(param.Var.Name)

		if !param.Required {
			paramID = jen.Call(jen.Op("*").Add(paramID))
		}

		code := typetransform.For(param.Var.Type).
			SetValueID(paramID).
			SetQualFunc(g.qual).
			Format()

		queryAdd := jen.Id("q").Dot("Add").Call(jen.Lit(param.Name), code)
		if !param.Required {
			queryAdd = jen.If(jen.Id(recvName).Dot("params").Dot(param.Var.Name).Op("!=").Nil()).Block(queryAdd)
		}

		group.Add(queryAdd)
	}

	group.Id("req").Dot("URL").Dot("RawQuery").Op("=").Id("q").Dot("Encode").Call()

	return group
}

func (g *ClientGenerator) genHeaderParams(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")
	for _, param := range methodOpt.HeaderParams {
		paramID := jen.Id(recvName).Dot("params").Dot(param.Var.Name)

		if !param.Required {
			paramID = jen.Call(jen.Op("*").Add(paramID))
		}

		code := typetransform.For(param.Var.Type).
			SetValueID(paramID).
			SetQualFunc(g.qual).
			Format()

		headerAdd := jen.Id("req").Dot("Header").Dot("Add").Call(jen.Lit(param.Name), code)

		if !param.Required {
			headerAdd = jen.If(jen.Id(recvName).Dot("params").Dot(param.Var.Name).Op("!=").Nil()).Block(headerAdd)
		}

		group.Add(headerAdd)
	}

	return group
}

func (g *ClientGenerator) genCookieParams(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	for _, param := range methodOpt.CookieParams {
		paramID := jen.Id(recvName).Dot("params").Dot(param.Var.Name)

		if !param.Required {
			paramID = jen.Call(jen.Op("*").Add(paramID))
		}

		code := typetransform.For(param.Var.Type).
			SetValueID(paramID).
			SetQualFunc(g.qual).
			Format()

		cookieAdd := jen.Id("req").Dot("AddCookie").Call(jen.Op("&").Qual(annotation.HTTPPkg, "Cookie").Values(
			jen.Id("Name").Op(":").Lit(param.Name),
			jen.Id("Value").Op(":").Add(code),
		))

		if !param.Required {
			cookieAdd = jen.If(jen.Id(recvName).Dot("params").Dot(param.Var.Name).Op("!=").Nil()).Block(cookieAdd)
		}

		group.Add(cookieAdd)
	}
	return group
}

func (g *ClientGenerator) genExecuteMethod(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	methodRequestName := methodRequestName(methodOpt)

	group.Func().Params(jen.Id(recvName).Op("*").Id(methodRequestName)).Id("Execute").
		Params(
			jen.Id("opts").Op("...").Id("ClientOption"),
		).
		ParamsFunc(func(group *jen.Group) {
			for _, result := range methodOpt.Func.Results {
				group.Id(result.Name).Add(jenutils.TypeInfoQual(result.Type, g.qual))
			}
		}).
		BlockFunc(func(group *jen.Group) {
			group.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
				jen.Id("o").Call(jen.Op("&").Id(recvName).Dot("opts")),
			)
			group.List(jen.Id("ctx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(jen.Id(recvName).Dot("opts").Dot("ctx"))
			group.Defer().Id("cancel").Call()

			group.Add(g.genStartTrace(methodOpt))

			group.Do(func(s *jen.Statement) {
				if len(methodOpt.PathParams) > 0 {
					var paramsCall []jen.Code
					paramsCall = append(paramsCall, jen.Lit(sprintfPath(methodOpt)))
					for _, p := range methodOpt.PathParams {
						paramsCall = append(paramsCall, jen.Id(recvName).Dot("params").Dot(p.PathParamName))
					}
					s.Id("path").Op(":=").Qual("fmt", "Sprintf").Call(paramsCall...)
				} else {
					s.Id("path").Op(":=").Lit(methodOpt.Path)
				}
			})

			group.Id("r").Dot("opts").Dot("ctx").Op("=").Qual(annotation.CTXPkg, "WithValue").Call(
				jen.Id("r").Dot("opts").Dot("ctx"),
				jen.Id("methodContextKey"),
				jen.Id(constFullName(methodOpt)),
			)

			group.Id("r").Dot("opts").Dot("ctx").Op("=").Qual(annotation.CTXPkg, "WithValue").Call(
				jen.Id("r").Dot("opts").Dot("ctx"),
				jen.Id("shortMethodContextKey"),
				jen.Id(constShortName(methodOpt)),
			)

			group.Id("r").Dot("opts").Dot("ctx").Op("=").Qual(annotation.CTXPkg, "WithValue").Call(
				jen.Id("r").Dot("opts").Dot("ctx"),
				jen.Id("scopeNameContextKey"),
				jen.Id(strcase.ToLowerCamel(methodOpt.Iface.NameTypeInfo.Name)+"ScopeName"),
			)

			group.List(jen.Id("req"), jen.Err()).Op(":=").Qual(annotation.HTTPPkg, "NewRequestWithContext").Call(
				jen.Id("r").Dot("opts").Dot("ctx"),
				jen.Lit(methodOpt.Method),
				jen.Id(recvName).Dot("c").Dot("target").Op("+").Id("path"), jen.Nil(),
			)
			group.If(jen.Err().Op("!=").Nil()).Block(
				g.genTrace(
					g.genAddEventTrace(jen.Lit("request make error"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call())),
					g.genSetStatusErrorTrace(jen.Lit("failed sent request")),
				),
				jen.Return(
					annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
				),
			)

			group.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("application/json"))

			group.If(jen.Id("r").Dot("opts").Dot("propagator").Op("!=").Nil()).Block(
				jen.Id("r").Dot("opts").Dot("propagator").Dot("Inject").Call(
					jen.Id("ctx"),
					jen.Qual(annotation.OtelPropagationPkg, "HeaderCarrier").Call(jen.Id("req").Dot("Header")),
				),
			)

			if len(methodOpt.BodyParams) > 0 {
				group.Switch(jen.Id("r").Dot("opts").Dot("content")).BlockFunc(func(group *jen.Group) {
					group.Default().Block(
						g.genJSONReqContent(methodOpt),
					)

					if !annotation.IsObjectType(methodOpt.BodyParams[0].Var.Type) {
						if methodOpt.Use.URLEncoded {
							group.Case(jen.Lit("application/x-www-form-urlencoded")).BlockFunc(func(group *jen.Group) {
								group.Id("req").Dot("Header").Dot("Add").Call(jen.Lit("Content-Type"), jen.Lit("application/x-www-form-urlencoded"))

								group.Id("body").Op(":=").Qual(annotation.URLPkg, "Values").Values()

								for _, p := range methodOpt.BodyParams {
									if p.Var.IsContext || !p.Var.Type.IsBasic {
										continue
									}
									paramID := jen.Id("r").Dot("params").Dot(strcase.ToLowerCamel(p.Var.Name))

									valueID := jen.Add(paramID)
									if !p.Required {
										valueID = jen.Op("*").Add(valueID)
									}

									code := typetransform.For(p.Var.Type).
										SetQualFunc(g.qual).
										SetValueID(valueID).
										Format()

									if !p.Required {
										group.If(jen.Add(paramID).Op("!=").Nil()).Block(
											group.Id("body").Dot("Set").Call(jen.Lit(p.Name), code),
										)
									} else {
										group.Id("body").Dot("Set").Call(jen.Lit(p.Name), code)
									}
								}

								group.Id("req").Dot("Body").Op("=").Add(wrapIOCloser(jen.Qual(annotation.StringsPkg, "NewReader").Call(jen.Id("body").Dot("Encode").Call())))
							})
						}

						if methodOpt.Use.Multipart {
							group.Case(jen.Lit("multipart/form-data")).BlockFunc(func(group *jen.Group) {
								group.Id("req").Dot("Header").Dot("Add").Call(jen.Lit("Content-Type"), jen.Lit("multipart/form-data"))

								group.Var().Id("body").Qual(annotation.BytesPkg, "Buffer")
								group.Id("multipartWriter").Op(":=").Qual(annotation.MimeMultipartPkg, "NewWriter").Call(jen.Op("&").Id("body"))

								for _, p := range methodOpt.BodyParams {
									if p.Var.IsContext || !p.Var.Type.IsBasic {
										continue
									}

									paramID := jen.Id("r").Dot("params").Dot(strcase.ToLowerCamel(p.Var.Name))
									valueID := paramID.Clone()

									if !p.Required {
										valueID = jen.Op("*").Add(valueID)
									}

									code := typetransform.For(p.Var.Type).
										SetQualFunc(g.qual).
										SetValueID(valueID).
										Format()

									writeFieldCode := jen.If(
										jen.Err().Op("=").Id("multipartWriter").Dot("WriteField").Call(jen.Lit(p.Name), code),
										jen.Err().Op("!=").Nil(),
									).Block(
										g.genTrace(
											g.genAddEventTrace(jen.Lit("multipart write feld "+p.Name+" error"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call())),
											g.genSetStatusErrorTrace(jen.Lit("failed sent request")),
										),
										jen.Return(
											annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
										),
									)

									if !p.Required {
										group.If(jen.Add(paramID).Op("!=").Nil()).Block(writeFieldCode)
									} else {
										group.Add(writeFieldCode)
									}
								}
								group.Id("multipartWriter").Dot("Close").Call()
								group.Id("req").Dot("Body").Op("=").Add(wrapIOCloser(jen.Op("&").Id("body")))
							})
						}
					}
				})
			}

			if len(methodOpt.QueryParams) > 0 {
				group.Add(g.genQueryParams(methodOpt))
			}

			if len(methodOpt.HeaderParams) > 0 {
				group.Add(g.genHeaderParams(methodOpt))
			}

			if len(methodOpt.CookieParams) > 0 {
				group.Add(g.genCookieParams(methodOpt))
			}

			group.Id("before").Op(":=").Append(jen.Id(recvName).Dot("c").Dot("opts").Dot("before"), jen.Id(recvName).Dot("opts").Dot("before").Op("..."))
			group.For(jen.List(jen.Id("_"), jen.Id("before")).Op(":=").Range().Id("before")).Block(
				jen.List(jen.Id("ctx"), jen.Err()).Op("=").Id("before").Call(jen.Id("ctx"), jen.Id("req")),

				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Return(
						annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
					),
				),
			)
			group.List(jen.Id("resp"), jen.Err()).Op(":=").Id(recvName).Dot("opts").Dot("client").Dot("Do").Call(jen.Id("req"))
			group.If(jen.Err().Op("!=").Nil()).Block(
				g.genTrace(
					g.genAddEventTrace(jen.Lit("do request error"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call())),
					g.genSetStatusErrorTrace(jen.Lit("failed sent request")),
				),
				jen.Return(
					annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
				),
			)

			group.Id("after").Op(":=").Append(jen.Id(recvName).Dot("c").Dot("opts").Dot("after"), jen.Id(recvName).Dot("opts").Dot("after").Op("..."))
			group.For(jen.List(jen.Id("_"), jen.Id("after")).Op(":=").Range().Id("after")).Block(
				jen.Id("ctx").Op("=").Id("after").Call(jen.Id("ctx"), jen.Id("resp")),
			)
			group.Defer().Id("resp").Dot("Body").Dot("Close").Call()
			group.Defer().Id("cancel").Call()

			group.If(jen.Id("resp").Dot("StatusCode").Op(">").Lit(httpStatusLastSuccessCode)).BlockFunc(func(group *jen.Group) {
				group.Add(
					g.genTrace(
						g.genAddEventTrace(jen.Lit("response status code failed"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Id("resp").Dot("Status"))),
						g.genSetStatusErrorTrace(jen.Lit("failed response")),
					),
				)

				group.If(jen.Id("r").Dot("opts").Dot("errorDecode").Op("!=").Nil()).Block(
					jen.Return(
						annotation.MakeEmptyResults(
							methodOpt.BodyResults,
							g.qual,
							jen.Id("r").Dot("opts").Dot("errorDecode").Call(
								jen.Id("resp").Dot("Body"),
								jen.Id("resp").Dot("StatusCode"),
							),
						)...,
					),
				)

				// group.If(jen.Id("resp").Dot("Body").Op("==").Qual(annotation.HTTPPkg, "NoBody")).Block(
				// 	jen.Return(
				// 		annotation.MakeEmptyResults(
				// 			methodOpt.BodyResults,
				// 			g.qual,
				// 			jen.Qual(annotation.FmtPkg, "Errorf").Call(jen.Lit("http error %d"), jen.Id("resp").Dot("StatusCode")),
				// 		)...,
				// 	),
				// )

				// if len(methodOpt.Iface.Errors) > 0 {
				// 	errorTypeName := errorTypeName(methodOpt.Iface)

				// 	group.Var().Id("clientErr").Id(errorTypeName)

				// 	for _, e := range methodOpt.Iface.Errors {
				// 		if e.StatusCode {
				// 			group.Id("clientErr").Dot(e.FldName).Op("=").Id("resp").Dot("StatusCode")
				// 		}
				// 	}

				// 	group.If(
				// 		jen.Err().Op(":=").Qual(annotation.JSONPkg, "NewDecoder").Call(jen.Id("resp").Dot("Body")).Dot("Decode").Call(jen.Op("&").Id("clientErr")),
				// 		jen.Id("err").Op("!=").Id("nil"),
				// 	).Block(
				// 		jen.Return(annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...),
				// 	)

				// 	group.Return(annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Op("&").Id("clientErr"))...)
				// } else {
				//

				// group.Id("err").Op("=").Do(g.qual(annotation.FmtPkg, "Errorf")).Call(jen.Lit("http error %d"), jen.Id("resp").Dot("StatusCode"))
				//
				group.Return(
					annotation.MakeEmptyResults(
						methodOpt.BodyResults,
						g.qual,
						jen.Qual(gomosaic.ClientPkg, "ErrorDecode").Call(
							jen.Id("resp").Dot("Body"),
							jen.Id("resp").Dot("StatusCode"),
							jen.Func().Params(
								jen.Id("_").Struct(),
								jen.Id("data").Index().Byte(),
							).Error().Block(
								jen.Return(
									jen.Op("&").Id("ClientError").Values(
										jen.Id("Data").Op(":").Id("data"),
										jen.Id("StatusCode").Op(":").Id("resp").Dot("StatusCode"),
									),
								),
							),
						),
					)...,
				)
				// }
			})

			if len(methodOpt.BodyResults) > 0 {
				group.Var().Id("reader").Qual("io", "ReadCloser")
				group.Switch(jen.Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("Content-Encoding"))).Block(
					jen.Default().Block(jen.Id("reader").Op("=").Id("resp").Dot("Body")),
					jen.Case(jen.Lit("gzip")).Block(
						jen.List(jen.Id("reader"), jen.Err()).Op("=").Qual("compress/gzip", "NewReader").Call(jen.Id("resp").Dot("Body")),
						jen.If(jen.Err().Op("!=").Nil()).Block(
							jen.Return(
								annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
							),
						),
						jen.Defer().Id("reader").Dot("Close").Call(),
					),
				)

				if len(methodOpt.BodyResults) == 1 && methodOpt.Single.Resp {
					group.Var().Id("respBody").Add(jenutils.TypeInfoQual(methodOpt.BodyResults[0].Var.Type, g.qual))
				} else {
					if len(methodOpt.WrapResp.PathParts) > 0 {
						group.Var().Id("respBody").Struct(annotation.WrapStruct(methodOpt.WrapResp.PathParts, annotation.MakeStructFieldsFromResults(methodOpt.BodyResults, g.qual)))
					} else {
						group.Var().Id("respBody").Struct(annotation.MakeStructFieldsFromResults(methodOpt.BodyResults, g.qual)).Line()
					}
				}

				group.If(
					jen.Err().Op(":=").Qual(annotation.JSONPkg, "NewDecoder").Call(jen.Id("reader")).Dot("Decode").Call(jen.Op("&").Id("respBody")),
					jen.Err().Op("!=").Nil(),
				).Block(
					g.genTrace(
						g.genAddEventTrace(jen.Lit("JSON decode error"), jen.Qual(annotation.OtelTraceAttrPkg, "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call())),
						g.genSetStatusErrorTrace(jen.Lit("failed read response")),
					),
					jen.Return(
						annotation.MakeEmptyResults(methodOpt.BodyResults, g.qual, jen.Err())...,
					),
				)

				group.Add(
					g.genTrace(
						g.genSetStatusOkTrace(jen.Lit("request sent successfully")),
					),
				)

				group.ReturnFunc(func(g *jen.Group) {
					if len(methodOpt.BodyResults) > 0 {
						if len(methodOpt.BodyResults) == 1 && methodOpt.Single.Resp {
							g.Id("respBody")
						} else {
							var ids []jen.Code

							for _, name := range methodOpt.WrapResp.PathParts {
								ids = append(ids, jen.Dot(strcase.ToCamel(name)))
							}
							for _, result := range methodOpt.BodyResults {
								g.Id("respBody").Add(ids...).Dot(strcase.ToCamel(result.Name))
							}
						}
					}
					g.Nil()
				})
			} else {
				group.Return(jen.Nil())
			}
		})
	return group
}

func (g *ClientGenerator) genParamSetters(params []*annotation.MethodParamOpt) jen.Code {
	group := jen.Null()

	for _, param := range params {
		if param.Required {
			continue
		}
		methodSetName := strcase.ToCamel(param.Var.Name)
		fldName := jen.Id(strcase.ToLowerCamel(param.Var.Name))

		group.Dot("Set" + methodSetName).Call(fldName)
	}

	return group
}

func (g *ClientGenerator) genClientMethod(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	clientName := clientStructName(methodOpt.Iface)
	methodMakeRequestName := methodMakeRequestName(methodOpt)
	methodName := methodRequestName(methodOpt)

	group.Func().Params(jen.Id("c").Op("*").Id(clientName)).Id(methodMakeRequestName).
		ParamsFunc(func(group *jen.Group) {
			for _, param := range methodOpt.Params {
				if param.Required {
					group.Id(strcase.ToLowerCamel(param.Var.Name)).Add(jenutils.TypeInfoQual(param.Var.Type, g.qual))
				}
			}
		}).
		Op("*").Id(methodName).BlockFunc(func(group *jen.Group) {
		group.Id("m").Op(":=").Op("&").Id(methodName).Values(
			jen.Id("opts").Op(":").Id("c").Dot("opts"),
			jen.Id("c").Op(":").Id("c"),
		)
		for _, param := range methodOpt.Params {
			if param.Var.IsContext {
				continue
			}
			if param.Required {
				group.Id("m").Dot("params").Dot(strcase.ToLowerCamel(param.Var.Name)).Op("=").Id(strcase.ToLowerCamel(param.Var.Name))
			}
		}
		group.Return(jen.Id("m"))
	})

	group.Func().Params(jen.Id("c").Op("*").Id(clientName)).Id(methodOpt.Func.Name).
		ParamsFunc(func(group *jen.Group) {
			for _, param := range methodOpt.Func.Params {
				group.Id(param.Name).Add(jenutils.TypeInfoQual(param.Type, g.qual))
			}
		}).
		ParamsFunc(func(group *jen.Group) {
			for _, result := range methodOpt.Results {
				group.Id(strcase.ToLowerCamel(result.Var.Name)).Add(jenutils.TypeInfoQual(result.Var.Type, g.qual))
			}
		}).
		BlockFunc(func(group *jen.Group) {
			group.ListFunc(func(group *jen.Group) {
				for _, param := range methodOpt.Results {
					group.Id(strcase.ToLowerCamel(param.Var.Name))
				}
			}).Op("=").Id("c").Dot(methodMakeRequestName).CallFunc(func(group *jen.Group) {
				for _, param := range methodOpt.Params {
					if param.Required {
						group.Id(strcase.ToLowerCamel(param.Var.Name))
					}
				}
			}).CustomFunc(jen.Options{}, func(group *jen.Group) {
				group.Add(g.genParamSetters(methodOpt.BodyParams))
				group.Add(g.genParamSetters(methodOpt.QueryParams))
				group.Add(g.genParamSetters(methodOpt.HeaderParams))
				group.Add(g.genParamSetters(methodOpt.CookieParams))
			}).Dot("Execute").CallFunc(func(group *jen.Group) {
				if methodOpt.Context != nil {
					group.Id("WithContext").Call(jen.Id(methodOpt.Context.Name))
				}
			})
			group.Return()
		})

	return group
}

func (g *ClientGenerator) genReqStructSetters(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	for _, param := range methodOpt.Params {
		if param.Var.IsContext {
			continue
		}
		methodRequestName := methodRequestName(methodOpt)

		fldName := strcase.ToLowerCamel(param.Var.Name)
		fnName := strcase.ToCamel(param.Var.Name)

		group.Func().Params(
			jen.Id(recvName).Op("*").Id(methodRequestName),
		).Id("Set" + fnName).Params(
			jen.Id(fldName).Add(jenutils.TypeInfoQual(param.Var.Type, g.qual)),
		).Op("*").Id(methodRequestName).BlockFunc(func(g *jen.Group) {
			g.Add(jen.CustomFunc(jen.Options{}, func(g *jen.Group) {
				g.Id(recvName).Dot("params").Dot(fldName).Op("=")
				if !param.Required && !param.Var.Type.IsPtr {
					g.Op("&")
				}
				g.Id(fldName)
			}))
			g.Return(jen.Id(recvName))
		})
	}
	return group
}

func (g *ClientGenerator) genRequestStructParam(p *annotation.MethodParamOpt) jen.Code {
	name := strcase.ToLowerCamel(p.Var.Name)

	paramNameID := jen.Id(name)
	if !p.Required && !p.Var.Type.IsPtr {
		paramNameID.Op("*")
	}

	return paramNameID.Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
}

func (g *ClientGenerator) genReqStruct(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	methodRequestName := methodRequestName(methodOpt)
	clientName := clientStructName(methodOpt.Iface)

	group.Type().Id(methodRequestName).StructFunc(func(group *jen.Group) {
		group.Id("c").Op("*").Id(clientName)
		group.Id("opts").Id(clientOptionName)
		group.Id("params").StructFunc(func(group *jen.Group) {
			for _, param := range methodOpt.Params {
				if param.Var.IsContext {
					continue
				}
				group.Add(g.genRequestStructParam(param))
			}
		})
	})

	return group
}

func (g *ClientGenerator) genClientEndpoint(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	group.Add(g.genClientMethod(methodOpt))

	group.Add(g.genReqStruct(methodOpt))
	group.Add(g.genReqStructSetters(methodOpt))

	if len(methodOpt.BodyParams) > 0 {
		group.Add(g.genMakeBodyRequetsMethod(methodOpt))
	}

	group.Add(g.genExecuteMethod(methodOpt))

	return group
}

func (g *ClientGenerator) genClientEndpoints(ifaceOpt *annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	for _, methodOpt := range ifaceOpt.Methods {
		group.Add(g.genClientEndpoint(methodOpt))
	}

	return group
}

func (g *ClientGenerator) genClientConstruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	clientName := clientStructName(ifaceOpt)

	group := jen.NewFile("")

	group.Func().Id("New"+ifaceOpt.NameTypeInfo.Name+"Client").
		Params(
			jen.Id("target").String(),
			jen.Id("opts").Op("...").Id("ClientOption"),
		).Op("*").Id(clientName).BlockFunc(
		func(g *jen.Group) {
			g.Id("c").Op(":=").Op("&").Id(clientName).Values(
				jen.Id("target").Op(":").Id("target"),
				jen.Id("opts").Op(":").Id(clientOptionName).Values(
					jen.Id("client").Op(":").Qual(annotation.CleanHTTPPkg, "DefaultClient").Call(),
				),
			)
			g.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
				jen.Id("o").Call(jen.Op("&").Id("c").Dot("opts")),
			)
			g.Return(jen.Id("c"))
		},
	)

	return group
}

func (g *ClientGenerator) genClientDTO(structGen *structure.Generator, ifaceOpt *annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	for _, m := range ifaceOpt.Methods {
		for _, p := range m.BodyParams {
			structGen.Generate(p.Var.Type)
		}
		for _, p := range m.BodyResults {
			structGen.Generate(p.Var.Type)
		}
	}

	return group
}

func (g *ClientGenerator) genClientStruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	for _, m := range ifaceOpt.Methods {
		group.Const().Id(constShortName(m)).Op("=").Lit(m.Func.ShortName)
		group.Const().Id(constFullName(m)).Op("=").Lit(m.Func.FullName)
	}

	clientName := clientStructName(ifaceOpt)

	group.Const().Id(strcase.ToLowerCamel(ifaceOpt.NameTypeInfo.Name) + "ScopeName").Op("=").Lit(filepath.Base(ifaceOpt.NameTypeInfo.Package.Path))

	group.Type().Id(clientName).StructFunc(func(g *jen.Group) {
		g.Id("target").String()
		g.Id("opts").Id(clientOptionName)
	})

	return group
}

func (g *ClientGenerator) genTypes() jen.Code {
	labelContextMethodShortName := jen.Id("labelFromContext").Call(jen.Lit("methodNameShort"), jen.Id("shortMethodContextKey"))
	labelContextMethodFullName := jen.Id("labelFromContext").Call(jen.Lit("methodNameFull"), jen.Id("methodContextKey"))
	labelContextScopeName := jen.Id("labelFromContext").Call(jen.Lit("scopeName"), jen.Id("scopeNameContextKey"))

	group := jen.NewFile("")

	group.Type().Id("contextKey").String()
	group.Const().Id("methodContextKey").Id("contextKey").Op("=").Lit("method")
	group.Const().Id("shortMethodContextKey").Id("contextKey").Op("=").Lit("shortMethod")
	group.Const().Id("scopeNameContextKey").Id("contextKey").Op("=").Lit("scopeName")

	group.Func().Id("labelFromContext").Params(
		jen.Id("lblName").String(),
		jen.Id("ctxKey").Id("contextKey"),
	).Qual(annotation.PromHTTPPkg, "Option").Block(
		jen.Return(
			jen.Qual(annotation.PromHTTPPkg, "WithLabelFromCtx").Call(
				jen.Id("lblName"),
				jen.Func().Params(jen.Id("ctx").Qual(annotation.CTXPkg, "Context")).String().Block(
					jen.List(jen.Id("v"), jen.Id("_")).Op(":=").Id("ctx").Dot("Value").Call(jen.Id("ctxKey")).Assert(jen.String()),
					jen.Return(jen.Id("v")),
				),
			),
		),
	)
	group.Func().Id("instrumentRoundTripperErrCounter").Params(
		jen.Id("counter").Op("*").Qual(annotation.PrometheusPkg, "CounterVec"),
		jen.Id("next").Qual(annotation.HTTPPkg, "RoundTripper"),
	).Qual(annotation.PromHTTPPkg, "RoundTripperFunc").Block(
		jen.Return(
			jen.Func().
				Params(
					jen.Id("r").Op("*").Qual(annotation.HTTPPkg, "Request"),
				).
				Params(
					jen.Op("*").Qual(annotation.HTTPPkg, "Response"),
					jen.Error(),
				).
				Block(
					jen.Id("labels").Op(":=").Qual(annotation.PrometheusPkg, "Labels").Values(
						jen.Lit("method").Op(":").Qual(annotation.StringsPkg, "ToLower").Call(jen.Id("r").Dot("Method")),
					),
					jen.List(jen.Id("labels").Index(jen.Lit("methodNameFull")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("methodContextKey")).Assert(jen.String()),
					jen.List(jen.Id("labels").Index(jen.Lit("methodNameShort")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("shortMethodContextKey")).Assert(jen.String()),
					jen.List(jen.Id("labels").Index(jen.Lit("scopeName")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("scopeNameContextKey")).Assert(jen.String()),
					jen.List(jen.Id("labels").Index(jen.Lit("code"))).Op("=").Lit(""),
					jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("next").Dot("RoundTrip").Call(jen.Id("r")),
					jen.If(jen.Id("err").Op("!=").Nil()).Block(
						jen.Var().Id("errType").String(),
						jen.Switch(jen.Id("e").Op(":=").Err().Assert(jen.Id("type"))).Block(
							jen.Default().Block(
								jen.Id("errType").Op("=").Err().Dot("Error").Call(),
							),
							jen.Case(jen.Op("*").Qual(annotation.TLSPkg, "CertificateVerificationError")).Block(
								jen.Id("errType").Op("=").Lit("failedVerifyCertificate"),
							),
							jen.Case(jen.Qual(annotation.NetPkg, "Error")).Block(
								jen.Id("errType").Op("+=").Lit("net."),
								jen.If(jen.Id("e").Dot("Timeout").Call()).Block(
									jen.Id("errType").Op("+=").Lit("timeout."),
								),
								jen.Switch(jen.Id("ee").Op(":=").Id("e").Assert(jen.Id("type"))).Block(
									jen.Case(jen.Op("*").Qual(annotation.NetPkg, "ParseError")).Block(
										jen.Id("errType").Op("+=").Lit("parse"),
									),
									jen.Case(jen.Op("*").Qual(annotation.NetPkg, "InvalidAddrError")).Block(
										jen.Id("errType").Op("+=").Lit("invalidAddr"),
									),
									jen.Case(jen.Op("*").Qual(annotation.NetPkg, "UnknownNetworkError")).Block(
										jen.Id("errType").Op("+=").Lit("unknownNetwork"),
									),
									jen.Case(jen.Op("*").Qual(annotation.NetPkg, "DNSError")).Block(
										jen.Id("errType").Op("+=").Lit("dns"),
									),
									jen.Case(jen.Op("*").Qual(annotation.NetPkg, "OpError")).Block(
										jen.Id("errType").Op("+=").Id("ee").Dot("Net").Op("+").Lit(".").Op("+").Id("ee").Dot("Op"),
									),
								),
							),
						),
						jen.Id("labels").Index(jen.Lit("errorCode")).Op("=").Id("errType"),
						jen.Id("counter").Dot("With").Call(jen.Id("labels")).Dot("Add").Call(jen.Lit(1)),
					).Else().If(jen.Id("resp").Dot("StatusCode").Op(">").Lit(httpStatusLastSuccessCode)).Block(
						jen.List(jen.Id("labels").Index(jen.Lit("code"))).Op("=").Qual(annotation.StrconvPkg, "Itoa").Call(jen.Id("resp").Dot("StatusCode")),
						jen.Id("labels").Index(jen.Lit("errorCode")).Op("=").Lit("respFailed"),
						jen.Id("counter").Dot("With").Call(jen.Id("labels")).Dot("Add").Call(jen.Lit(1)),
					),

					jen.Return(jen.Id("resp"), jen.Err()),
				),
		),
	)

	group.Type().Id("prometheusCollector").Interface(
		jen.Qual(annotation.PrometheusPkg, "Collector"),
		// jen.Id("Inflight").Params().Params(jen.Qual(annotation.PrometheusPkg, "Gauge")),
		jen.Id("Requests").Params().Params(jen.Op("*").Qual(annotation.PrometheusPkg, "CounterVec")),
		jen.Id("ErrRequests").Params().Params(jen.Op("*").Qual(annotation.PrometheusPkg, "CounterVec")),
		jen.Id("Duration").Params().Params(jen.Op("*").Qual(annotation.PrometheusPkg, "HistogramVec")),
		// jen.Id("DNSDuration").Params().Params(jen.Op("*").Qual(annotation.PrometheusPkg, "HistogramVec")),
		// jen.Id("TLSDuration").Params().Params(jen.Op("*").Qual(annotation.PrometheusPkg, "HistogramVec")),
	)

	group.Type().Id("ClientBeforeFunc").Func().Params(
		jen.Qual("context", "Context"),
		jen.Op("*").Qual("net/http", "Request"),
	).Params(jen.Qual("context", "Context"), jen.Error())

	group.Type().Id("ClientAfterFunc").Func().Params(
		jen.Qual("context", "Context"),
		jen.Op("*").Qual("net/http", "Response"),
	).Qual("context", "Context")

	group.Type().Id("ClientError").Struct(
		jen.Id("Data").Index().Byte(),
		jen.Id("StatusCode").Int(),
	)

	group.Func().Params(jen.Id("e").Op("*").Id("ClientError")).Id("Error").Params().String().Block(
		jen.Return(jen.Qual(annotation.FmtPkg, "Sprint").Call(jen.Id("e").Dot("StatusCode")).Op("+").Lit(": ").Op("+").String().Call(jen.Id("e").Dot("Data"))),
	)

	group.Type().Id("ErrorDecoder").Func().Params(jen.Qual("io", "ReadCloser"), jen.Int()).Error()

	group.Type().Id(clientOptionName).Struct(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("content").String(),
		jen.Id("tracer").Qual(annotation.OtelTracePkg, "Tracer"),
		jen.Id("propagator").Qual(annotation.OtelPropagationPkg, "TextMapPropagator"),
		jen.Id("before").Index().Id("ClientBeforeFunc"),
		jen.Id("after").Index().Id("ClientAfterFunc"),
		jen.Id("errorDecode").Id("ErrorDecoder"),
		jen.Id("client").Op("*").Qual(annotation.HTTPPkg, "Client"),
	)

	group.Type().Id("ClientOption").Func().Params(jen.Op("*").Id(clientOptionName))

	group.Func().Id("WithTracer").Params(jen.Id("tracer").Qual(annotation.OtelTracePkg, "Tracer")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("tracer").Op("=").Id("tracer"),
		)),
	)

	group.Func().Id("WithPropagator").Params(jen.Id("propagator").Qual(annotation.OtelPropagationPkg, "TextMapPropagator")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("propagator").Op("=").Id("propagator"),
		)),
	)

	group.Func().Id("WithContent").Params(jen.Id("content").String()).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("content").Op("=").Id("content"),
		)),
	)

	group.Func().Id("WithContext").Params(jen.Id("ctx").Qual("context", "Context")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("ctx").Op("=").Id("ctx"),
		)),
	)

	group.Func().Id("WithHTTPClient").Params(jen.Id("client").Op("*").Qual(annotation.HTTPPkg, "Client")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("client").Op("=").Id("client"),
		)),
	)

	group.Func().Id("WithPromCollector").Params(jen.Id("c").Id("prometheusCollector")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.If(jen.Id("o").Dot("client").Dot("Transport").Op("==").Nil()).Block(
				jen.Panic(jen.Lit("no transport is set for the http client")),
			),
			// jen.Id("trace").Op(":=").Op("&").Qual(annotation.PromHTTPPkg, "InstrumentTrace").Values(),
			jen.Id("o").Dot("client").Dot("Transport").Op("=").
				Id("instrumentRoundTripperErrCounter").Call(jen.Id("c").Dot("ErrRequests").Call(),
				// jen.Qual(annotation.PromHTTPPkg, "InstrumentRoundTripperInFlight").Call(
				// jen.Id("c").Dot("Inflight").Call(),
				jen.Qual(annotation.PromHTTPPkg, "InstrumentRoundTripperCounter").Call(
					jen.Id("c").Dot("Requests").Call(),
					// jen.Qual(annotation.PromHTTPPkg, "InstrumentRoundTripperTrace").Call(
					// jen.Id("trace"),
					jen.Qual(annotation.PromHTTPPkg, "InstrumentRoundTripperDuration").Call(
						jen.Id("c").Dot("Duration").Call(),
						jen.Id("o").Dot("client").Dot("Transport"),
						labelContextMethodShortName,
						labelContextMethodFullName,
						labelContextScopeName,
					),
					// ),
					labelContextMethodShortName,
					labelContextMethodFullName,
					labelContextScopeName,
				),
				// ),
			),
		)),
	)

	group.Func().Id("WithErrorDecode").Params(jen.Id("errorDecode").Id("ErrorDecoder")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("errorDecode").Op("=").Id("errorDecode"),
		)),
	)

	group.Func().Id("Before").Params(jen.Id("before").Op("...").Id("ClientBeforeFunc")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("before").Op("=").Append(jen.Id("o").Dot("before"), jen.Id("before").Op("...")),
		)),
	)

	group.Func().Id("After").Params(jen.Id("after").Op("...").Id("ClientAfterFunc")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id(clientOptionName)).Block(
			jen.Id("o").Dot("after").Op("=").Append(jen.Id("o").Dot("after"), jen.Id("after").Op("...")),
		)),
	)

	return group
}

// func (g *ClientGenerator) genErrorTypes(s *annotation.IfaceOpt) jen.Code {
// 	group := jen.NewFile("")

// 	fieldsMap := make(map[string]annotation.ErrorOpt)

// 	var structFields []jen.Code
// 	for _, e := range s.Errors {
// 		fieldsMap[e.FldName] = e
// 		structFields = append(structFields, jen.Id(e.FldName).Id(e.Type).Tag(map[string]string{"json": e.TagName}))
// 	}

// 	errorTypeName := errorTypeName(s)
// 	group.Type().Id(errorTypeName).Struct(structFields...)

// 	errorText := jen.Null()

// 	startTag := "{{"
// 	endTag := "}}"
// 	input := s.ErrorText
// 	index := 0
// 	for {
// 		startIndex := strings.Index(input, startTag)
// 		if startIndex == -1 {
// 			break
// 		}
// 		endIndex := strings.Index(input[startIndex:], endTag)
// 		if endIndex == -1 {
// 			break
// 		}

// 		endIndex += startIndex

// 		fldName := strings.TrimSpace(input[startIndex+len(startTag) : endIndex])
// 		fldNameID := jen.Id("e").Dot(fldName)

// 		e := fieldsMap[fldName]
// 		if e.Type != "" && e.Type != "string" {
// 			fldNameID = jen.Qual(annotation.FmtPkg, "Sprint").Call(fldNameID)
// 		}

// 		if index > 0 {
// 			errorText.Op("+")
// 		}

// 		if s := input[:startIndex]; s != "" {
// 			errorText.Lit(s).Op("+")
// 		}

// 		errorText.Add(fldNameID)
// 		input = input[endIndex+len(endTag):]
// 		index++
// 	}

// 	if input != "" {
// 		if index > 0 {
// 			errorText.Op("+")
// 		}
// 		errorText.Lit(input)
// 	}

// 	group.Func().Params(jen.Id("e").Op("*").Id(errorTypeName)).Id("Error").Params().String().Block(
// 		jen.Return(errorText),
// 	)

// 	return group
// }

func (g *ClientGenerator) genTrace(codes ...jen.Code) jen.Code {
	group := jen.NewFile("").Null()
	tracerID := jen.Id("r").Dot("opts").Dot("tracer")
	group.If(jen.Add(tracerID).Op("!=").Nil()).Block(codes...)
	return group
}

func (g *ClientGenerator) genStartTrace(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	tracerID := jen.Id("r").Dot("opts").Dot("tracer")

	group.Var().Id("span").Qual(annotation.OtelTracePkg, "Span")

	group.Add(
		g.genTrace(
			jen.List(jen.Id("ctx"), jen.Id("span")).Op("=").Add(tracerID).Dot("Start").Call(jen.Id("ctx"), jen.Id(constShortName(methodOpt)), jen.Qual(annotation.OtelTracePkg, "WithSpanKind").Call(jen.Qual(annotation.OtelTracePkg, "SpanKindServer"))),
			jen.Defer().Id("span").Dot("End").Call(),
		),
	)

	return group
}

func (g *ClientGenerator) genAddEventTrace(msg jen.Code, options ...jen.Code) jen.Code {
	group := jen.NewFile("").Null()
	group.Id("span").Dot("AddEvent").CallFunc(func(group *jen.Group) {
		group.Add(msg)

		if len(options) > 0 {
			group.Add(jen.Qual(annotation.OtelTracePkg, "WithAttributes")).Call(options...)
		}
	})

	return group
}

func (g *ClientGenerator) genSetStatusErrorTrace(msg jen.Code) jen.Code {
	return jen.Id("span").Dot("SetStatus").Call(jen.Qual(annotation.OtelCodesPkg, "Error"), msg)
}

func (g *ClientGenerator) genSetStatusOkTrace(msg jen.Code) jen.Code {
	return jen.Id("span").Dot("SetStatus").Call(jen.Qual(annotation.OtelCodesPkg, "Ok"), msg)
}
