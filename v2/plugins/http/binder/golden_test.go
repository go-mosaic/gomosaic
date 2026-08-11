package binder

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestHTTPBinderPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		strategy   Strategy
		filename   string
		goldenName string
	}{
		{
			name:       "std/basic",
			fixture:    "fixtures/basic/service.go",
			strategy:   &StdStrategy{},
			filename:   "http_bind_std_gen.go",
			goldenName: "http_bind_std",
		},
		{
			name:       "std/full",
			fixture:    "fixtures/full/service.go",
			strategy:   &StdStrategy{},
			filename:   "http_bind_std_gen.go",
			goldenName: "http_bind_std_full",
		},
		{
			name:       "chi/basic",
			fixture:    "fixtures/basic/service.go",
			strategy:   &ChiStrategy{},
			filename:   "http_bind_chi_gen.go",
			goldenName: "http_bind_chi",
		},
		{
			name:       "chi/full",
			fixture:    "fixtures/full/service.go",
			strategy:   &ChiStrategy{},
			filename:   "http_bind_chi_gen.go",
			goldenName: "http_bind_chi_full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, types := golden.ParseAndGenerate(t, ".", tt.fixture)

			plugin := NewPlugin(tt.strategy)

			pluginReg := gomosaic.NewPluginRegistry()
			pluginReg.MustRegister(plugin)

			fs := gomosaic.NewMemoryFileSystem("test")

			cg := gomosaic.NewCodeGenerator(pluginReg, gomosaic.DefaultTransformRegistry(), fs)

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/controller")

			outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("empty generation result")
			}

			fileContent, ok := fs.File(tt.filename)
			if !ok {
				t.Fatalf("file %s not found in generated: %v", tt.filename, outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
