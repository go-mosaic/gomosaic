// Package client предоставляет HTTP-клиентский плагин для v2.
// Реализация приведена в соответствие с v1.
package client

import (
	"context"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

const httpStatusLastSuccessCode = 399
const recvName = "r"

// Qualifier определяет интерфейс квалификации имён.
type Qualifier interface {
	Qual(pkgPath, name string) func(s *jen.Statement)
}

// Plugin — HTTP-клиентский плагин v2.
type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "http-client" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	a, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := NewClientGenerator(f, module.Path)
	code, err := gen.Generate(a)
	if err != nil {
		return nil, err
	}
	f.Add(code)

	return map[string]gomosaic.File{"client_gen.go": f}, nil
}

func NewClientGenerator(qualifier Qualifier, modulePath string) *ClientGenerator {
	return &ClientGenerator{qualifier: qualifier, modulePath: modulePath}
}

type ClientGenerator struct {
	qualifier  Qualifier
	modulePath string
}

func (g *ClientGenerator) qual(pkgPath, name string) func(s *jen.Statement) {
	return func(s *jen.Statement) {
		g.qualifier.Qual(pkgPath, name)(s)
	}
}

func (g *ClientGenerator) Generate(services []*annotation.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")

	group.Add(g.genClientTypes())

	for _, s := range services {
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

// genClientTypes генерирует вспомогательные типы клиента.
func (g *ClientGenerator) genClientTypes() jen.Code {
	f := jen.NewFile("")
	f.Type().Id("ClientOption").Func().Params(jen.Op("*").Id("clientOptions"))
	f.Line()
	f.Type().Id("clientOptions").Struct(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("client").Op("*").Qual("net/http", "Client"),
	)
	f.Line()
	f.Func().Id("WithContext").Params(jen.Id("ctx").Qual("context", "Context")).Id("ClientOption").Block(
		jen.Return(jen.Func().Params(jen.Id("o").Op("*").Id("clientOptions")).Block(
			jen.Id("o").Dot("ctx").Op("=").Id("ctx"),
		)),
	)
	return f
}

// genClientStruct генерирует структуру клиента.
func (g *ClientGenerator) genClientStruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	name := clientStructName(ifaceOpt)
	return jen.Type().Id(name).Struct(
		jen.Id("target").String(),
		jen.Id("opts").Id("clientOptions"),
	)
}

// genClientConstruct генерирует конструктор клиента.
func (g *ClientGenerator) genClientConstruct(ifaceOpt *annotation.IfaceOpt) jen.Code {
	name := clientStructName(ifaceOpt)
	return jen.Func().Id("New"+name).Params(
		jen.Id("target").String(),
		jen.Id("opts").Op("...").Id("ClientOption"),
	).Op("*").Id(name).BlockFunc(func(body *jen.Group) {
		body.Id(recvName).Op(":=").Op("&").Id(name).Values(jen.Dict{
			jen.Id("target"): jen.Id("target"),
		})
		body.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
			jen.Id("o").Call(jen.Op("&").Id(recvName).Dot("opts")),
		)
		body.Return(jen.Id(recvName))
	})
}

// genClientMethod генерирует метод клиента.
func (g *ClientGenerator) genClientMethod(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	clientName := clientStructName(s)
	reqName := methodRequestName(m)

	group := jen.NewFile("")
	group.Add(g.genReqStruct(s, m))
	group.Add(g.genReqStructSetters(s, m))
	group.Add(g.genExecuteMethod(m))

	// Метод клиента
	method := jen.Func().Params(
		jen.Id(recvName).Op("*").Id(clientName),
	).Id(m.Func.Name)

	// Параметры
	if countNonContextParams(m.Params) > 0 {
		method = method.ParamsFunc(func(gp *jen.Group) {
			for _, p := range m.Params {
				if p.Var.IsContext {
					continue
				}
				gp.Id(p.Var.Name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
			}
		})
	}

	// Результаты
	method = method.ParamsFunc(func(gp *jen.Group) {
		for _, r := range m.Results {
			gp.Add(jenutils.TypeInfoQual(r.Var.Type, g.qual))
		}
	})

	// Тело
	method = method.BlockFunc(func(body *jen.Group) {
		body.Id("params").Op(":=").Id(reqName).Values(jen.Dict{
			jen.Id("c"): jen.Id(recvName),
		})
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			setterName := "Set" + strcase.ToCamel(p.Var.Name)
			body.Id("params").Dot(setterName).Call(jen.Id(p.Var.Name))
		}
		body.Return(jen.Id("params").Dot("Execute").Call())
	})

	group.Add(method)
	return group
}

