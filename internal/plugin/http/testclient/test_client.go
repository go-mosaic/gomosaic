package testclient

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/jaswdr/faker/v2"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
	"github.com/go-mosaic/gomosaic/pkg/flatten"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
	"github.com/go-mosaic/gomosaic/pkg/typetransform"
)

type Config struct {
	StatusCode int
	CheckError bool
}

type ClientTestGenerator struct {
	fake   faker.Faker
	qualFn jenutils.QualFunc
}

func (g *ClientTestGenerator) basicTypeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	switch {
	default:
		return jen.Lit(g.fake.Lorem().Sentence(10)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsBoolean:
		return jen.Lit(true)
	case typeInfo.BasicInfo == gomosaic.IsInteger || typeInfo.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned:
		return jen.Lit(g.fake.RandomNumber(5)) //nolint: mnd
	case typeInfo.BasicInfo == gomosaic.IsFloat:
		return jen.Lit(g.fake.Float64(2, 1, 100)) //nolint: mnd
	}
}

func (g *ClientTestGenerator) typeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	var isPtr bool

	if typeInfo.IsPtr {
		isPtr = true
		typeInfo = typeInfo.ElemType
	}

	switch {
	case typeInfo.IsBasic:
		c := g.basicTypeToValue(typeInfo)
		if isPtr {
			if typeInfo.BasicKind != gomosaic.Int && typeInfo.BasicKind != gomosaic.Uint {
				c = jen.Id(typeInfo.Name).Call(c)
			}
			c = jen.Id("ptr").Call(c)
		}
		return c
	case typeInfo.Named != nil:
		var s jen.Statement
		if isPtr {
			s.Op("&")
		}

		if typeInfo.Named.IsBasic {
			var value any
			if typeInfo.Named.BasicInfo == gomosaic.IsString {
				value = "1"
			}

			return s.Do(g.qualFn(typeInfo.Package, typeInfo.Name)).Call(jen.Lit(value))
		}

		return s.Do(g.qualFn(typeInfo.Package, typeInfo.Name)).Values()
	case typeInfo.IsMap:
		if typeInfo.ElemType.IsBasic {
			return jenutils.TypeInfoQual(typeInfo, g.qualFn).Values(
				jen.Add(g.typeToValue(typeInfo.KeyType)).Op(":").Add(g.typeToValue(typeInfo.ElemType)),
			)
		}
		return jenutils.TypeInfoQual(typeInfo, g.qualFn).Values()
	case typeInfo.IsSlice:
		if typeInfo.ElemType.Struct != nil {
			return jen.Nil()
		}
		return jen.Index().Add(jenutils.TypeInfoQual(typeInfo.ElemType, g.qualFn)).Values()
	case typeInfo.IsArray:
		return jen.Index(jen.Lit(typeInfo.ArrayLen)).Add(jenutils.TypeInfoQual(typeInfo.ElemType, g.qualFn)).Values()
	case typeInfo.Struct != nil:
		return jenutils.TypeInfoQual(typeInfo, g.qualFn).Values()
	case typeInfo.Interface != nil:
		return jen.Nil()
	}

	panic(fmt.Sprintf("unreachable %s", typeInfo.Name))
}

func (g *ClientTestGenerator) genServerResponseGenerate(cfg Config, methodOpt *service.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if !cfg.CheckError && len(methodOpt.BodyResults) > 0 {
		if len(methodOpt.BodyResults) == 1 {
			serverResponse := methodOpt.BodyResults[0]

			serverResponseTypeInfo := serverResponse.Var.Type
			if serverResponseTypeInfo.Named != nil {
				serverResponseTypeInfo = serverResponseTypeInfo.Named
			}

			group.Var().Id("serverResponse").Add(jenutils.TypeInfoQual(serverResponse.Var.Type, g.qualFn)).Line()

			if serverResponseTypeInfo.Struct != nil {
				for _, f := range serverResponseTypeInfo.Struct.Fields {
					for _, v := range flatten.Flatten(f) {
						group.Id("serverResponse").Op(".").Add(v.Path).Op("=").Add(g.typeToValue(v.Var.Type)).Line()
					}
				}
			}
		}
	}

	return group
}

