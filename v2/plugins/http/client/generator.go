package client

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

const (
	recvNameReq    = "r"
	recvNameClient = "c"
)

type generator struct {
	qual       gomosaic.QualFunc
	modulePath string
}

func (g *generator) Generate(services []*annotation.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")

	group.Add(g.genClientTypes())

	for _, s := range services {
		group.Add(g.genMethodConstants(s))
		group.Add(g.genClientStruct(s))
		group.Add(g.genClientConstruct(s))
		for _, m := range s.Methods {
			group.Add(g.genClientMethod(s, m))
		}
	}
	return group, nil
}

func clientStructName(ifaceOpt *annotation.IfaceOpt) string {
	return ifaceOpt.NameTypeInfo.Name + "Client"
}

func methodRequestName(methodOpt *annotation.MethodOpt) string {
	return strcase.ToCamel(methodOpt.Iface.NameTypeInfo.Name) + methodOpt.Func.Name + "Request"
}

func methodShortName(m *annotation.MethodOpt) string {
	return "(" + m.Iface.NameTypeInfo.Package.Name + "." + m.Iface.NameTypeInfo.Name + ")." + m.Func.Name
}

func methodFullName(m *annotation.MethodOpt) string {
	pkg := m.Iface.NameTypeInfo.Package
	return "(" + pkg.Path + "." + m.Iface.NameTypeInfo.Name + ")." + m.Func.Name
}

func scopeName(ifaceOpt *annotation.IfaceOpt) string {
	return ifaceOpt.NameTypeInfo.Package.Name
}

