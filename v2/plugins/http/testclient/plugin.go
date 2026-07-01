// Package testclient предоставляет плагин генерации тестов HTTP-клиента.
// Портирован из v1 с полной поддержкой flatten, сравнения параметров и faker.
package testclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/flatten"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

type Config struct {
	StatusCode int
	CheckError bool
}

const FakerPkg = "github.com/jaswdr/faker/v2"

type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "http-client-test" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir, gomosaic.UseTestPkg())
	f.ImportAlias(FakerPkg, "faker")

	gen := NewGenerator(f.Qual)
	f.Add(gen.Generate(annotations, []Config{
		{StatusCode: 200},
		{StatusCode: 400, CheckError: true},
	}))

	return map[string]gomosaic.File{"client_gen_test.go": f}, nil
}

type Generator struct {
	qualFn gomosaic.QualFunc
}

func NewGenerator(qualFn gomosaic.QualFunc) *Generator {
	return &Generator{qualFn: qualFn}
}

func (g *Generator) Generate(ifaceOpts []*annotation.IfaceOpt, configs []Config) jen.Code {
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
func (g *Generator) genServerResponseGenerate(cfg Config, methodOpt *annotation.MethodOpt) jen.Code {
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
				// Генерируем x := &Type{Field: faker, ...}
				if serverResponse.Var.Type.IsPtr {
					group.Id("serverResponse").Op(":=").Op("&").Add(jenutils.TypeInfoQual(serverResponse.Var.Type.ElemType, g.qualFn)).ValuesFunc(func(dict *jen.Group) {
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
					group.Var().Id("serverResponse").Add(jenutils.TypeInfoQual(serverResponse.Var.Type, g.qualFn)).Line()
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
func (g *Generator) genBodyParamsGenerate(methodOpt *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("").Null()

	if len(methodOpt.BodyParams) > 0 {
		group.Var().Id("serverRequest").StructFunc(func(group *jen.Group) {
			for _, p := range methodOpt.BodyParams {
				group.Id(strcase.ToCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))
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
				// Указатель на структуру: x := &Type{Field: faker, ...}
				innerType := p.Var.Type.ElemType.ElemType
				group.Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op("=").Op("&").Add(jenutils.TypeInfoQual(p.Var.Type.ElemType, g.qualFn)).ValuesFunc(func(dict *jen.Group) {
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
func (g *Generator) genParamsGenerate(params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("").Null()

	for _, p := range params {
		if p.Var.IsContext || p.HTTPType == annotation.BodyHTTPType {
			continue
		}
		postfix := strcase.ToCamel(p.HTTPType)
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name) + postfix).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn)).Op("=").Add(g.typeToValue(p.Var.Type)).Line()
	}
	return group
}

// genMockServerGenerate — генерирует mock-сервер с проверкой входящих параметров.
func (g *Generator) genMockServerGenerate(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
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
			).BlockFunc(func(body *jen.Group) {
				// Декодируем тело запроса и сравниваем
				if len(methodOpt.BodyParams) > 0 {
					body.Var().Id("body").StructFunc(func(group *jen.Group) {
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

					body.Id("_").Op("=").Qual("encoding/json", "NewDecoder").Call(jen.Id("r").Dot("Body")).Dot("Decode").Call(bodyVar)

					// Сравниваем каждое body-поле
					for _, p := range methodOpt.BodyParams {
						typeInfo := p.Var.Type
						if typeInfo.IsPtr {
							typeInfo = typeInfo.ElemType
						}
						if typeInfo.IsNamed {
							typeInfo = typeInfo.ElemType
						}

						switch {
						case typeInfo.Struct != nil:
							g.genCompareStructFields(body, methodOpt, p, typeInfo)
						case typeInfo.IsSlice:
							g.genCompareSliceFields(body, methodOpt, p, typeInfo)
						default:
							body.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name))).Block(
								jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + p.Name)),
							)
						}
					}
				}

				// Сравниваем Query/Path/Header параметры
				if len(methodOpt.QueryParams) > 0 {
					body.Id("q").Op(":=").Id("r").Dot("URL").Dot("Query").Call()
				}
				for _, p := range methodOpt.Params {
					if p.HTTPType == annotation.BodyHTTPType {
						continue
					}
					switch p.HTTPType {
					case annotation.PathHTTPType:
						g.genComparePathParam(body, methodOpt, p)
					case annotation.QueryHTTPType:
						g.genCompareQueryParam(body, methodOpt, p)
					case annotation.HeaderHTTPType:
						g.genCompareHeaderParam(body, methodOpt, p)
					}
				}

				// Отправляем ответ
				if cfg.StatusCode != 0 {
					body.Id("w").Dot("WriteHeader").Call(jen.Lit(cfg.StatusCode))
				}
				if !cfg.CheckError && len(methodOpt.BodyResults) > 0 {
					responseData := g.buildResponseData(methodOpt)
					body.List(jen.Id("data"), jen.Id("_")).Op(":=").Qual("encoding/json", "Marshal").Call(responseData)
					body.Id("w").Dot("Write").Call(jen.Id("data"))
				}
			}),
		),
	).Line()

	return group
}

func (g *Generator) genCompareStructFields(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt, typeInfo *gomosaic.TypeInfo) {
	for _, f := range typeInfo.Struct.Fields {
		for _, v := range flatten.Flatten(f) {
			if v.IsArray {
				continue
			}
			fieldPath := v.Paths.String()
			body.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Op(".").Add(v.Path).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Op(".").Add(v.Path)).Block(
				jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath)),
			)
		}
	}
}

