package http

import (
	"context"
	_ "embed"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/annotation"
	"github.com/go-mosaic/gomosaic/internal/plugin/http/client"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type PluginClient struct{}

func (p *PluginClient) Name() string { return "http-client" }

func (p *PluginClient) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	a, err := annotation.Load(module, "http", types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	clientGen := client.NewClientGenerator(f)
	code, err := clientGen.Generate(a)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		f.Add(code)
	}

	return map[string]gomosaic.File{"client_gen.go": f}, errs
}