func sprintfPath(methodOpt *annotation.MethodOpt) string {
	parts := splitPath(methodOpt.Path)
	for i, part := range parts {
		if len(part) > 0 && (part[0] == ':' || (part[0] == '{' && part[len(part)-1] == '}')) {
			parts[i] = "%s"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}

func joinPath(parts []string) string {
	var result strings.Builder

	for i, p := range parts {
		if i > 0 {
			result.WriteString("/")
		}

		result.WriteString(p)
	}

	return result.String()
}

// genClientTypes генерирует вспомогательные типы клиента.
func (g *generator) genClientTypes() jen.Code {
	f := jen.NewFile("")

	f.Type().Id("contextKey").String()
	f.Line()
	f.Const().DefsFunc(func(g *jen.Group) {
		g.Id("methodContextKey").Id("contextKey").Op("=").Lit("method")
		g.Id("shortMethodContextKey").Id("contextKey").Op("=").Lit("shortMethod")
		g.Id("scopeNameContextKey").Id("contextKey").Op("=").Lit("scopeName")
	})
	f.Line()

	f.Func().Id("labelFromContext").Params(
		jen.Id("lblName").String(),
		jen.Id("ctxKey").Id("contextKey"),
	).Qual("github.com/prometheus/client_golang/prometheus/promhttp", "Option").Block(
		jen.Return(
			jen.Qual("github.com/prometheus/client_golang/prometheus/promhttp", "WithLabelFromCtx").Call(
				jen.Id("lblName"),
				jen.Func().Params(jen.Id("ctx").Qual("context", "Context")).String().Block(
					jen.Id("v").Op(",").Id("_").Op(":=").Id("ctx").Dot("Value").Call(jen.Id("ctxKey")).Assert(jen.String()),
					jen.Return(jen.Id("v")),
				),
			),
		),
	)
	f.Line()

	f.Func().Id("instrumentRoundTripperErrCounter").Params(
		jen.Id("counter").Op("*").Qual("github.com/prometheus/client_golang/prometheus", "CounterVec"),
		jen.Id("next").Qual("net/http", "RoundTripper"),
	).Qual("github.com/prometheus/client_golang/prometheus/promhttp", "RoundTripperFunc").Block(
		jen.Return(
			jen.Func().Params(jen.Id("r").Op("*").Qual("net/http", "Request")).Params(
				jen.Op("*").Qual("net/http", "Response"),
				jen.Error(),
			).BlockFunc(func(body *jen.Group) {
				body.Id("labels").Op(":=").Qual("github.com/prometheus/client_golang/prometheus", "Labels").Values(jen.Dict{
					jen.Lit("method"): jen.Qual("strings", "ToLower").Call(jen.Id("r").Dot("Method")),
				})
				body.List(jen.Id("labels").Index(jen.Lit("methodNameFull")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("methodContextKey")).Assert(jen.String())
				body.List(jen.Id("labels").Index(jen.Lit("methodNameShort")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("shortMethodContextKey")).Assert(jen.String())
				body.List(jen.Id("labels").Index(jen.Lit("scopeName")), jen.Id("_")).Op("=").Id("r").Dot("Context").Call().Dot("Value").Call(jen.Id("scopeNameContextKey")).Assert(jen.String())
				body.Id("labels").Index(jen.Lit("code")).Op("=").Lit("")
				body.List(jen.Id("resp"), jen.Err()).Op(":=").Id("next").Dot("RoundTrip").Call(jen.Id("r"))
				body.If(jen.Err().Op("!=").Nil()).BlockFunc(func(errBody *jen.Group) {
					errBody.Var().Id("errType").String()
					errBody.Switch(jen.Id("e").Op(":=").Err().Assert(jen.Type())).BlockFunc(func(sw *jen.Group) {
						sw.Default().Block(
							jen.Id("errType").Op("=").Err().Dot("Error").Call(),
						)
						sw.Case(jen.Op("*").Qual("crypto/tls", "CertificateVerificationError")).Block(
							jen.Id("errType").Op("=").Lit("failedVerifyCertificate"),
						)
						sw.Case(jen.Qual("net", "Error")).BlockFunc(func(netBody *jen.Group) {
							netBody.Id("errType").Op("+=").Lit("net.")
							netBody.If(jen.Id("e").Dot("Timeout").Call()).Block(
								jen.Id("errType").Op("+=").Lit("timeout."),
							)
							netBody.Switch(jen.Id("ee").Op(":=").Id("e").Assert(jen.Type())).BlockFunc(func(netSw *jen.Group) {
								netSw.Case(jen.Op("*").Qual("net", "ParseError")).Block(
									jen.Id("errType").Op("+=").Lit("parse"),
								)
								netSw.Case(jen.Op("*").Qual("net", "InvalidAddrError")).Block(
									jen.Id("errType").Op("+=").Lit("invalidAddr"),
								)
								netSw.Case(jen.Op("*").Qual("net", "UnknownNetworkError")).Block(
									jen.Id("errType").Op("+=").Lit("unknownNetwork"),
								)
								netSw.Case(jen.Op("*").Qual("net", "DNSError")).Block(
									jen.Id("errType").Op("+=").Lit("dns"),
								)
								netSw.Case(jen.Op("*").Qual("net", "OpError")).Block(
									jen.Id("errType").Op("+=").Id("ee").Dot("Net").Op("+").Lit(".").Op("+").Id("ee").Dot("Op"),
								)
							})
						})
					})
					errBody.Id("labels").Index(jen.Lit("errorCode")).Op("=").Id("errType")
					errBody.Id("counter").Dot("With").Call(jen.Id("labels")).Dot("Add").Call(jen.Lit(1))
				}).Else().If(jen.Id("resp").Dot("StatusCode").Op(">").Lit(httpStatusLastSuccessCode)).Block(
					jen.Id("labels").Index(jen.Lit("code")).Op("=").Qual("strconv", "Itoa").Call(jen.Id("resp").Dot("StatusCode")),
					jen.Id("labels").Index(jen.Lit("errorCode")).Op("=").Lit("respFailed"),
					jen.Id("counter").Dot("With").Call(jen.Id("labels")).Dot("Add").Call(jen.Lit(1)),
				)
				body.Return(jen.Id("resp"), jen.Err())
			}),
		),
	)
	f.Line()

	f.Type().Id("prometheusCollector").Interface(
		jen.Qual("github.com/prometheus/client_golang/prometheus", "Collector"),
		jen.Id("Requests").Params().Op("*").Qual("github.com/prometheus/client_golang/prometheus", "CounterVec"),
		jen.Id("ErrRequests").Params().Op("*").Qual("github.com/prometheus/client_golang/prometheus", "CounterVec"),
		jen.Id("Duration").Params().Op("*").Qual("github.com/prometheus/client_golang/prometheus", "HistogramVec"),
	)
	f.Line()

	f.Type().Id("ClientBeforeFunc").Func().Params(
		jen.Qual("context", "Context"),
		jen.Op("*").Qual("net/http", "Request"),
	).Params(jen.Qual("context", "Context"), jen.Error())
	f.Line()
	f.Type().Id("ClientAfterFunc").Func().Params(
		jen.Qual("context", "Context"),
		jen.Op("*").Qual("net/http", "Response"),
	).Qual("context", "Context")
	f.Line()

	f.Type().Id("ClientError").Struct(
		jen.Id("Data").Index().Byte(),
		jen.Id("StatusCode").Int(),
	)
	f.Line()
	f.Func().Params(jen.Id("e").Op("*").Id("ClientError")).Id("Error").Params().String().Block(
		jen.Return(jen.Qual("fmt", "Sprint").Call(jen.Id("e").Dot("StatusCode")).Op("+").Lit(": ").Op("+").String().Call(jen.Id("e").Dot("Data"))),
	)
	f.Line()

	f.Type().Id("ErrorDecoder").Func().Params(
		jen.Qual("io", "ReadCloser"),
		jen.Int(),
	).Error()
	f.Line()

	f.Func().Id("errorDecode").Params(
		jen.Id("body").Qual("io", "ReadCloser"),
		jen.Id("statusCode").Int(),
		jen.Id("errorFn").Func().Params(jen.Index().Byte()).Error(),
	).Error().Block(
		jen.List(jen.Id("data"), jen.Err()).Op(":=").Qual("io", "ReadAll").Call(jen.Id("body")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("%d: failed to read error body: %w"), jen.Id("statusCode"), jen.Err())),
		),
		jen.Return(jen.Id("errorFn").Call(jen.Id("data"))),
	)
	f.Line()

	f.Type().Id("clientOptions").Struct(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("content").String(),
		jen.Id("tracer").Qual("go.opentelemetry.io/otel/trace", "Tracer"),
		jen.Id("propagator").Qual("go.opentelemetry.io/otel/propagation", "TextMapPropagator"),
		jen.Id("before").Index().Id("ClientBeforeFunc"),
		jen.Id("after").Index().Id("ClientAfterFunc"),
		jen.Id("errorDecode").Id("ErrorDecoder"),
		jen.Id("client").Op("*").Qual("net/http", "Client"),
	)
	f.Line()

	f.Type().Id("ClientOption").Func().Params(jen.Op("*").Id("clientOptions"))
	f.Line()

	f.Func().Id("WithTracer").Params(
		jen.Id("tracer").Qual("go.opentelemetry.io/otel/trace", "Tracer"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("tracer").Op("=").Id("tracer"),
		)),
	)
	f.Line()

	f.Func().Id("WithPropagator").Params(
		jen.Id("propagator").Qual("go.opentelemetry.io/otel/propagation", "TextMapPropagator"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("propagator").Op("=").Id("propagator"),
		)),
	)
	f.Line()

	f.Func().Id("WithContent").Params(
		jen.Id("content").String(),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("content").Op("=").Id("content"),
		)),
	)
	f.Line()

	f.Func().Id("WithContext").Params(
		jen.Id("ctx").Qual("context", "Context"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("ctx").Op("=").Id("ctx"),
		)),
	)
	f.Line()

	f.Func().Id("WithHTTPClient").Params(
		jen.Id("client").Op("*").Qual("net/http", "Client"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("client").Op("=").Id("client"),
		)),
	)
	f.Line()

	f.Func().Id("WithPromCollector").Params(
		jen.Id("c").Id("prometheusCollector"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).BlockFunc(func(body *jen.Group) {
			body.If(jen.Id("o").Dot("client").Dot("Transport").Op("==").Nil()).Block(
				jen.Panic(jen.Lit("no transport is set for the http client")),
			)
			body.Id("o").Dot("client").Dot("Transport").Op("=").Id("instrumentRoundTripperErrCounter").Call(
				jen.Id("c").Dot("ErrRequests").Call(),
				jen.Qual("github.com/prometheus/client_golang/prometheus/promhttp", "InstrumentRoundTripperCounter").Call(
					jen.Id("c").Dot("Requests").Call(),
					jen.Qual("github.com/prometheus/client_golang/prometheus/promhttp", "InstrumentRoundTripperDuration").Call(
						jen.Id("c").Dot("Duration").Call(),
						jen.Id("o").Dot("client").Dot("Transport"),
						jen.Id("labelFromContext").Call(jen.Lit("methodNameShort"), jen.Id("shortMethodContextKey")),
						jen.Id("labelFromContext").Call(jen.Lit("methodNameFull"), jen.Id("methodContextKey")),
						jen.Id("labelFromContext").Call(jen.Lit("scopeName"), jen.Id("scopeNameContextKey")),
					),
					jen.Id("labelFromContext").Call(jen.Lit("methodNameShort"), jen.Id("shortMethodContextKey")),
					jen.Id("labelFromContext").Call(jen.Lit("methodNameFull"), jen.Id("methodContextKey")),
					jen.Id("labelFromContext").Call(jen.Lit("scopeName"), jen.Id("scopeNameContextKey")),
				),
			)
		})),
	)
	f.Line()

	f.Func().Id("WithErrorDecode").Params(
		jen.Id("errorDecode").Id("ErrorDecoder"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("errorDecode").Op("=").Id("errorDecode"),
		)),
	)
	f.Line()

	f.Func().Id("Before").Params(
		jen.Id("before").Op("...").Id("ClientBeforeFunc"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("before").Op("=").Append(jen.Id("o").Dot("before"), jen.Id("before").Op("...")),
		)),
	)
	f.Line()

	f.Func().Id("After").Params(
		jen.Id("after").Op("...").Id("ClientAfterFunc"),
	).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("after").Op("=").Append(jen.Id("o").Dot("after"), jen.Id("after").Op("...")),
		)),
	)

	return f
}

