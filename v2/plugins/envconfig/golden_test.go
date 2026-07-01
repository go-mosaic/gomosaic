package envconfig

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestEnvConfigPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		filename   string
		goldenName string
	}{
		{
			name:       "basic",
			fixture:    "fixtures/basic/config.go",
			filename:   "env_config_gen.go",
			goldenName: "env_config",
		},
		{
			name:       "nested",
			fixture:    "fixtures/full/nested_config.go",
			filename:   "env_config_gen.go",
			goldenName: "env_config_nested",
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

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/config")

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
