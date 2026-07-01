package binder

import "github.com/dave/jennifer/jen"

type ChiStrategy struct{}

func (s *ChiStrategy) Name() string { return "http-bind-chi" }

func (s *ChiStrategy) PathParamExtract(reqVar, paramName string) *jen.Statement {
	return jen.Qual("github.com/go-chi/chi/v5", "URLParam").Call(jen.Id(reqVar), jen.Lit(paramName))
}