// genMethodConstants генерирует константы имён методов.
func (g *generator) genMethodConstants(ifaceOpt *annotation.IfaceOpt) jen.Code {
	f := jen.NewFile("")

	scopeConstName := strcase.ToLowerCamel(scopeName(ifaceOpt)) + "ScopeName"
	f.Const().Id(scopeConstName).Op("=").Lit(scopeName(ifaceOpt))
	f.Line()

	for _, m := range ifaceOpt.Methods {
		shortNameConst := strcase.ToLowerCamel(scopeName(ifaceOpt)) + m.Func.Name + "ShortName"
		fullNameConst := strcase.ToLowerCamel(scopeName(ifaceOpt)) + m.Func.Name + "FullName"

		f.Const().Id(shortNameConst).Op("=").Lit(methodShortName(m))
		f.Const().Id(fullNameConst).Op("=").Lit(methodFullName(m))
	}

	return f
}

// genClientStruct генерирует структуру клиента.
func (g *generator) genClientStruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	name := clientStructName(ifaceOpt)
	return jen.Type().Id(name).Struct(
		jen.Id("target").String(),
		jen.Id("opts").Id("clientOptions"),
	)
}

// genClientConstruct генерирует конструтор клиента.
func (g *generator) genClientConstruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	name := clientStructName(ifaceOpt)
	return jen.Func().Id("New"+name).Params(
		jen.Id("target").String(),
		jen.Id("opts").Op("...").Id("ClientOption"),
	).Op("*").Id(name).BlockFunc(func(body *jen.Group) {
		body.Id(recvNameClient).Op(":=").Op("&").Id(name).Values(jen.Dict{
			jen.Id("target"): jen.Id("target"),
			jen.Id("opts"): jen.Id("clientOptions").Values(jen.Dict{
				jen.Id("client"): jen.Qual("github.com/hashicorp/go-cleanhttp", "DefaultClient").Call(),
			}),
		})
		body.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
			jen.Id("o").Call(jen.Op("&").Id(recvNameClient).Dot("opts")),
		)
		body.Return(jen.Id(recvNameClient))
	})
}

