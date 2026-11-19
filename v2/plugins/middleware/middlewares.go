package middleware

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// NewLogMiddleware создаёт плагин логирующего middleware.
func NewLogMiddleware() *Plugin {
	return NewPlugin(Config{
		Name:           "log-middleware",
		MiddlewareName: "Log",
		Annotation:     "log",
		Fields: []jen.Code{
			jen.Id("logger"),
			jen.Qual(gomosaic.SpanPkg, "Logger"),
		},
		BeforeFn: func(group *jen.Group, m *gomosaic.MethodInfo) {
			group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartLogSpan").Call(
				jen.Id("m").Dot("logger"),
				jen.Lit(m.ShortName),
			)
		},
		AfterFn: func(group *jen.Group, m *gomosaic.MethodInfo) {
			errName := "err"
			if len(m.Results) > 0 {
				errName = m.Results[len(m.Results)-1].Name
			}

			group.If(jen.Id(errName).Op("!=").Nil()).Block(
				jen.Id("span").Dot("FinishWithError").Call(jen.Id("ctx"), jen.Id(errName)),
			).Else().Block(
				jen.Id("span").Dot("Finish").Call(jen.Id("ctx")),
			)
		},
		OutputFile: "log_middleware_gen.go",
	})
}

// NewMetricMiddleware создаёт плагин middleware метрик.
func NewMetricMiddleware() *Plugin {
	return NewPlugin(Config{
		Name:           "metric-middleware",
		MiddlewareName: "Metric",
		Annotation:     "metric",
		Fields: []jen.Code{
			jen.Id("collector"),
			jen.Qual(gomosaic.SpanPkg, "MetricsCollector"),
		},
		BeforeFn: func(group *jen.Group, m *gomosaic.MethodInfo) {
			group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartMetricSpan").Call(
				jen.Id("m").Dot("collector"),
				jen.Lit(m.ShortName),
			)
		},
		AfterFn: func(group *jen.Group, m *gomosaic.MethodInfo) {
			if errVar, ok := gomosaic.HasError(m.Results); ok {
				group.If(jen.Id(errVar.Name).Op("!=").Nil()).Block(
					jen.Id("span").Dot("FinishWithError").Call(jen.Id("ctx"), jen.Id(errVar.Name)),
				).Else().Block(
					jen.Id("span").Dot("Finish").Call(jen.Id("ctx")),
				)

				return
			}

			group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
		},
		OutputFile: "metric_middleware_gen.go",
	})
}
