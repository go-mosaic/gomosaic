// Package testclient предоставляет плагин генерации тестов HTTP-клиента.
// Портирован из v1 с полной поддержкой flatten, сравнения параметров и faker.
package testclient

import (
	"context"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

type Config struct {
	StatusCode int
	CheckError bool
}

const FakerPkg = "github.com/jaswdr/faker/v2"

type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "http-client-test" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir, gomosaic.UseTestPkg())
	f.ImportAlias(FakerPkg, "faker")

	gen := &generator{qual: f.Qual}
	f.Add(gen.Generate(annotations, []Config{
		{StatusCode: 200},
		{StatusCode: 400, CheckError: true},
	}))

	return map[string]gomosaic.File{"client_gen_test.go": f}, nil
}

// Ensure Plugin implements gomosaic.Generator.
var _ gomosaic.Generator = (*Plugin)(nil)
