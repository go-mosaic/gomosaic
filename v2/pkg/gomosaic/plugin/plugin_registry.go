package plugin

import (
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// NamedGenerator расширяет gomosaic.Generator методами для метаинформации о плагине.
type NamedGenerator interface {
	gomosaic.Generator

	Description() string
	DependsOn() []string
}