func (g *ClientTestGenerator) genBodyParamsGenerate(methodOpt *service.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if len(methodOpt.BodyParams) > 0 {
		group.Var().Id("serverRequest").StructFunc(func(group *jen.Group) {
			for _, p := range methodOpt.BodyParams {
				group.Id(strcase.ToCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))
			}
		})

		group.Line()

		for _, p := range methodOpt.BodyParams {
			typeInfo := p.Var.Type
			if typeInfo.Named != nil {
				typeInfo = p.Var.Type.Named
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
			} else {
				group.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op("=").Add(g.typeToValue(p.Var.Type)).Line()
			}
		}
	}

	return group
}

func (g *ClientTestGenerator) genParamsGenerate(params []*service.MethodParamOpt) jen.Code {
	group := jen.NewFile("").Null()

	for _, p := range params {
		if p.Var.IsContext {
			continue
		}
		if p.HTTPType == service.BodyHTTPType {
			continue
		}
		postfix := strcase.ToCamel(p.HTTPType)
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name) + postfix).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn)).Op("=").Add(g.typeToValue(p.Var.Type)).Line()
	}

	return group
}

func (g ClientTestGenerator) genMockServerGenerate(methodOpt *service.MethodOpt, errorWrapperName string, cfg Config) jen.Code {
	group := jen.NewFile("").Null()

	pathParts := strings.Split(methodOpt.Path, "/")
	for i, part := range pathParts {
		if strings.HasPrefix(part, ":") {
			pathParts[i] = "{" + part[1:] + "}"
		}
	}

	group.Id("mockServer").Op(":=").Qual("net/http", "NewServeMux").Call().Line()
	group.Id("mockServer").Dot("Handle").Call(
		jen.Lit(strings.Join(pathParts, "/")),
		jen.Qual("net/http", "HandlerFunc").Call(
			jen.Func().Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).BlockFunc(func(group *jen.Group) {
				if len(methodOpt.BodyParams) > 0 {
					group.Var().Id("body").StructFunc(func(group *jen.Group) {
						for _, p := range methodOpt.BodyParams {
							group.Id(strcase.ToCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn)).Tag(map[string]string{
								"json": p.Name,
							})
						}
					})

					var bodyVar jen.Code
					if methodOpt.Single.Req && len(methodOpt.BodyParams) == 1 {
						bodyVar = jen.Op("&").Id("body").Dot(strcase.ToCamel(methodOpt.BodyParams[0].Var.Name))
					} else {
						bodyVar = jen.Op("&").Id("body")
					}

					group.Id("_").Op("=").Qual(service.JSONPkg, "NewDecoder").Call(jen.Id("r").Dot("Body")).Dot("Decode").Call(bodyVar)

					for _, p := range methodOpt.BodyParams {
						typeInfo := p.Var.Type
						if typeInfo.Named != nil {
							typeInfo = typeInfo.Named
						}

						switch {
						default:
							group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).BlockFunc(func(g *jen.Group) {
								g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + p.Name))
							}))
						case typeInfo.Struct != nil:
							for _, f := range typeInfo.Struct.Fields {
								for _, v := range flatten.Flatten(f) {
									if v.IsArray {
										continue
									}
									fieldPath := v.Paths.String()

									if v.Var.Type.IsMap {
										group.If(jen.Op("!").Qual("reflect", "DeepEqual").Call(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)), jen.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name))).BlockFunc(func(g *jen.Group) {
											g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + p.Name))
										}))
									} else {
										group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Op(".").Add(v.Path).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op(".").Add(v.Path)).BlockFunc(func(g *jen.Group) {
											g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath))
										})
									}
								}
							}
						case typeInfo.IsSlice:
							if typeInfo.ElemType.Struct != nil {
								for _, f := range typeInfo.ElemType.Struct.Fields {
									for _, v := range flatten.Flatten(f) {
										if v.IsArray {
											continue
										}
										fieldPath := v.Paths.String()

										switch {
										default:
											group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path)).BlockFunc(func(g *jen.Group) {
												g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath))
											})
										case v.Var.Type.IsBasic:
											if v.Var.Type.IsPtr {
												group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path).Op("==").Nil()).BlockFunc(func(g *jen.Group) {
													g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " is nil"))
												})
												group.If(jen.Op("*").Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path).Op("!=").Op("*").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path)).BlockFunc(func(g *jen.Group) {
													g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath))
												})
											} else {
												group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path)).BlockFunc(func(g *jen.Group) {
													g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath))
												})
											}
										case gomosaic.IsTime(v.Var.Type) || gomosaic.IsDuration(v.Var.Type):
											group.If(jen.Op("!").Qual("reflect", "DeepEqual").Call(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path), jen.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path))).BlockFunc(func(g *jen.Group) {
												g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath))
											})
										}
									}
								}
							} else if typeInfo.ElemType.IsBasic {
								group.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).BlockFunc(func(g *jen.Group) {
									g.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + p.Name))
								}))
							}
						}
					}
				}

				if len(methodOpt.QueryParams) > 0 {
					group.Id("q").Op(":=").Id("r").Dot("URL").Dot("Query").Call()
				}

				for _, p := range methodOpt.Params {
					if p.HTTPType == service.BodyHTTPType {
						continue
					}
					switch p.HTTPType {
					case service.PathHTTPType:
						paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "PathReq")
						group.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

						code := typetransform.For(p.Var.Type).
							SetAssignID(paramID).
							SetValueID(jen.Id("r").Dot("PathValue").Call(jen.Lit(p.Name))).
							SetQualFunc(g.qualFn).
							SetErrStatements(
								jen.Id("t").Dot("Fatal").Call(jen.Err()),
							).Parse()

						group.Add(code)

						group.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Path").Op("!=").Add(paramID).BlockFunc(func(group *jen.Group) {
							group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name)))
						}))
					case service.QueryHTTPType:
						paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "QueryReq")
						group.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

						code := typetransform.For(p.Var.Type).
							SetAssignID(paramID).
							SetValueID(jen.Id("q").Dot("Get").Call(jen.Lit(p.Name))).
							SetQualFunc(g.qualFn).
							SetErrStatements(
								jen.Id("t").Dot("Fatal").Call(jen.Err()),
							).Parse()

						group.Add(code)

						group.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Query").Op("!=").Add(paramID).BlockFunc(func(group *jen.Group) {
							group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name)))
						}))
					case service.HeaderHTTPType:
						paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "HeaderReq")
						group.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

						code := typetransform.For(p.Var.Type).
							SetAssignID(paramID).
							SetValueID(jen.Id("r").Dot("Header").Dot("Get").Call(jen.Lit(p.Name))).
							SetQualFunc(g.qualFn).
							SetErrStatements(
								jen.Id("t").Dot("Fatal").Call(jen.Err()),
							).Parse()

						group.Add(code)

						group.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Header").Op("!=").Add(paramID).BlockFunc(func(group *jen.Group) {
							group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name)))
						}))
					}
				}

				if cfg.StatusCode != 0 {
					group.Id("w").Dot("WriteHeader").Call(jen.Lit(cfg.StatusCode))
				}
				if !cfg.CheckError && len(methodOpt.BodyResults) > 0 {
					group.List(jen.Id("data"), jen.Id("_")).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id("serverResponse"))
					group.Id("w").Dot("Write").Call(jen.Id("data"))
				}

				if errorWrapperName != "" {
					group.List(jen.Id("data"), jen.Id("_")).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id(errorWrapperName))
					group.Id("w").Dot("Write").Call(jen.Id("data"))
				}
			}),
		),
	).Line()

	return group
}