func (g *Generator) genCompareSliceFields(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt, typeInfo *gomosaic.TypeInfo) {
	if typeInfo.ElemType.Struct != nil {
		for _, f := range typeInfo.ElemType.Struct.Fields {
			for _, v := range flatten.Flatten(f) {
				if v.IsArray {
					continue
				}
				fieldPath := v.Paths.String()
				body.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op(".").Add(v.Path)).Block(
					jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath)),
				)
			}
		}
	} else if typeInfo.ElemType.IsBasic {
		body.If(jen.Id("body").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0)).Op("!=").Id("serverRequest").Dot(strcase.ToCamel(p.Var.Name)).Index(jen.Lit(0))).Block(
			jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + p.Name)),
		)
	}
}

func (g *Generator) genComparePathParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "PathReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

	// Используем gomosaic.For вместо typetransform.For (из v2)
	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("r").Dot("PathValue").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qualFn).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Path").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

func (g *Generator) genCompareQueryParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "QueryReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("q").Dot("Get").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qualFn).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Query").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

func (g *Generator) genCompareHeaderParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "HeaderReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qualFn))

	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("r").Dot("Header").Dot("Get").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qualFn).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Header").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

// buildResponseData — строит структуру для маршалинга ответа.
func (g *Generator) buildResponseData(methodOpt *annotation.MethodOpt) jen.Code {
	if len(methodOpt.BodyResults) == 1 && methodOpt.Single.Resp {
		return jen.Id("serverResponse")
	}

	innerStruct := annotation.MakeStructFieldsFromResults(methodOpt.BodyResults, g.qualFn)
	innerValue := jen.DictFunc(func(d jen.Dict) {
		for _, r := range methodOpt.BodyResults {
			d[jen.Id(strcase.ToCamel(r.Var.Name))] = jen.Id("serverResponse")
		}
	})

	if len(methodOpt.WrapResp.PathParts) == 0 {
		return jen.StructFunc(func(g *jen.Group) { g.Add(innerStruct) }).Values(innerValue)
	}

	typeCode := innerStruct
	valueDict := innerValue
	for i := len(methodOpt.WrapResp.PathParts) - 1; i >= 0; i-- {
		partName := methodOpt.WrapResp.PathParts[i]
		camelName := strcase.ToCamel(partName)
		typeCode = jen.Id(camelName).Struct(typeCode).Tag(map[string]string{"json": partName})
		valueDict = jen.Dict{jen.Id(camelName): jen.Values(valueDict)}
	}

	return jen.StructFunc(func(g *jen.Group) { g.Add(typeCode) }).Values(valueDict)
}

func (g *Generator) genCheckError(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("").Null()

	errVar, hasErr := gomosaic.HasError(methodOpt.Func.Results)
	if !hasErr {
		return group
	}
	if !cfg.CheckError {
		group.If(jen.Id(errVar.Name).Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatalf").Call(jen.Lit("%s: %s"), jen.Lit("failed execute method "+methodOpt.Func.ShortName), jen.Id(errVar.Name)),
		)
		return group
	}
	group.If(jen.Id(errVar.Name).Op("==").Nil()).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed execute method " + methodOpt.Func.ShortName + " error is nil")),
	)
	return group
}

func (g *Generator) genCheckBodyResult(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("")

	if cfg.CheckError || len(methodOpt.BodyResults) == 0 {
		return jen.Null()
	}
	for _, r := range methodOpt.BodyResults {
		if !r.Var.Type.IsNamed {
			continue
		}
		if r.Var.Type.ElemType.Struct == nil {
			continue
		}
		st := r.Var.Type.ElemType.Struct
		for _, f := range st.Fields {
			for _, v := range flatten.Flatten(f) {
				if v.IsArray {
					continue
				}
				fieldPath := v.Paths.String()
				group.If(jen.Id(r.Var.Name).Op(".").Add(v.Path).Op("!=").Id("serverResponse").Op(".").Add(v.Path)).Block(
					jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " not equal")),
				)
			}
		}
	}
	return group
}

// typeToValue — генерирует тестовое значение через faker.
func (g *Generator) typeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	var isPtr bool
	if typeInfo.IsPtr {
		isPtr = true
		typeInfo = typeInfo.ElemType
	}

	switch {
	case typeInfo.IsBasic:
		c := g.basicTypeToValue(typeInfo)
		if isPtr {
			c = jen.Id("ptr").Call(c)
		}
		return c
	case typeInfo.IsNamed:
		var s jen.Statement
		if isPtr {
			s.Op("&")
		}
		if typeInfo.ElemType.IsBasic {
			var value any
			if typeInfo.ElemType.BasicInfo == gomosaic.IsString {
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
	return jen.Qual(FakerPkg, "New").Call().Dot("Lorem").Call().Dot("Sentence").Call(jen.Lit(10))
}

func (g *Generator) basicTypeToValue(typeInfo *gomosaic.TypeInfo) jen.Code {
	switch typeInfo.BasicInfo {
	case gomosaic.IsBoolean:
		return jen.Lit(true)
	case gomosaic.IsInteger, gomosaic.IsInteger | gomosaic.IsUnsigned:
		return jen.Qual(FakerPkg, "New").Call().Dot("RandomNumber").Call(jen.Lit(5))
	case gomosaic.IsFloat:
		return jen.Qual(FakerPkg, "New").Call().Dot("Float64").Call(jen.Lit(2), jen.Lit(1), jen.Lit(100))
	default:
		return jen.Qual(FakerPkg, "New").Call().Dot("Lorem").Call().Dot("Sentence").Call(jen.Lit(10))
	}
}

var _ gomosaic.Generator = (*Plugin)(nil)
