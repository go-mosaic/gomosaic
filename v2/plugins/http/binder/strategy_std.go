package binder

import "github.com/dave/jennifer/jen"

type StdStrategy struct{}

func (s *StdStrategy) Name() string { return "http-bind-std" }

func (s *StdStrategy) PathParamExtract(reqVar, paramName string) *jen.Statement {
	return jen.Id(reqVar).Dot("PathValue").Call(jen.Lit(paramName))
}
