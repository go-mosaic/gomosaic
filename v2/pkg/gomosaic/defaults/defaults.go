// Package defaults предоставляет готовые конфигурации для gomosaic,
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
	"github.com/go-mosaic/gomosaic/v2/plugins/http/server"
	"github.com/go-mosaic/gomosaic/v2/plugins/openapi"
	"github.com/go-mosaic/gomosaic/v2/plugins/validation"
)

// PluginRegistry возвращает реестр со всеми встроенными плагинами.
func PluginRegistry() *gomosaic.PluginRegistry {
	reg := gomosaic.NewPluginRegistry()

	reg.MustRegister(server.NewPlugin(&server.ChiStrategy{}))
	reg.MustRegister(server.NewPlugin(&server.EchoStrategy{}))
	reg.MustRegister(grpcserver.NewPlugin())
	reg.MustRegister(openapi.NewPlugin())
	reg.MustRegister(validation.NewPlugin())
	reg.MustRegister(envconfig.NewPlugin())
	reg.MustRegister(binder.NewPlugin(&binder.StdStrategy{}))
	reg.MustRegister(binder.NewPlugin(&binder.ChiStrategy{}))

	return reg
}

func WithPlugins() gomosaic.BuilderOption {
	return gomosaic.WithPluginRegistry(PluginRegistry())
}
