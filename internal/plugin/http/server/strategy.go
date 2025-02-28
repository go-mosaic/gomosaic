package server

import "github.com/dave/jennifer/jen"

type Strategy interface {
	ID() string
	ReqArgName() string
	RespType() jen.Code
	RespArgName() string
	LibType() jen.Code
	LibArgName() string
	Context() jen.Code
	QueryParams() jen.Code
	QueryParamValue(name string) jen.Code
	PathParamValue(name string) jen.Code
	HeaderParamValue(name string) jen.Code
	BodyPathParam() (typ jen.Code)
	FormParam(formName string) jen.Code
	MultipartFormParam(formName string) jen.Code
	FormParams(errorStmts ...jen.Code) jen.Code
	MultipartFormParams(multipartMaxMemory int64, errorStmts ...jen.Code) jen.Code
	MiddlewareType() jen.Code
	HandlerFunc(method, pattern string, middlewares jen.Code, handlerFunc func(g *jen.Group)) jen.Code
	SetHeader(k, v jen.Code) jen.Code
	UsePathParams() bool
	WriteBody(data, statusCode jen.Code) jen.Code
}
