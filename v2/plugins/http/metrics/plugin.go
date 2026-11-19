package metrics

import (
	"context"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Plugin генератор HTTP-middleware метрик, конфигурируемый через Strategy.
type Plugin struct {
	strategy Strategy
}

// NewPlugin создаёт плагин для указанной стратегии роутера.
func NewPlugin(strategy Strategy) *Plugin {
	return &Plugin{strategy: strategy}
}

// Name реализует gomosaic.Generator.
func (p *Plugin) Name() string { return p.strategy.Name() }

// Generate реализует gomosaic.Generator.
func (p *Plugin) Generate(
	ctx context.Context,
	module *gomosaic.ModuleInfo,
	_ []*gomosaic.NameTypeInfo,
) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &generator{f: f, strategy: p.strategy}
	f.Add(gen.Generate())

	return map[string]gomosaic.File{p.strategy.FileName(): f}, nil
}

var _ gomosaic.Generator = (*Plugin)(nil)
