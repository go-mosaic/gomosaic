package log

import "github.com/go-mosaic/gomosaic/pkg/gomosaic"

func init() {
	gomosaic.RegisterPlugin(new(Plugin))
}