// genClientMethod генерирует метод клиента и вспомогательные структуры.
func (g *generator) genClientMethod(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	clientName := clientStructName(s)

	group := jen.NewFile("")
	group.Add(g.genReqStruct(s, m))
	group.Add(g.genReqStructSetters(m))
	if len(m.BodyParams) > 0 {
		group.Add(g.genMakeBodyRequestMethod(m))
	}
	group.Add(g.genExecuteMethod(m))

	group.Add(g.genRequestConvenienceMethod(s, m))

	// Метод клиента
	method := jen.Func().Params(
		jen.Id(recvNameReq).Op("*").Id(clientName),
	).Id(m.Func.Name)

	method = method.ParamsFunc(func(gp *jen.Group) {
		gp.Id("ctx").Qual("context", "Context")

		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}

			gp.Id(p.Var.Name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
		}
	})

	method = method.ParamsFunc(func(gp *jen.Group) {
		for _, r := range m.Results {
			gp.Add(jenutils.TypeInfoQual(r.Var.Type, g.qual))
		}
	})

	method = method.BlockFunc(func(body *jen.Group) {
		nonErrVars := make([]jen.Code, 0)
		zeroVals := make([]jen.Code, 0)

		for _, r := range m.Results {
			if r.Var.IsError {
				continue
			}

			nonErrVars = append(nonErrVars, jen.Id(r.Var.Name))
			zeroVals = append(zeroVals, jenutils.ZeroValue(r.Var.Type, g.qual))
		}

		returnVars := make([]jen.Code, len(nonErrVars))
		copy(returnVars, nonErrVars)
		returnVars = append(returnVars, jen.Err())

		reqCallArgs := make([]jen.Code, 0)
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}

			reqCallArgs = append(reqCallArgs, jen.Id(p.Var.Name))
		}

		body.Add(jen.List(returnVars...).Op(":=").Id(recvNameReq).Dot(m.Func.Name + "Request").Call(reqCallArgs...).Dot("Execute").Call(
			jen.Id("WithContext").Call(jen.Id("ctx")),
		))

		if len(nonErrVars) > 0 {
			body.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.List(append(zeroVals, jen.Err())...)),
			)
			body.Line()
			body.Return(jen.List(append(nonErrVars, jen.Nil())...))
		} else {
			body.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Err()),
			)
			body.Line()
			body.Return(jen.Nil())
		}
	})

	group.Add(method)
	return group
}

