package binder

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Plugin генерирует хелперы связывания параметров.
type Plugin struct {
	strategy Strategy
}

// NewPlugin создаёт плагин для указанной стратегии.
func NewPlugin(strategy Strategy) *Plugin {
	return &Plugin{strategy: strategy}
}

// Name реализует gomosaic.Generator.
func (p *Plugin) Name() string {
	return p.strategy.Name()
}

// Generate реализует gomosaic.Generator.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	structs, err := Load(module, types)
	if err != nil {
		return nil, err
	}

	if len(structs) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &generator{
		module:            module,
		strategy:          p.strategy,
		qual:              f.Qual,
		transformRegistry: gomosaic.DefaultTransformRegistry(),
	}

	code, err := gen.Generate(structs)
	if err != nil {
		return nil, err
	}
	f.Add(code)

	filename := fmt.Sprintf(
		"http_bind_%s_gen.go",
		strings.ReplaceAll(strings.TrimPrefix(p.strategy.Name(), "http-bind-"), "-", "_"),
	)

	return map[string]gomosaic.File{filename: f}, nil
}

var _ gomosaic.Generator = (*Plugin)(nil)
