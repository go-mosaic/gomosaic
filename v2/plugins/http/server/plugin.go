package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// Plugin — HTTP-серверный плагин v2.
// Конфигурируется через Strategy (Chi, Echo, и т.д.).
type Plugin struct {
	strategy Strategy
}

// NewPlugin создает новый серверный плагин.
func NewPlugin(strategy Strategy) *Plugin {
	return &Plugin{strategy: strategy}
}

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return p.strategy.Name() }

// Generate генерирует код HTTP-сервера.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	gen := &generator{
		module:            module,
		strategy:          p.strategy,
		qual:              f.Qual,
		transformRegistry: gomosaic.DefaultTransformRegistry(),
	}

	code, err := gen.Generate(annotations)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		f.Add(code)
	}

	filename := fmt.Sprintf("server_%s_gen.go", strings.TrimPrefix(p.strategy.Name(), "http-server-"))
	return map[string]gomosaic.File{filename: f}, errs
}

// Ensure Plugin implements gomosaic.Generator.
var _ gomosaic.Generator = (*Plugin)(nil)
