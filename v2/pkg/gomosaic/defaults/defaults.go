// Package defaults предоставляет готовые конфигурации для gomosaic,
//
// Использование:
//
//	// Создать builder с плагинами по умолчанию:
//	builder := gomosaic.NewBuilder(defaults.WithPlugins())
package defaults

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/envconfig"
	grpcserver "github.com/go-mosaic/gomosaic/v2/plugins/grpc/server"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/binder"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/client"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/server"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/testclient"
	"github.com/go-mosaic/gomosaic/v2/plugins/middleware"
	"github.com/go-mosaic/gomosaic/v2/plugins/openapi"
	"github.com/go-mosaic/gomosaic/v2/plugins/validation"
)

// PluginRegistry возвращает реестр со всеми встроенными плагинами.
func PluginRegistry() *gomosaic.PluginRegistry {
	reg := gomosaic.NewPluginRegistry()

	reg.MustRegister(server.NewPlugin(&server.ChiStrategy{}))
	reg.MustRegister(server.NewPlugin(&server.EchoStrategy{}))
	reg.MustRegister(client.NewPlugin())
	reg.MustRegister(testclient.NewPlugin())
	reg.MustRegister(grpcserver.NewPlugin())
	reg.MustRegister(openapi.NewPlugin())
	reg.MustRegister(validation.NewPlugin())
	reg.MustRegister(envconfig.NewPlugin())
	reg.MustRegister(binder.NewPlugin(&binder.StdStrategy{}))
	reg.MustRegister(binder.NewPlugin(&binder.ChiStrategy{}))
	reg.MustRegister(newLogMiddleware())
	reg.MustRegister(newMetricMiddleware())

	return reg
}

func WithPlugins() gomosaic.BuilderOption {
	return gomosaic.WithPluginRegistry(PluginRegistry())
}

func newLogMiddleware() *middleware.Plugin {
	return middleware.NewPlugin(middleware.Config{
		Name:           "log-middleware",
		MiddlewareName: "Log",
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

func newMetricMiddleware() *middleware.Plugin {
	return middleware.NewPlugin(middleware.Config{
		Name:           "metric-middleware",
		MiddlewareName: "Metric",
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
		AfterFn: func(group *jen.Group, _ *gomosaic.MethodInfo) {
			group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
		},
		OutputFile: "metric_middleware_gen.go",
	})
}
