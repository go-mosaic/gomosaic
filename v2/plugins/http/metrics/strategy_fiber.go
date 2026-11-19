package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

const fiberPkg = "github.com/gofiber/fiber/v2"

// FiberStrategy генерирует middleware для фреймворка gofiber/fiber.
type FiberStrategy struct{}

// Name реализует Strategy.
func (s *FiberStrategy) Name() string { return "http-metrics-fiber" }

// FileName реализует Strategy.
func (s *FiberStrategy) FileName() string { return "metrics_fiber_gen.go" }

// ConstructorName реализует Strategy.
func (s *FiberStrategy) ConstructorName() string { return "NewFiberMetricsMiddleware" }

// StructName реализует Strategy.
func (s *FiberStrategy) StructName() string { return "fiberMetrics" }

// MiddlewareType реализует Strategy.
func (s *FiberStrategy) MiddlewareType(_ *gomosaic.GoFile) jen.Code {
	return jen.Func().Params(jen.Op("*").Qual(fiberPkg, "Ctx")).Error()
}

// ExtraTypes реализует Strategy.
func (s *FiberStrategy) ExtraTypes(_ *gomosaic.GoFile) jen.Code { return nil }

// GenerateReturn реализует Strategy.
func (s *FiberStrategy) GenerateReturn(_ *gomosaic.GoFile) jen.Code {
	return jen.Return(
		jen.Func().Params(
			jen.Id("c").Op("*").Qual(fiberPkg, "Ctx"),
		).Error().Block(
			jen.Id("start").Op(":=").Qual(timePkg, "Now").Call(),
			jen.Id("err").Op(":=").Id("c").Dot("Next").Call(),
			jen.Id("path").Op(":=").Id("c").Dot("Path").Call(),
			jen.If(
				jen.Id("route").Op(":=").Id("c").Dot("Route").Call(),
				jen.Id("route").Op("!=").Nil(),
			).Block(
				jen.Id("path").Op("=").Id("route").Dot("Path"),
			),
			jen.Id("m").Dot("observe").Call(
				jen.Id("c").Dot("Method").Call(),
				jen.Id("path"),
				jen.Id("c").Dot("Response").Call().Dot("StatusCode").Call(),
				jen.Qual(timePkg, "Since").Call(jen.Id("start")),
			),
			jen.Return(jen.Id("err")),
		),
	)
}

var _ Strategy = (*FiberStrategy)(nil)
