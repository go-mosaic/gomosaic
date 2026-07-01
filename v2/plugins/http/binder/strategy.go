package binder

import "github.com/dave/jennifer/jen"

// Strategy определяет стратегию извлечения параметров для конкретного HTTP-роутера.
type Strategy interface {
	Name() string
	PathParamExtract(reqVar, paramName string) *jen.Statement
}
