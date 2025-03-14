package http

import (
	"context"
	_ "embed"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/server"
	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type PluginServerEcho struct{}

func (p *PluginServerEcho) Name() string { return "http-server-echo" }

func (p *PluginServerEcho) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	services, err := service.ServiceLoad(module, "http", types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	serverGen := server.NewServer(new(server.StrategyEcho), module, f)
	code, err := serverGen.Generate(services)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		f.Add(code)
	}

	return map[string]gomosaic.File{"server.go": f}, errs
}
