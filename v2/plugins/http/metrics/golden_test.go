package metrics

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestMetricsPlugin_Golden(t *testing.T) {
	tests := []struct {
		name       string
		strategy   Strategy
		goldenName string
	}{
		{name: "Chi", strategy: &ChiStrategy{}, goldenName: "metrics_chi"},
		{name: "Echo", strategy: &EchoStrategy{}, goldenName: "metrics_echo"},
		{name: "Fiber", strategy: &FiberStrategy{}, goldenName: "metrics_fiber"},
		{name: "Mux", strategy: &MuxStrategy{}, goldenName: "metrics_mux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, types := golden.ParseAndGenerate(t, ".", "fixtures/service/service.go")

			plugin := NewPlugin(tt.strategy)

			pluginReg := gomosaic.NewPluginRegistry()
			pluginReg.MustRegister(plugin)

			fs := gomosaic.NewMemoryFileSystem("test")

			cg := gomosaic.NewCodeGenerator(pluginReg, gomosaic.DefaultTransformRegistry(), fs)

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/metrics")

			outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("пустой результат генерации")
			}

			fileContent, ok := fs.File(tt.strategy.FileName())
			if !ok {
				t.Fatalf("файл %s не найден в сгенерированных: %v", tt.strategy.FileName(), outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
