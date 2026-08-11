package main

import (
	"github.com/go-mosaic/gomosaic/v2/pkg/cmd"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic/defaults"
)

var Version = "dev"

func main() {
	builder := gomosaic.NewBuilder(defaults.WithPlugins())

	cmd.Run(Version, builder)
}
