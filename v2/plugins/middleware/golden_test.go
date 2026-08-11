package middleware

import (
	"context"
	"testing"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestMiddlewarePlugin_Golden(t *testing.T) {
	tests := []struct {
		name           string
		fixtureFile    string
		config         Config
		outputFileName string
		goldenName     string
	}{
		{
			name:        "LogMiddleware",
			fixtureFile: "fixtures/logservice/log_service.go",
			config: Config{
				Name:           "log-middleware",
				MiddlewareName: "Log",
				Fields: []jen.Code{
					jen.Id("logger"),
					jen.Qual(gomosaic.SpanPkg, "Logger"),
				},
				BeforeFn: func(group *jen.Group) {
					group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartLogSpan").Call(
						jen.Id("m").Dot("logger"),
						jen.Id("ctx"),
					)
				},
				AfterFn: func(group *jen.Group) {
					group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
				},
				OutputFile: "log_middleware_gen.go",
			},
			outputFileName: "log_middleware_gen.go",
			goldenName:     "log_middleware",
		},
		{
			name:        "MetricMiddleware",
			fixtureFile: "fixtures/metricservice/metric_service.go",
			config: Config{
				Name:           "metric-middleware",
				MiddlewareName: "Metric",
				Fields: []jen.Code{
					jen.Id("collector"),
					jen.Qual(gomosaic.SpanPkg, "MetricsCollector"),
				},
				BeforeFn: func(group *jen.Group) {
					group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartMetricSpan").Call(
						jen.Id("m").Dot("collector"),
						jen.Id("ctx"),
					)
				},
				AfterFn: func(group *jen.Group) {
					group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
				},
				OutputFile: "metric_middleware_gen.go",
			},
			outputFileName: "metric_middleware_gen.go",
			goldenName:     "metric_middleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, types := golden.ParseAndGenerate(t, ".", tt.fixtureFile)

			plugin := NewPlugin(tt.config)

			pluginReg := gomosaic.NewPluginRegistry()
			pluginReg.MustRegister(plugin)

			fs := gomosaic.NewMemoryFileSystem("test")

			cg := gomosaic.NewCodeGenerator(pluginReg, gomosaic.DefaultTransformRegistry(), fs)

			ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/middleware")

			outputFiles, err := cg.Generate(ctx, module, types, plugin.Name())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("пустой результат генерации")
			}

			fileContent, ok := fs.File(tt.outputFileName)
			if !ok {
				t.Fatalf("файл %s не найден в сгенерированных: %v", tt.outputFileName, outputFiles)
			}

			golden.AssertBytes(t, fileContent, tt.goldenName)
		})
	}
}
