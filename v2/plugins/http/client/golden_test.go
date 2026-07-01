package client

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestHTTPClientPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		filename   string
		goldenName string
	}{
		{
			name:       "basic",
			fixture:    "fixtures/basic/service.go",
			filename:   "client_gen.go",
			goldenName: "http_client",
		},
		{
			name:       "full",
			fixture:    "fixtures/full/service.go",
			filename:   "client_gen.go",
			goldenName: "http_client_full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, types := golden.ParseAndGenerate(t, ".", tt.fixture)

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

			fileContent, ok := fs.File(tt.filename)
			if !ok {
				t.Fatalf("файл %s не найден в сгенерированных: %v", tt.filename, outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
