// Package testclient предоставляет плагин генерации тестов HTTP-клиента.
// Портирован из v1 с полной поддержкой flatten, сравнения параметров и faker.
package testclient

import (
	"context"
	"net/http"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// Config задаёт ожидаемый статус и проверку ошибки в тесте.
type Config struct {
	StatusCode int
	CheckError bool
}

// FakerPkg содержит путь пакета faker, используемого в тестах.
const FakerPkg = "github.com/jaswdr/faker/v2"

// Plugin генерирует тесты HTTP-клиента.
type Plugin struct{}

// NewPlugin создаёт плагин тестов HTTP-клиента.
func NewPlugin() *Plugin { return &Plugin{} }

// Name реализует gomosaic.Generator.
func (p *Plugin) Name() string { return "http-client-test" }

// Generate реализует gomosaic.Generator.
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
		{StatusCode: http.StatusOK},
		{StatusCode: http.StatusBadRequest, CheckError: true},
	}))

	return map[string]gomosaic.File{"client_gen_test.go": f}, nil
}

// Гарантирует, что Plugin реализует gomosaic.Generator.
var _ gomosaic.Generator = (*Plugin)(nil)
