package http

import (
	"context"
	_ "embed"

	"github.com/jaswdr/faker/v2"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
	"github.com/go-mosaic/gomosaic/internal/plugin/http/testclient"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type PluginClientTesting struct{}

func (p *PluginClientTesting) Name() string { return "http-client-test" }

func (p *PluginClientTesting) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	services, err := service.ServiceLoad(module, "http", types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir, gomosaic.UseTestPkg())

	fake := faker.New()

	clientTestGen := testclient.NewClientTest(fake, f.Qual)
	f.Add(clientTestGen.Generate(services, []testclient.Config{
		{StatusCode: 200},                   //nolint: mnd
		{StatusCode: 400, CheckError: true}, //nolint: mnd
	}))

	return map[string]gomosaic.File{"client_test.go": f}, nil
}
