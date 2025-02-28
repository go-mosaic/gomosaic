package http

import "github.com/go-mosaic/gomosaic/pkg/gomosaic"

func init() {
	gomosaic.RegisterPlugin(new(PluginServerChi))
	gomosaic.RegisterPlugin(new(PluginServerEcho))
	gomosaic.RegisterPlugin(new(PluginClient))
	gomosaic.RegisterPlugin(new(PluginClientTesting))
}
