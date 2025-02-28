package server

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
)

var _ Strategy = &StrategyChi{}

type StrategyChi struct{}

func (s *StrategyChi) ID() string {
	return "chi"
}

func (s *StrategyChi) QueryParams() (typ jen.Code) {
	return jen.Id("q").Op(":=").Id(s.ReqArgName()).Dot("URL").Dot("Query").Call()
}

func (s *StrategyChi) QueryParamValue(name string) jen.Code {
	return jen.Id("q").Dot("Get").Call(jen.Lit(name))
}

func (s *StrategyChi) PathParamValue(name string) jen.Code {
	return jen.Qual(service.ChiPkg, "URLParam").Call(jen.Id(s.ReqArgName()), jen.Lit(name))
}

func (s *StrategyChi) HeaderParamValue(name string) jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Header").Dot("Get").Call(jen.Lit(name))
}

func (s *StrategyChi) BodyPathParam() (typ jen.Code) {
	return jen.Id(s.ReqArgName()).Dot("Body")
}

func (s *StrategyChi) FormParam(formName string) jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Form").Dot("Get").Call(jen.Lit(formName))
}

func (s *StrategyChi) FormParams(errorStmts ...jen.Code) jen.Code {
	return jen.If(jen.Err().Op(":=").Id(s.ReqArgName()).Dot("ParseForm").Call(), jen.Err().Op("!=").Nil()).Block(errorStmts...)
}

func (s *StrategyChi) MultipartFormParam(formName string) jen.Code {
	return jen.Id(s.ReqArgName()).Dot("FormValue").Call(jen.Lit(formName))
}

func (s *StrategyChi) MultipartFormParams(multipartMaxMemory int64, errorStmts ...jen.Code) jen.Code {
	return jen.If(jen.Err().Op(":=").Id(s.ReqArgName()).Dot("ParseMultipartForm").Call(jen.Lit(multipartMaxMemory)), jen.Err().Op("!=").Nil()).Block(errorStmts...)
}

func (*StrategyChi) RespType() jen.Code {
	return jen.Qual(service.HTTPPkg, "ResponseWriter")
}

func (*StrategyChi) LibType() jen.Code {
	return jen.Qual(service.ChiPkg, "Router")
}

func (s *StrategyChi) HandlerFuncParams() (in, out []jen.Code) {
	return []jen.Code{
			jen.Id(s.ReqArgName()).Qual(service.CTXPkg, "Context"),
		}, []jen.Code{
			jen.Id("_").Error(),
		}
}

func (s *StrategyChi) HandlerFunc(method string, pattern string, middlewares jen.Code, handlerFunc func(g *jen.Group)) jen.Code {
	return jen.Id(s.LibArgName()).Dot("With").Call(middlewares).Dot("Method").Call(
		jen.Lit(method),
		jen.Lit(pattern),

		jen.Qual(service.HTTPPkg, "HandlerFunc").Call(
			jen.Func().Params(
				jen.Id(s.RespArgName()).Qual(service.HTTPPkg, "ResponseWriter"),
				jen.Id(s.ReqArgName()).Op("*").Qual(service.HTTPPkg, "Request"),
			).BlockFunc(func(g *jen.Group) {
				handlerFunc(g)
				g.Return()
			}),
		),
	)
}

func (*StrategyChi) MiddlewareType() jen.Code {
	return jen.Func().Params(jen.Qual(service.HTTPPkg, "Handler")).Qual(service.HTTPPkg, "Handler")
}

func (s *StrategyChi) SetHeader(k jen.Code, v jen.Code) (typ jen.Code) {
	return jen.Id(s.RespArgName()).Dot("Header").Call().Dot("Set").Call(k, v)
}

func (s *StrategyChi) WriteBody(data, statusCode jen.Code) jen.Code {
	group := jen.NewFile("")
	group.Id(s.RespArgName()).Dot("WriteHeader").Call(statusCode)
	if data != nil {
		group.Id(s.RespArgName()).Dot("Write").Call(data)
	}
	return group
}

func (*StrategyChi) UsePathParams() bool {
	return true
}

func (s *StrategyChi) Context() jen.Code {
	return jen.Id(s.ReqArgName()).Dot("Context").Call()
}

func (*StrategyChi) RespArgName() string {
	return "w"
}

func (*StrategyChi) ReqArgName() string {
	return "r"
}

func (*StrategyChi) LibArgName() string {
	return "r"
}
