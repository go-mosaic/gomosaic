package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

const echoPkg = "github.com/labstack/echo/v4"

// EchoStrategy генерирует middleware для фреймворка labstack/echo.
type EchoStrategy struct{}

// Name реализует Strategy.
func (s *EchoStrategy) Name() string { return "http-metrics-echo" }

// FileName реализует Strategy.
func (s *EchoStrategy) FileName() string { return "metrics_echo_gen.go" }

// ConstructorName реализует Strategy.
func (s *EchoStrategy) ConstructorName() string { return "NewEchoMetricsMiddleware" }

// StructName реализует Strategy.
func (s *EchoStrategy) StructName() string { return "echoMetrics" }

// MiddlewareType реализует Strategy.
func (s *EchoStrategy) MiddlewareType(_ *gomosaic.GoFile) jen.Code {
	return jen.Qual(echoPkg, "MiddlewareFunc")
}

// ExtraTypes реализует Strategy.
func (s *EchoStrategy) ExtraTypes(_ *gomosaic.GoFile) jen.Code { return nil }

// GenerateReturn реализует Strategy.
func (s *EchoStrategy) GenerateReturn(_ *gomosaic.GoFile) jen.Code {
	return jen.Return(
		jen.Func().Params(
			jen.Id("next").Qual(echoPkg, "HandlerFunc"),
		).Qual(echoPkg, "HandlerFunc").Block(
			jen.Return(jen.Func().Params(
				jen.Id("c").Qual(echoPkg, "Context"),
			).Error().Block(
				jen.Id("start").Op(":=").Qual(timePkg, "Now").Call(),
				jen.Id("err").Op(":=").Id("next").Call(jen.Id("c")),
				jen.Id("path").Op(":=").Id("c").Dot("Path").Call(),
				jen.If(jen.Id("path").Op("==").Lit("")).Block(
					jen.Id("path").Op("=").Id("c").Dot("Request").Call().Dot("URL").Dot("Path"),
				),
				jen.Id("m").Dot("observe").Call(
					jen.Id("c").Dot("Request").Call().Dot("Method"),
					jen.Id("path"),
					jen.Id("c").Dot("Response").Call().Dot("Status"),
					jen.Qual(timePkg, "Since").Call(jen.Id("start")),
				),
				jen.Return(jen.Id("err")),
			)),
		),
	)
}

var _ Strategy = (*EchoStrategy)(nil)
