package testclient

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestClientTestPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/service.go")

	plugin := NewPlugin()

	pluginReg := gomosaic.NewPluginRegistry()
	pluginReg.MustRegister(plugin)

	fs := gomosaic.NewMemoryFileSystem("test")

	cg := gomosaic.NewCodeGenerator(pluginReg, gomosaic.DefaultTransformRegistry(), fs)

	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/client")

	outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(outputFiles) == 0 {
		t.Fatal("пустой результат генерации")
	}

	filename := "client_gen_test.go"

	fileContent, ok := fs.File(filename)
	if !ok {
		t.Fatalf("файл %s не найден в сгенерированных: %v", filename, outputFiles)
	}

	golden.AssertBytes(t, fileContent, "client_test")
}
