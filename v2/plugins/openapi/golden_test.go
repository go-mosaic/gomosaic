package openapi

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestOpenAPIPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		outputDir  string
		goldenName string
	}{
		{
			name:       "service",
			fixture:    "fixtures/service.go",
			outputDir:  "/internal/docs",
			goldenName: "openapi",
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

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+tt.outputDir)

			outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("пустой результат генерации")
			}

			filename := "openapi_gen.go"

			fileContent, ok := fs.File(filename)
			if !ok {
				t.Fatalf("файл %s не найден в сгенерированных: %v", filename, outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
