// Command gomosaic реализует CLI-генератор кода GoMosaic.
package main

import (
	"runtime/debug"

	"github.com/go-mosaic/gomosaic/v2/pkg/cmd"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic/defaults"
)

// Version хранит версию сборки; по умолчанию "dev".
var Version = "dev"

func main() {
	resolveVersion()

	builder := gomosaic.NewBuilder(defaults.WithPlugins())

	cmd.Run(Version, builder)
}

func resolveVersion() {
	if Version != "dev" {
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if v := info.Main.Version; v != "" && v != "(devel)" {
		Version = v
	}
}
