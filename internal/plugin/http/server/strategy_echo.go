package server

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
)

var _ Strategy = &StrategyEcho{}

type StrategyEcho struct{}

func (s *StrategyEcho) ID() string {
	return "echo"
}

func (s *StrategyEcho) QueryParams() jen.Code {
	return jen.Id("q").Op(":=").Id(s.ReqArgName()).Dot("QueryParams").Call()
}

func (s *StrategyEcho) QueryParamValue(name string) jen.Code {
	return jen.Id("q").Dot("Get").Call(jen.Lit(name))
}

func (s *StrategyEcho) PathParamValue(name string) jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Param").Call(jen.Lit(name))
}

func (s *StrategyEcho) HeaderParamValue(name string) jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Request").Call().Dot("Header").Dot("Get").Call(jen.Lit(name))
}

func (s *StrategyEcho) BodyPathParam() (typ jen.Code) {
	return jen.Id(s.ReqArgName()).Dot("Request").Call().Dot("Body")
}

func (*StrategyEcho) FormParam(formName string) jen.Code {
	return jen.Id("f").Dot("Get").Call(jen.Lit(formName))
}

func (s *StrategyEcho) MultipartFormParam(formName string) jen.Code {
	return jen.Id("f").Dot("Get").Call(jen.Lit(formName))
}

func (s *StrategyEcho) FormParams(errorStmts ...jen.Code) jen.Code {
	g := jen.NewFile("")
	g.List(jen.Id("f"), jen.Err()).Op(":=").Id(s.ReqArgName()).Dot("FormParams").Call()
	g.If(jen.Err().Op("!=").Nil()).Block(errorStmts...)

	return g
}

func (s *StrategyEcho) MultipartFormParams(multipartMaxMemory int64, errorStmts ...jen.Code) jen.Code {
	return jen.List(jen.Id("f"), jen.Err()).Op(":=").Id(s.ReqArgName()).Dot("FormParams").Call()
}

func (s *StrategyEcho) Context() jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Request").Call().Dot("Context").Call()
}

func (*StrategyEcho) RespType() jen.Code {
	return jen.Qual(service.EchoPkg, "Context")
}

func (*StrategyEcho) MiddlewareType() jen.Code {
	return jen.Qual(service.EchoPkg, "MiddlewareFunc")
}

func (*StrategyEcho) LibType() jen.Code {
	return jen.Op("*").Qual(service.EchoPkg, "Echo")
}

func (s *StrategyEcho) HandlerFuncParams() (in, out []jen.Code) {
	return []jen.Code{
			jen.Id(s.ReqArgName()).Qual(service.EchoPkg, "Context"),
		}, []jen.Code{
			jen.Id("_").Error(),
		}
}

func (s *StrategyEcho) HandlerFunc(method string, pattern string, middlewares jen.Code, handlerFunc func(g *jen.Group)) jen.Code {
	return jen.Id(s.LibArgName()).Dot("Add").Call(
		jen.Lit(method),
		jen.Lit(pattern),
		jen.Func().Params(jen.Id(s.ReqArgName()).Qual(service.EchoPkg, "Context")).Params(jen.Id("_").Error()).BlockFunc(func(g *jen.Group) {
			handlerFunc(g)
		}),
		middlewares,
	)
}

func (*StrategyEcho) PathParamWrap(paramName string) string {
	return "{" + paramName + "}"
}

func (s *StrategyEcho) SetHeader(k jen.Code, v jen.Code) (typ jen.Code) {
	return jen.Id(s.RespArgName()).Dot("Response").Call().Dot("Header").Call().Dot("Add").Call(k, v)
}

func (s *StrategyEcho) WriteBody(data, statusCode jen.Code) jen.Code {
	group := jen.NewFile("")
	group.Id(s.RespArgName()).Dot("Response").Call().Dot("WriteHeader").Call(statusCode)
	if data != nil {
		group.Id(s.RespArgName()).Dot("Response").Call().Dot("Write").Call(data)
	}
	return group
}

func (*StrategyEcho) RespArgName() string {
	return "ctx"
}

func (*StrategyEcho) ReqArgName() string {
	return "ctx"
}

func (*StrategyEcho) LibArgName() string {
	return "e"
}

func (*StrategyEcho) UsePathParams() bool {
	return true
}
