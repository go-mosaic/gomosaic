package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

const (
	netHTTPPkg       = "net/http"
	chiPkg           = "github.com/go-chi/chi/v5"
	chiMiddlewarePkg = "github.com/go-chi/chi/v5/middleware"
)

// ChiStrategy генерирует middleware для роутера go-chi/chi.
type ChiStrategy struct{}

// Name реализует Strategy.
func (s *ChiStrategy) Name() string { return "http-metrics-chi" }

// FileName реализует Strategy.
func (s *ChiStrategy) FileName() string { return "metrics_chi_gen.go" }

// ConstructorName реализует Strategy.
func (s *ChiStrategy) ConstructorName() string { return "NewChiMetricsMiddleware" }

// StructName реализует Strategy.
func (s *ChiStrategy) StructName() string { return "chiMetrics" }

// MiddlewareType реализует Strategy.
func (s *ChiStrategy) MiddlewareType(_ *gomosaic.GoFile) jen.Code {
	return jen.Func().Params(jen.Qual(netHTTPPkg, "Handler")).Qual(netHTTPPkg, "Handler")
}

// ExtraTypes реализует Strategy.
func (s *ChiStrategy) ExtraTypes(_ *gomosaic.GoFile) jen.Code { return nil }

// GenerateReturn реализует Strategy.
func (s *ChiStrategy) GenerateReturn(_ *gomosaic.GoFile) jen.Code {
	return jen.Return(
		jen.Func().Params(
			jen.Id("next").Qual(netHTTPPkg, "Handler"),
		).Qual(netHTTPPkg, "Handler").Block(
			jen.Return(jen.Qual(netHTTPPkg, "HandlerFunc").Call(
				jen.Func().Params(
					jen.Id("w").Qual(netHTTPPkg, "ResponseWriter"),
					jen.Id("r").Op("*").Qual(netHTTPPkg, "Request"),
				).Block(
					jen.Id("start").Op(":=").Qual(timePkg, "Now").Call(),
					jen.Id("ww").Op(":=").Qual(chiMiddlewarePkg, "NewWrapResponseWriter").Call(
						jen.Id("w"), jen.Id("r").Dot("ProtoMajor"),
					),
					jen.Id("next").Dot("ServeHTTP").Call(jen.Id("ww"), jen.Id("r")),
					jen.Id("status").Op(":=").Id("ww").Dot("Status").Call(),
					jen.If(jen.Id("status").Op("==").Lit(0)).Block(
						jen.Id("status").Op("=").Qual(netHTTPPkg, "StatusOK"),
					),
					jen.Id("path").Op(":=").Qual(chiPkg, "RouteContext").Call(
						jen.Id("r").Dot("Context").Call(),
					).Dot("RoutePattern").Call(),
					jen.If(jen.Id("path").Op("==").Lit("")).Block(
						jen.Id("path").Op("=").Id("r").Dot("URL").Dot("Path"),
					),
					jen.Id("m").Dot("observe").Call(
						jen.Id("r").Dot("Method"),
						jen.Id("path"),
						jen.Id("status"),
						jen.Qual(timePkg, "Since").Call(jen.Id("start")),
					),
				),
			)),
		),
	)
}

var _ Strategy = (*ChiStrategy)(nil)