func (g *ClientTestGenerator) genCheckError(methodOpt *service.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("").Null()

	if !gomosaic.HasError(methodOpt.Func.Results) {
		return jen.Null()
	}
	if !cfg.CheckError {
		group.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatalf").Call(jen.Lit("%s: %s"), jen.Lit("failed execute method "+methodOpt.Func.ShortName), jen.Id("err")),
		)

		return group
	}
	group.If(jen.Err().Op("==").Nil()).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed execute method " + methodOpt.Func.ShortName + " error is nil")),
	)

	return group
}

func (g *ClientTestGenerator) genCheckBodyResult(methodOpt *service.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("")

	if cfg.CheckError || len(methodOpt.BodyResults) == 0 {
		return jen.Null()
	}
	for _, r := range methodOpt.BodyResults {
		if r.Var.Type.Named == nil {
			continue
		}

		if r.Var.Type.Named.Struct == nil {
			continue
		}

		st := r.Var.Type.Named.Struct

		for _, f := range st.Fields {
			for _, v := range flatten.Flatten(f) {
				if v.IsArray {
					continue
				}

				fieldPath := v.Paths.String()

				if v.Var.Type.IsPtr {
					group.If(jen.Id(r.Name).Op(".").Add(v.Path).Op("==").Nil()).BlockFunc(func(group *jen.Group) {
						group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " is nil"))
					})
					if v.Var.Type.IsBasic {
						group.If(jen.Op("*").Id(r.Name).Op(".").Add(v.Path).Op("!=").Op("*").Id("serverResponse").Op(".").Add(v.Path)).BlockFunc(func(group *jen.Group) {
							group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " not equal"))
						})
					}
				} else {
					group.If(
						jen.Id(r.Name).Op(".").Add(v.Path).Op("!=").Id("serverResponse").Op(".").Add(v.Path),
					).BlockFunc(func(group *jen.Group) {
						group.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " not equal"))
					})
				}
			}
		}
	}

	return group
}