func countNonContextParams(params []*annotation.MethodParamOpt) int {
	n := 0
	for _, p := range params {
		if !p.Var.IsContext {
			n++
		}
	}
	return n
}

// genReqStruct генерирует структуру параметров запроса.
func (g *ClientGenerator) genReqStruct(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
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
func (g *ClientGenerator) genReqStructSetters(s *annotation.IfaceOpt, m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)
	group := jen.NewFile("")

	for _, p := range m.Params {
		if p.Var.IsContext {
			continue
		}
		name := strcase.ToCamel(p.Var.Name)
		group.Func().Params(
			jen.Id(recvName).Op("*").Id(reqName),
		).Id("Set"+name).Params(
			jen.Id("v").Add(jenutils.TypeInfoQual(p.Var.Type, g.qual)),
		).Op("*").Id(reqName).Block(
			jen.Id(recvName).Dot(p.Var.Name).Op("=").Id("v"),
			jen.Return(jen.Id(recvName)),
		)
	}
	return group
}

// genExecuteMethod генерирует метод Execute для выполнения HTTP-запроса.
func (g *ClientGenerator) genExecuteMethod(m *annotation.MethodOpt) jen.Code {
	reqName := methodRequestName(m)

	fn := jen.Func().Params(
		jen.Id(recvName).Op("*").Id(reqName),
	).Id("Execute").Params(
		jen.Id("opts").Op("...").Id("ClientOption"),
	)

	// Результаты
	fn = fn.ParamsFunc(func(group *jen.Group) {
		for _, result := range m.Results {
			group.Add(jenutils.TypeInfoQual(result.Var.Type, g.qual))
		}
	})

	// Тело
	fn = fn.BlockFunc(func(body *jen.Group) {
		body.For(jen.List(jen.Id("_"), jen.Id("o")).Op(":=").Range().Id("opts")).Block(
			jen.Id("o").Call(jen.Op("&").Id(recvName).Dot("opts")),
		)
		body.List(jen.Id("ctx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(
			jen.Id(recvName).Dot("opts").Dot("ctx"),
		)
		body.Defer().Id("cancel").Call()

		// Путь
		if len(m.PathParams) > 0 {
			g.genPathWithParams(body, m)
		} else {
			body.Id("path").Op(":=").Lit(m.Path)
		}

		// Создаём запрос
		body.List(jen.Id("req"), jen.Err()).Op(":=").Qual("net/http", "NewRequestWithContext").Call(
			jen.Id("ctx"), jen.Lit(m.Method),
			jen.Id(recvName).Dot("c").Dot("target").Op("+").Id("path"), jen.Nil(),
		)
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
		)

		// Content-Type
		if len(m.BodyParams) > 0 {
			g.genJSONReqContent(body, m)
		}

		// Query, Header, Cookie
		if len(m.QueryParams) > 0 {
			g.genQueryParams(body, m)
		}
		if len(m.HeaderParams) > 0 {
			g.genHeaderParams(body, m)
		}
		if len(m.CookieParams) > 0 {
			g.genCookieParams(body, m)
		}

		// Отправка
		body.List(jen.Id("resp"), jen.Err()).Op(":=").Id(recvName).Dot("opts").Dot("client").Dot("Do").Call(jen.Id("req"))
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
		)
		body.Defer().Id("resp").Dot("Body").Dot("Close").Call()
		body.Defer().Id("cancel").Call()

		// Статус
		body.If(jen.Id("resp").Dot("StatusCode").Op(">").Lit(httpStatusLastSuccessCode)).Block(
			jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual,
				jen.Qual("fmt", "Errorf").Call(jen.Lit("http error %d"), jen.Id("resp").Dot("StatusCode")),
			)...),
		)

		// Чтение ответа
		if len(m.BodyResults) > 0 {
			g.genReadResponse(body, m)
		} else {
			body.Return(jen.Nil())
		}
	})

	return fn
}

