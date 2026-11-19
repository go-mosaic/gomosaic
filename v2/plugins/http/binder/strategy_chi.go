package binder

import "github.com/dave/jennifer/jen"

// ChiStrategy реализует Strategy для роутера chi.
type ChiStrategy struct{}

// Name реализует Strategy.
func (s *ChiStrategy) Name() string { return "http-bind-chi" }

// PathParamExtract реализует Strategy.
func (s *ChiStrategy) PathParamExtract(reqVar, paramName string) *jen.Statement {
	return jen.Qual("github.com/go-chi/chi/v5", "URLParam").Call(jen.Id(reqVar), jen.Lit(paramName))
}
