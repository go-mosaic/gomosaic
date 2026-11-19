// Package defaults предоставляет готовые конфигурации для gomosaic.
//
// Использование:
//
//	// Создать builder с плагинами по умолчанию:
//	builder := gomosaic.NewBuilder(defaults.WithPlugins())
package defaults

import (
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/envconfig"
	grpcserver "github.com/go-mosaic/gomosaic/v2/plugins/grpc/server"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/binder"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/client"
	httpmetrics "github.com/go-mosaic/gomosaic/v2/plugins/http/metrics"
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
	reg.MustRegister(middleware.NewLogMiddleware())
	reg.MustRegister(middleware.NewMetricMiddleware())
	reg.MustRegister(httpmetrics.NewPlugin(&httpmetrics.ChiStrategy{}))
	reg.MustRegister(httpmetrics.NewPlugin(&httpmetrics.EchoStrategy{}))
	reg.MustRegister(httpmetrics.NewPlugin(&httpmetrics.FiberStrategy{}))
	reg.MustRegister(httpmetrics.NewPlugin(&httpmetrics.MuxStrategy{}))

	return reg
}

// WithPlugins возвращает опцию Builder со всеми встроенными плагинами.
func WithPlugins() gomosaic.BuilderOption {
	return gomosaic.WithPluginRegistry(PluginRegistry())
}
