package log

import (
	"context"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/gen/middleware"
	"github.com/go-mosaic/gomosaic/internal/plugin/log/service"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type Plugin struct{}

func (p *Plugin) Name() string { return "logging" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	f := gomosaic.NewGoFile(module, outputDir)

	services, err := service.ServiceLoad(module, "log", types)
	if err != nil {
		return nil, err
	}

	for _, service := range services {
		if !service.Enable {
			continue
		}

		g := middleware.NewMiddlewareGenerator(
			service.NameTypeInfo,
			"Logging",
			f.Qual,
			[]jen.Code{
				jen.Id("logger"), jen.Qual(gomosaic.LogPkg, "Logger"),
			},
		)

		var foundMetric bool

		for _, m := range service.Methods {
			g.GenerateMethod(m.Func, func(group *jen.Group) {
				spanFuncName := "StartLogSpan"
				if m.Metric {
					foundMetric = true
					spanFuncName = "StartMetricSpan"
				}

				group.Id("span").Op(":=").Qual(gomosaic.LogPkg, spanFuncName).CallFunc(func(group *jen.Group) {
					group.Id("ctx")
					group.Id("m").Dot("logger")
					group.Lit(m.Func.ShortName)
					if m.Metric {
						group.Id("m").Dot("metricCollector")
					}
				})
			},
				func(group *jen.Group) {
					if v, ok := gomosaic.HasError(m.Func.Results); ok {
						group.If(jen.Id(v.Name).Op("!=").Nil()).Block(
							jen.Id("span").Dot("FinishWithError").Call(jen.Id(v.Name)),
						).Else().Block(
							jen.Id("span").Dot("Finish").Call(),
						)
					} else {
						group.Id("span").Dot("Finish").Call()
					}
				})
		}

		if foundMetric {
			g.AddParam(jen.Id("metricCollector"), jen.Qual(gomosaic.LogPkg, "MetricsCollector"))
		}

		code, err := g.Generate()
		if err != nil {
			return nil, err
		}

		f.Add(code)
	}

	return map[string]gomosaic.File{"logging.go": f}, errs
}
