package validation

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestValidationPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		goldenName string
	}{
		{
			name:       "basic",
			fixture:    "fixtures/basic/user.go",
			goldenName: "validation",
		},
		{
			name:       "full",
			fixture:    "fixtures/full/full_user.go",
			goldenName: "validation_full",
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

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/model")

			outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("пустой результат генерации")
			}

			filename := "validation_gen.go"

			fileContent, ok := fs.File(filename)
			if !ok {
				t.Fatalf("файл %s не найден в сгенерированных: %v", filename, outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