func (g *ClientGenerator) genPathWithParams(body *jen.Group, m *annotation.MethodOpt) {
	body.Do(func(s *jen.Statement) {
		var paramsCall []jen.Code
		paramsCall = append(paramsCall, jen.Lit(sprintfPath(m)))
		for _, p := range m.PathParams {
			paramsCall = append(paramsCall, jen.Id(recvName).Dot(p.PathParamName))
		}
		s.Id("path").Op(":=").Qual("fmt", "Sprintf").Call(paramsCall...)
	})
}

func (g *ClientGenerator) genJSONReqContent(body *jen.Group, m *annotation.MethodOpt) {
	body.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json"))

	makeBodyCall := g.genMakeRequestBodyMethod(m)
	body.List(jen.Id("reqBody"), jen.Err()).Op(":=").Do(func(s *jen.Statement) {
		s.Qual("encoding/json", "Marshal").Call(makeBodyCall)
	})
	body.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
	)
	body.Id("req").Dot("Body").Op("=").Qual("io", "NopCloser").Call(
		jen.Qual("bytes", "NewReader").Call(jen.Id("reqBody")),
	)
}

func (g *ClientGenerator) genMakeRequestBodyMethod(m *annotation.MethodOpt) jen.Code {
	if len(m.BodyParams) == 1 && m.Single.Req {
		fldName := strcase.ToLowerCamel(m.BodyParams[0].Var.Name)
		return jen.Id(recvName).Dot(fldName)
	}
	return jen.Struct(annotation.MakeStructFieldsFromParams(m.BodyParams, g.qual)).ValuesFunc(func(dict *jen.Group) {
		for _, p := range m.BodyParams {
			dict.Id(strcase.ToCamel(p.Var.Name)).Op(":").Id(recvName).Dot(strcase.ToLowerCamel(p.Var.Name))
		}
	})
}

func (g *ClientGenerator) genQueryParams(body *jen.Group, m *annotation.MethodOpt) {
	body.Id("q").Op(":=").Id("req").Dot("URL").Dot("Query").Call()
	for _, p := range m.QueryParams {
		body.Id("q").Dot("Set").Call(jen.Lit(p.Name), jen.Qual("fmt", "Sprintf").Call(
			jen.Lit("%v"), jen.Id(recvName).Dot(strcase.ToLowerCamel(p.Var.Name)),
		))
	}
	body.Id("req").Dot("URL").Dot("RawQuery").Op("=").Id("q").Dot("Encode").Call()
}

func (g *ClientGenerator) genHeaderParams(body *jen.Group, m *annotation.MethodOpt) {
	for _, p := range m.HeaderParams {
		body.Id("req").Dot("Header").Dot("Set").Call(
			jen.Lit(p.Name),
			jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(recvName).Dot(strcase.ToLowerCamel(p.Var.Name))),
		)
	}
}

func (g *ClientGenerator) genCookieParams(body *jen.Group, m *annotation.MethodOpt) {
	for _, p := range m.CookieParams {
		body.Id("req").Dot("AddCookie").Call(
			jen.Qual("net/http", "Cookie").Values(jen.Dict{
				jen.Id("Name"):  jen.Lit(p.Name),
				jen.Id("Value"): jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(recvName).Dot(strcase.ToLowerCamel(p.Var.Name))),
			}),
		)
	}
}

func (g *ClientGenerator) genReadResponse(body *jen.Group, m *annotation.MethodOpt) {
	if len(m.BodyResults) == 1 && m.Single.Resp {
		body.Var().Id("respBody").Add(jenutils.TypeInfoQual(m.BodyResults[0].Var.Type, g.qual))
	} else {
		body.Var().Id("respBody").Struct(annotation.MakeStructFieldsFromResults(m.BodyResults, g.qual))
	}

	body.If(jen.Err().Op("=").Qual("encoding/json", "NewDecoder").Call(
		jen.Id("resp").Dot("Body"),
	).Dot("Decode").Call(jen.Op("&").Id("respBody")), jen.Err().Op("!=").Nil()).Block(
		jen.Return(annotation.MakeEmptyResults(m.BodyResults, g.qual, jen.Err())...),
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

func sprintfPath(methodOpt *annotation.MethodOpt) string {
	parts := strings.Split(methodOpt.Path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "%s"
		}
	}
	return strings.Join(parts, "/")
}

var _ gomosaic.Generator = (*Plugin)(nil)