func (g *ClientTestGenerator) Generate(ifaceOpts []*service.IfaceOpt, configs []Config) jen.Code {
	group := jen.NewFile("")

	group.Func().Id("ptr").Types(jen.Id("T").Any()).Params(jen.Id("t").Id("T")).Op("*").Id("T").Block(
		jen.Return(jen.Op("&").Id("t")),
	)

	for _, ifaceOpt := range ifaceOpts {
		for _, methodOpt := range ifaceOpt.Methods {
			constructName := "create" + methodOpt.Iface.NameTypeInfo.Name + "Client"

			for _, cfg := range configs {
				testMethod := fmt.Sprintf("%s_%d", methodOpt.Func.Name, cfg.StatusCode)
				testName := "Test" + methodOpt.Iface.NameTypeInfo.Name + "_" + testMethod

				group.Func().Id(testName).Params(jen.Id("t").Op("*").Qual("testing", "T")).BlockFunc(func(group *jen.Group) {
					group.Add(g.genServerResponseGenerate(cfg, methodOpt))
					group.Add(g.genBodyParamsGenerate(methodOpt))
					group.Add(g.genParamsGenerate(methodOpt.Params))
					group.Add(g.genMockServerGenerate(methodOpt, "", cfg))

					group.Id("server").Op(":=").Qual("net/http/httptest", "NewServer").Call(
						jen.Id("mockServer"),
					)

					opts := []jen.Code{
						jen.Id("server").Dot("URL"),
						jen.Lit(methodOpt.Func.Name),
						jen.Lit(cfg.StatusCode),
					}

					group.Id("client").Op(":=").Id(constructName).Call(opts...)

					group.Do(func(s *jen.Statement) {
						if len(methodOpt.Func.Results) > 0 {
							s.ListFunc(func(group *jen.Group) {
								for _, r := range methodOpt.Func.Results {
									if r.IsError {
										group.Id(r.Name)
										continue
									}
									if !cfg.CheckError {
										group.Id(r.Name)
									} else {
										group.Id("_")
									}
								}
							})
							if cfg.CheckError || gomosaic.HasError(methodOpt.Func.Results) {
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
							case service.HeaderHTTPType:
								group.Id(name + "Header")
							case service.CookieHTTPType:
								group.Id(name + "Cookie")
							case service.QueryHTTPType:
								group.Id(name + "Query")
							case service.BodyHTTPType:
								group.Id("serverRequest").Dot(strcase.ToCamel(name))
							case service.PathHTTPType:
								group.Id(name + "Path")
							}
						}
					})

					group.Add(g.genCheckError(methodOpt, cfg))
					group.Add(g.genCheckBodyResult(methodOpt, cfg))
				})
			}
		}
	}

	return group
}

func NewClientTest(
	fake faker.Faker,
	qualFn jenutils.QualFunc,
) *ClientTestGenerator {
	return &ClientTestGenerator{
		fake:   fake,
		qualFn: qualFn,
	}
}
