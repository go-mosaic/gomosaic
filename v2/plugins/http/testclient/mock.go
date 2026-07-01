package testclient

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/flatten"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// genMockServerGenerate — генерирует mock-сервер с проверкой входящих параметров.
func (g *generator) genMockServerGenerate(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
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
							group.Id(strcase.ToCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual)).Tag(map[string]string{
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

func (g *generator) genCompareStructFields(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt, typeInfo *gomosaic.TypeInfo) {
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

func (g *generator) genCompareSliceFields(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt, typeInfo *gomosaic.TypeInfo) {
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

func (g *generator) genComparePathParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "PathReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("r").Dot("PathValue").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qual).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Path").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

func (g *generator) genCompareQueryParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "QueryReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("q").Dot("Get").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qual).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Query").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

func (g *generator) genCompareHeaderParam(body *jen.Group, methodOpt *annotation.MethodOpt, p *annotation.MethodParamOpt) {
	paramID := jen.Id(strcase.ToLowerCamel(p.Var.Name) + "HeaderReq")
	body.Var().Add(paramID).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

	tr := gomosaic.DefaultTransformRegistry().For(p.Var.Type).
		SetAssignID(paramID).
		SetValueID(jen.Id("r").Dot("Header").Dot("Get").Call(jen.Lit(p.Name))).
		SetQualFunc(g.qual).
		SetErrStatements(jen.Id("t").Dot("Fatal").Call(jen.Err()))
	body.Add(tr.Parse())

	body.If(jen.Id(strcase.ToLowerCamel(p.Var.Name) + "Header").Op("!=").Add(paramID)).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + strcase.ToLowerCamel(p.Var.Name))),
	)
}

// buildResponseData — строит структуру для маршалинга ответа.
func (g *generator) buildResponseData(methodOpt *annotation.MethodOpt) jen.Code {
	if len(methodOpt.BodyResults) == 1 && methodOpt.Single.Resp {
		return jen.Id("serverResponse")
	}

	innerStruct := annotation.MakeStructFieldsFromResults(methodOpt.BodyResults, g.qual)
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
