package middleware

import (
	"context"
	"testing"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

func TestMiddlewarePlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/service.go")

	cfg := Config{
		Name:           "log-middleware",
		MiddlewareName: "Log",
		Annotation:     "log",
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
			group.Comment("Завершение лог-спана")
			group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
		},
		OutputFile: "log_middleware_gen.go",
	}

	outputDir := module.Dir + "/internal/middleware"
	f := gomosaic.NewGoFile(module, outputDir)

	for _, nt := range types {
		if nt.Type.Interface == nil {
			continue
		}
		if !nt.Annotations.Has(cfg.Annotation) {
			continue
		}
		serviceName := nt.Name
		g := &Generator{
			nameTypeInfo:  nt,
			structName:    strcase.ToCamel(serviceName) + cfg.MiddlewareName + "Middleware",
			constructName: cfg.MiddlewareName + strcase.ToCamel(serviceName) + "Middleware",
			qualFunc:      f.Qual,
			params:        append([]jen.Code{}, cfg.Fields...),
		}
		for _, m := range nt.Type.Interface.Methods {
			g.GenerateMethod(m, func(group *jen.Group) {
				if cfg.BeforeFn != nil {
					cfg.BeforeFn(group)
				}
			}, func(group *jen.Group) {
				if cfg.AfterFn != nil {
					cfg.AfterFn(group)
				}
			})
		}
		code, _ := g.Generate()
		f.Add(code)
	}

	golden.AssertFile(t, f, "log_middleware")
	_ = context.Background()
}

func TestMetricMiddlewarePlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/metric_service.go")

	cfg := Config{
		Name:           "metric-middleware",
		MiddlewareName: "Metric",
		Annotation:     "metric",
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
			group.Comment("Завершение метрик-спана")
			group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
		},
		OutputFile: "metric_middleware_gen.go",
	}

	outputDir := module.Dir + "/internal/middleware"
	f := gomosaic.NewGoFile(module, outputDir)

	for _, nt := range types {
		if nt.Type.Interface == nil {
			continue
		}
		if !nt.Annotations.Has(cfg.Annotation) {
			continue
		}
		serviceName := nt.Name
		g := &Generator{
			nameTypeInfo:  nt,
			structName:    strcase.ToCamel(serviceName) + cfg.MiddlewareName + "Middleware",
			constructName: cfg.MiddlewareName + strcase.ToCamel(serviceName) + "Middleware",
			qualFunc:      f.Qual,
			params:        append([]jen.Code{}, cfg.Fields...),
		}
		for _, m := range nt.Type.Interface.Methods {
			g.GenerateMethod(m, func(group *jen.Group) {
				if cfg.BeforeFn != nil {
					cfg.BeforeFn(group)
				}
			}, func(group *jen.Group) {
				if cfg.AfterFn != nil {
					cfg.AfterFn(group)
				}
			})
		}
		code, _ := g.Generate()
		f.Add(code)
	}

	golden.AssertFile(t, f, "metric_middleware")
}
