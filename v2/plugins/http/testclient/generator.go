package testclient

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/flatten"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// generator генерирует код тестов HTTP-клиента.
type generator struct {
	qual gomosaic.QualFunc
}

// Generate генерирует все тестовые функции для переданных конфигураций.
func (g *generator) Generate(ifaceOpts []*annotation.IfaceOpt, configs []Config) jen.Code {
	group := jen.NewFile("")

	// ptr helper
	group.Func().Id("ptr").Types(jen.Id("T").Any()).Params(jen.Id("t").Id("T")).Op("*").Id("T").Block(
		jen.Return(jen.Op("&").Id("t")),
	)

	for _, ifaceOpt := range ifaceOpts {
		for _, methodOpt := range ifaceOpt.Methods {
			constructName := "create" + ifaceOpt.NameTypeInfo.Name + "Client"

			for _, cfg := range configs {
				testMethod := fmt.Sprintf("%s_%d", methodOpt.Func.Name, cfg.StatusCode)
				testName := "Test" + ifaceOpt.NameTypeInfo.Name + "_" + testMethod

				group.Func().Id(testName).Params(jen.Id("t").Op("*").Qual("testing", "T")).BlockFunc(func(body *jen.Group) {
					body.Add(g.genServerResponseGenerate(cfg, methodOpt))
					body.Add(g.genBodyParamsGenerate(methodOpt))
					body.Add(g.genParamsGenerate(methodOpt.Params))
					body.Add(g.genMockServerGenerate(methodOpt, cfg))

					body.Id("server").Op(":=").Qual("net/http/httptest", "NewServer").Call(jen.Id("mockServer"))
					body.Id("client").Op(":=").Id(constructName).Call(
						jen.Id("server").Dot("URL"),
						jen.Lit(methodOpt.Func.Name),
						jen.Lit(cfg.StatusCode),
					)

					// Вызов метода клиента
					body.Do(func(s *jen.Statement) {
						if len(methodOpt.Func.Results) > 0 {
							s.ListFunc(func(group *jen.Group) {
								for _, r := range methodOpt.Func.Results {
									if r.IsError {
										group.Id(r.Name)
									} else if !cfg.CheckError {
										group.Id(r.Name)
									} else {
										group.Id("_")
									}
								}
							})
							if _, ok := gomosaic.HasError(methodOpt.Func.Results); cfg.CheckError || ok {
								s.Op(":=")
							} else {
								s.Op("=")
							}
						}
					}).Id("client").Dot(methodOpt.Func.Name).CallFunc(func(group *jen.Group) {
						if methodOpt.Context != nil {
							group.Qual("context", "TODO").Call()
						}
						for _, p := range methodOpt.Params {
							if p.Var.IsContext {
								continue
							}
							name := strcase.ToLowerCamel(p.Var.Name)
							switch p.HTTPType {
							case annotation.HeaderHTTPType:
								group.Id(name + "Header")
							case annotation.CookieHTTPType:
								group.Id(name + "Cookie")
							case annotation.QueryHTTPType:
								group.Id(name + "Query")
							case annotation.BodyHTTPType:
								group.Id("serverRequest").Dot(strcase.ToCamel(name))
							case annotation.PathHTTPType:
								group.Id(name + "Path")
							}
						}
					})

					body.Add(g.genCheckError(methodOpt, cfg))
					body.Add(g.genCheckBodyResult(methodOpt, cfg))
				})
			}
		}
	}

	return group
}

// genServerResponseGenerate — генерирует serverResponse с заполнением полей через faker.
func (g *generator) genServerResponseGenerate(cfg Config, methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if !cfg.CheckError && len(methodOpt.BodyResults) > 0 {
		if len(methodOpt.BodyResults) == 1 {
			serverResponse := methodOpt.BodyResults[0]
			serverResponseTypeInfo := serverResponse.Var.Type
			if serverResponseTypeInfo.IsPtr {
				serverResponseTypeInfo = serverResponseTypeInfo.ElemType
			}
			if serverResponseTypeInfo.IsNamed {
				serverResponseTypeInfo = serverResponseTypeInfo.ElemType
			}

			if serverResponseTypeInfo.Struct != nil {
				if serverResponse.Var.Type.IsPtr {
					group.Id("serverResponse").Op(":=").Op("&").Add(jenutils.TypeInfoQual(serverResponse.Var.Type.ElemType, g.qual)).ValuesFunc(func(dict *jen.Group) {
						for _, f := range serverResponseTypeInfo.Struct.Fields {
							for _, v := range flatten.Flatten(f) {
								if v.IsArray {
									continue
								}
								dict.Add(v.Path).Op(":").Add(g.typeToValue(v.Var.Type))
							}
						}
					})
				} else {
					group.Var().Id("serverResponse").Add(jenutils.TypeInfoQual(serverResponse.Var.Type, g.qual)).Line()
					for _, f := range serverResponseTypeInfo.Struct.Fields {
						for _, v := range flatten.Flatten(f) {
							if v.IsArray {
								continue
							}
							group.Id("serverResponse").Op(".").Add(v.Path).Op("=").Add(g.typeToValue(v.Var.Type)).Line()
						}
					}
				}
			} else {
				group.Id("serverResponse").Op(":=").Add(g.typeToValue(serverResponse.Var.Type))
			}
		}
	}
	return group
}

// genBodyParamsGenerate — генерирует serverRequest с заполнением вложенных полей.
func (g *generator) genBodyParamsGenerate(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if len(methodOpt.BodyParams) > 0 {
		group.Var().Id("serverRequest").StructFunc(func(group *jen.Group) {
			for _, p := range methodOpt.BodyParams {
				group.Id(strcase.ToCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
			}
		}).Line()

		for _, p := range methodOpt.BodyParams {
			typeInfo := p.Var.Type
			if typeInfo.IsNamed {
				typeInfo = typeInfo.ElemType
			}

			if typeInfo.Struct != nil {
				for _, f := range typeInfo.Struct.Fields {
					for _, v := range flatten.Flatten(f) {
						if v.IsArray {
							continue
						}
						group.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op(".").Add(v.Path).Op("=").Add(g.typeToValue(v.Var.Type)).Line()
					}
				}
			} else if p.Var.Type.IsPtr && p.Var.Type.ElemType.IsNamed && p.Var.Type.ElemType.ElemType != nil && p.Var.Type.ElemType.ElemType.Struct != nil {
				innerType := p.Var.Type.ElemType.ElemType
				group.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op("=").Op("&").Add(jenutils.TypeInfoQual(p.Var.Type.ElemType, g.qual)).ValuesFunc(func(dict *jen.Group) {
					for _, f := range innerType.Struct.Fields {
						for _, v := range flatten.Flatten(f) {
							if v.IsArray {
								continue
							}
							dict.Add(v.Path).Op(":").Add(g.typeToValue(v.Var.Type))
						}
					}
				})
			} else {
				group.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op("=").Add(g.typeToValue(p.Var.Type)).Line()
			}
		}
	}
	return group
}

// genParamsGenerate — генерирует параметры (query, header, cookie, path).
func (g *generator) genParamsGenerate(params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("").Null()

	for _, p := range params {
		if p.Var.IsContext || p.HTTPType == annotation.BodyHTTPType {
			continue
		}
		postfix := strcase.ToCamel(p.HTTPType)
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name) + postfix).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual)).Op("=").Add(g.typeToValue(p.Var.Type)).Line()
	}
	return group
}
