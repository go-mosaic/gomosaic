package client

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

const httpStatusLastSuccessCode = 399

// genExecuteMethod генерирует метод Execute для выполнения HTTP-запроса.
func (g *generator) genExecuteMethod(m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)

	fn := jen.Func().Params(
		jen.Id(recvNameReq).Op("*").Id(reqName),
	).Id("Execute").Params(
		jen.Id("opts").Op("...").Id("ClientOption"),
	)

	fn = fn.ParamsFunc(func(group *jen.Group) {
		for _, result := range m.Results {
			group.Add(jenutils.TypeInfoQual(result.Var.Type, g.qual))
		}
	})

	scopeConstName := strcase.ToLowerCamel(scopeName(m.Iface)) + "ScopeName"
	fullNameConst := strcase.ToLowerCamel(scopeName(m.Iface)) + m.Func.Name + "FullName"
	shortNameConst := strcase.ToLowerCamel(scopeName(m.Iface)) + m.Func.Name + "ShortName"

	fn = fn.BlockFunc(func(body *jen.Group) {
		body.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
			jen.Id("o").Call(jen.Op("&").Id(recvNameClient).Dot("opts")),
		)

		body.List(jen.Id("ctx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(
			jen.Id(recvNameReq).Dot("opts").Dot("ctx"),
		)
		body.Defer().Id("cancel").Call()

		body.Var().Id("span").Qual("go.opentelemetry.io/otel/trace", "Span")
		body.If(
			jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil(),
		).Block(
			jen.List(jen.Id("ctx"), jen.Id("span")).Op("=").Id(recvNameReq).Dot("opts").Dot("tracer").Dot("Start").Call(
				jen.Id("ctx"),
				jen.Id(shortNameConst),
				jen.Qual("go.opentelemetry.io/otel/trace", "WithSpanKind").Call(
					jen.Qual("go.opentelemetry.io/otel/trace", "SpanKindServer"),
				),
			),
			jen.Defer().Id("span").Dot("End").Call(),
		)

		if len(m.PathParams) > 0 {
			g.genPathWithParams(body, m)
		} else {
			body.Id("path").Op(":=").Lit(m.Path)
		}

		body.Id(recvNameReq).Dot("opts").Dot("ctx").Op("=").Qual("context", "WithValue").Call(
			jen.Id(recvNameReq).Dot("opts").Dot("ctx"),
			jen.Id("methodContextKey"),
			jen.Id(fullNameConst),
		)
		body.Id(recvNameReq).Dot("opts").Dot("ctx").Op("=").Qual("context", "WithValue").Call(
			jen.Id(recvNameReq).Dot("opts").Dot("ctx"),
			jen.Id("shortMethodContextKey"),
			jen.Id(shortNameConst),
		)
		body.Id(recvNameReq).Dot("opts").Dot("ctx").Op("=").Qual("context", "WithValue").Call(
			jen.Id(recvNameReq).Dot("opts").Dot("ctx"),
			jen.Id("scopeNameContextKey"),
			jen.Id(scopeConstName),
		)

		body.List(jen.Id("req"), jen.Err()).Op(":=").Qual("net/http", "NewRequestWithContext").Call(
			jen.Id(recvNameReq).Dot("opts").Dot("ctx"), jen.Lit(m.Method),
			jen.Id(recvNameReq).Dot("c").Dot("target").Op("+").Id("path"), jen.Nil(),
		)
		body.If(jen.Err().Op("!=").Nil()).BlockFunc(func(errBody *jen.Group) {
			errBody.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
				jen.Id("span").Dot("AddEvent").Call(
					jen.Lit("request make error"),
					jen.Qual("go.opentelemetry.io/otel/trace", "WithAttributes").Call(
						jen.Qual("go.opentelemetry.io/otel/attribute", "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call()),
					),
				),
				jen.Id("span").Dot("SetStatus").Call(
					jen.Qual("go.opentelemetry.io/otel/codes", "Error"),
					jen.Lit("failed sent request"),
				),
			)
			errBody.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...)
		})

		body.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("application/json"))

		body.If(jen.Id(recvNameReq).Dot("opts").Dot("propagator").Op("!=").Nil()).Block(
			jen.Id(recvNameReq).Dot("opts").Dot("propagator").Dot("Inject").Call(
				jen.Id("ctx"),
				jen.Qual("go.opentelemetry.io/otel/propagation", "HeaderCarrier").Call(jen.Id("req").Dot("Header")),
			),
		)

		if len(m.BodyParams) > 0 {
			g.genJSONReqContent(body, m)
		}

		if len(m.QueryParams) > 0 {
			g.genQueryParams(body, m)
		}

		if len(m.HeaderParams) > 0 {
			g.genHeaderParams(body, m)
		}

		if len(m.CookieParams) > 0 {
			g.genCookieParams(body, m)
		}

		body.Id("before").Op(":=").Append(jen.Id(recvNameReq).Dot("c").Dot("opts").Dot("before"), jen.Id(recvNameReq).Dot("opts").Dot("before").Op("..."))
		body.For(jen.List(jen.Id("_"), jen.Id("before")).Op(":=").Range().Id("before")).Block(
			jen.List(jen.Id("ctx"), jen.Err()).Op("=").Id("before").Call(jen.Id("ctx"), jen.Id("req")),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
			),
		)

		body.List(jen.Id("resp"), jen.Err()).Op(":=").Id(recvNameReq).Dot("opts").Dot("client").Dot("Do").Call(jen.Id("req"))
		body.If(jen.Err().Op("!=").Nil()).BlockFunc(func(errBody *jen.Group) {
			errBody.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
				jen.Id("span").Dot("AddEvent").Call(
					jen.Lit("do request error"),
					jen.Qual("go.opentelemetry.io/otel/trace", "WithAttributes").Call(
						jen.Qual("go.opentelemetry.io/otel/attribute", "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call()),
					),
				),
				jen.Id("span").Dot("SetStatus").Call(
					jen.Qual("go.opentelemetry.io/otel/codes", "Error"),
					jen.Lit("failed sent request"),
				),
			)
			errBody.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...)
		})

		body.Id("after").Op(":=").Append(jen.Id(recvNameReq).Dot("c").Dot("opts").Dot("after"), jen.Id(recvNameReq).Dot("opts").Dot("after").Op("..."))
		body.For(jen.List(jen.Id("_"), jen.Id("after")).Op(":=").Range().Id("after")).Block(
			jen.Id("ctx").Op("=").Id("after").Call(jen.Id("ctx"), jen.Id("resp")),
		)

		body.Defer().Id("resp").Dot("Body").Dot("Close").Call()
		body.Defer().Id("cancel").Call()

		body.If(jen.Id("resp").Dot("StatusCode").Op(">").Lit(httpStatusLastSuccessCode)).BlockFunc(func(errBody *jen.Group) {
			errBody.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
				jen.Id("span").Dot("AddEvent").Call(
					jen.Lit("response status code failed"),
					jen.Qual("go.opentelemetry.io/otel/trace", "WithAttributes").Call(
						jen.Qual("go.opentelemetry.io/otel/attribute", "String").Call(jen.Lit("reason"), jen.Id("resp").Dot("Status")),
					),
				),
				jen.Id("span").Dot("SetStatus").Call(
					jen.Qual("go.opentelemetry.io/otel/codes", "Error"),
					jen.Lit("failed response"),
				),
			)
			errBody.If(jen.Id(recvNameReq).Dot("opts").Dot("errorDecode").Op("!=").Nil()).Block(
				jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual,
					jen.Id(recvNameReq).Dot("opts").Dot("errorDecode").Call(
						jen.Id("resp").Dot("Body"),
						jen.Id("resp").Dot("StatusCode"),
					),
				)...),
			)
			errBody.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual,
				jen.Id("errorDecode").Call(
					jen.Id("resp").Dot("Body"),
					jen.Id("resp").Dot("StatusCode"),
					jen.Func().Params(jen.Id("data").Index().Byte()).Error().Block(
						jen.Return(jen.Op("&").Id("ClientError").Values(jen.Dict{
							jen.Id("Data"):       jen.Id("data"),
							jen.Id("StatusCode"): jen.Id("resp").Dot("StatusCode"),
						})),
					),
				),
			)...)
		})

		if len(m.BodyResults) > 0 {
			g.genGzipReader(body, m)
			g.genReadResponse(body, m)
		} else {
			body.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
				jen.Id("span").Dot("SetStatus").Call(
					jen.Qual("go.opentelemetry.io/otel/codes", "Ok"),
					jen.Lit("request sent successfully"),
				),
			)
			body.Return(jen.Nil())
		}
	})

	return fn
}