// genRequestConvenienceMethod генерирует метод XxxRequest на клиенте.
func (g *generator) genRequestConvenienceMethod(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	clientName := clientStructName(s)
	reqName := methodRequestName(m)

	fn := jen.Func().Params(
		jen.Id(recvNameReq).Op("*").Id(clientName),
	).Id(m.Func.Name + "Request")

	fn = fn.ParamsFunc(func(gp *jen.Group) {
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			gp.Id(p.Var.Name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
		}
	})

	fn = fn.Op("*").Id(reqName)

	fn = fn.BlockFunc(func(body *jen.Group) {
		body.Id("m").Op(":=").Op("&").Id(reqName).Values(jen.Dict{
			jen.Id("opts"): jen.Id(recvNameReq).Dot("opts"),
			jen.Id("c"):    jen.Id(recvNameReq),
		})
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			setterName := "Set" + strcase.ToCamel(p.Var.Name)
			body.Id("m").Dot(setterName).Call(jen.Id(p.Var.Name))
		}
		body.Return(jen.Id("m"))
	})

	return fn
}

// genReqStruct генерирует структуру параметров запроса.
func (g *generator) genReqStruct(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)
	clientName := clientStructName(s)
	return jen.Type().Id(reqName).StructFunc(func(body *jen.Group) {
		body.Id("c").Op("*").Id(clientName)
		body.Id("opts").Id("clientOptions")
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			body.Id(p.Var.Name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
		}
	})
}

// genReqStructSetters генерирует методы-сеттеры для структуры запроса.
func (g *generator) genReqStructSetters(m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)
	group := jen.NewFile("")

	for _, p := range m.Params {
		if p.Var.IsContext {
			continue
		}
		name := strcase.ToCamel(p.Var.Name)
		group.Func().Params(
			jen.Id(recvNameReq).Op("*").Id(reqName),
		).Id("Set"+name).Params(
			jen.Id("v").Add(jenutils.TypeInfoQual(p.Var.Type, g.qual)),
		).Op("*").Id(reqName).Block(
			jen.Id(recvNameReq).Dot(p.Var.Name).Op("=").Id("v"),
			jen.Return(jen.Id(recvNameReq)),
		)
	}
	return group
}

// genMakeBodyRequestMethod генерирует метод makeBodyRequest.
func (g *generator) genMakeBodyRequestMethod(m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)

	fn := jen.Func().Params(
		jen.Id(recvNameReq).Op("*").Id(reqName),
	).Id("makeBodyRequest").Params().Interface()

	if len(m.BodyParams) == 1 && m.Single.Req {
		fldName := strcase.ToLowerCamel(m.BodyParams[0].Var.Name)
		fn.Block(
			jen.Return(jen.Id(recvNameReq).Dot(fldName)),
		)
	} else {
		fn.Block(
			jen.Return(jen.Struct(annotation.MakeStructFieldsFromParams(m.BodyParams, g.qual)).ValuesFunc(func(dict *jen.Group) {
				for _, p := range m.BodyParams {
					dict.Id(strcase.ToCamel(p.Var.Name)).Op(":").Id(recvNameReq).Dot(strcase.ToLowerCamel(p.Var.Name))
				}
			})),
		)
	}

	return fn
}
