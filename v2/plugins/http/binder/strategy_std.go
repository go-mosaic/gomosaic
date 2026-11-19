package binder

import "github.com/dave/jennifer/jen"

// StdStrategy реализует Strategy для net/http.
type StdStrategy struct{}

// Name реализует Strategy.
func (s *StdStrategy) Name() string { return "http-bind-std" }

// PathParamExtract реализует Strategy.
func (s *StdStrategy) PathParamExtract(reqVar, paramName string) *jen.Statement {
	return jen.Id(reqVar).Dot("PathValue").Call(jen.Lit(paramName))
}