func (g *generator) genPathWithParams(body *jen.Group, m *annotation.MethodOpt) {
	pathParts := make([]jen.Code, 0)
	pathParts = append(pathParts, jen.Lit(sprintfPath(m)))
	for _, p := range m.PathParams {
		pathParts = append(pathParts, jen.Id(recvNameReq).Dot(p.PathParamName))
	}
	body.Id("path").Op(":=").Qual("fmt", "Sprintf").Call(pathParts...)
}

func (g *generator) genJSONReqContent(body *jen.Group, m *annotation.MethodOpt) {
	body.Id("req").Dot("Header").Dot("Add").Call(jen.Lit("Content-Type"), jen.Lit("application/json"))
	body.Var().Id("reqData").Qual("bytes", "Buffer")

	body.If(
		jen.Err().Op(":=").Qual("encoding/json", "NewEncoder").Call(jen.Op("&").Id("reqData")).Dot("Encode").Call(
			jen.Id(recvNameReq).Dot("makeBodyRequest").Call(),
		),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
			jen.Id("span").Dot("AddEvent").Call(
				jen.Lit("JSON encode error"),
				jen.Qual("go.opentelemetry.io/otel/trace", "WithAttributes").Call(
					jen.Qual("go.opentelemetry.io/otel/attribute", "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call()),
				),
			),
			jen.Id("span").Dot("SetStatus").Call(
				jen.Qual("go.opentelemetry.io/otel/codes", "Error"),
				jen.Lit("failed sent request"),
			),
		),
		jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
	)

	body.Id("req").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Op("&").Id("reqData"))
}

