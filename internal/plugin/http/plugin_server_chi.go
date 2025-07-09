package http

import (
	"context"
	_ "embed"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/annotation"
	"github.com/go-mosaic/gomosaic/internal/plugin/http/server"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type PluginServerChi struct{}

func (p *PluginServerChi) Name() string { return "http-server-chi" }

func (p *PluginServerChi) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, "http", types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	serverGen := server.NewServer(new(server.StrategyChi), module, f)
	code, err := serverGen.Generate(annotations)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		f.Add(code)
	}

	return map[string]gomosaic.File{"server_chi_gen.go": f}, errs
}
