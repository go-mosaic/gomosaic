// Package client предоставляет HTTP-клиентский плагин для v2.
// Реализация приведена в соответствие с v1.
package client

import (
	"context"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// Plugin — HTTP-клиентский плагин v2.
type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "http-client" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	a, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := &generator{qual: f.Qual, modulePath: module.Path}
	code, err := gen.Generate(a)
	if err != nil {
		return nil, err
	}
	f.Add(code)

	return map[string]gomosaic.File{"client_gen.go": f}, nil
}

var _ gomosaic.Generator = (*Plugin)(nil)