func (g *generator) genQueryParams(body *jen.Group, m *annotation.MethodOpt) {
	body.Id("q").Op(":=").Id("req").Dot("URL").Dot("Query").Call()
	for _, p := range m.QueryParams {
		body.Id("q").Dot("Set").Call(jen.Lit(p.Name), jen.Qual("fmt", "Sprintf").Call(
			jen.Lit("%v"), jen.Id(recvNameReq).Dot(strcase.ToLowerCamel(p.Var.Name)),
		))
	}
	body.Id("req").Dot("URL").Dot("RawQuery").Op("=").Id("q").Dot("Encode").Call()
}

func (g *generator) genHeaderParams(body *jen.Group, m *annotation.MethodOpt) {
	for _, p := range m.HeaderParams {
		body.Id("req").Dot("Header").Dot("Set").Call(
			jen.Lit(p.Name),
			jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(recvNameReq).Dot(strcase.ToLowerCamel(p.Var.Name))),
		)
	}
}

func (g *generator) genCookieParams(body *jen.Group, m *annotation.MethodOpt) {
	for _, p := range m.CookieParams {
		body.Id("req").Dot("AddCookie").Call(
			jen.Qual("net/http", "Cookie").Values(jen.Dict{
				jen.Id("Name"):  jen.Lit(p.Name),
				jen.Id("Value"): jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(recvNameReq).Dot(strcase.ToLowerCamel(p.Var.Name))),
			}),
		)
	}
}

func (g *generator) genGzipReader(body *jen.Group, m *annotation.MethodOpt) {
	body.Var().Id("reader").Qual("io", "ReadCloser")
	body.If(
		jen.Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("Content-Encoding")).Op("==").Lit("gzip"),
	).Block(
		jen.List(jen.Id("reader"), jen.Err()).Op("=").Qual("compress/gzip", "NewReader").Call(jen.Id("resp").Dot("Body")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
		),
		jen.Defer().Id("reader").Dot("Close").Call(),
	).Else().Block(
		jen.Id("reader").Op("=").Id("resp").Dot("Body"),
	)
}

func (g *generator) genReadResponse(body *jen.Group, m *annotation.MethodOpt) {
	if len(m.BodyResults) == 1 && m.Single.Resp {
		body.Var().Id("respBody").Add(jenutils.TypeInfoQual(m.BodyResults[0].Var.Type, g.qual))
	} else {
		body.Var().Id("respBody").Struct(annotation.MakeStructFieldsFromResults(m.BodyResults, g.qual))
	}

	body.If(jen.Err().Op("=").Qual("encoding/json", "NewDecoder").Call(
		jen.Id("reader"),
	).Dot("Decode").Call(jen.Op("&").Id("respBody")), jen.Err().Op("!=").Nil()).Block(
		jen.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
			jen.Id("span").Dot("AddEvent").Call(
				jen.Lit("JSON decode error"),
				jen.Qual("go.opentelemetry.io/otel/trace", "WithAttributes").Call(
					jen.Qual("go.opentelemetry.io/otel/attribute", "String").Call(jen.Lit("reason"), jen.Err().Dot("Error").Call()),
				),
			),
			jen.Id("span").Dot("SetStatus").Call(
				jen.Qual("go.opentelemetry.io/otel/codes", "Error"),
				jen.Lit("failed read response"),
			),
		),
		jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
	)

	body.If(jen.Id(recvNameReq).Dot("opts").Dot("tracer").Op("!=").Nil()).Block(
		jen.Id("span").Dot("SetStatus").Call(
			jen.Qual("go.opentelemetry.io/otel/codes", "Ok"),
			jen.Lit("request sent successfully"),
		),
	)

	body.ReturnFunc(func(ret *jen.Group) {
		if len(m.BodyResults) == 1 && m.Single.Resp {
			ret.Id("respBody")
		} else {
			for _, result := range m.BodyResults {
				if result.Var.IsError {
					continue
				}
				ret.Id("respBody").Dot(strcase.ToCamel(result.Var.Name))
			}
		}
		ret.Nil()
	})
}
