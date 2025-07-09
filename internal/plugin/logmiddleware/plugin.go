package logmiddleware

import (
	"context"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/gen/middleware"
	"github.com/go-mosaic/gomosaic/internal/plugin/logmiddleware/annotation"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type Plugin struct{}

func (p *Plugin) Name() string { return "log-middleware" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	f := gomosaic.NewGoFile(module, outputDir)

	annotations, err := annotation.Load(module, "log", types)
	if err != nil {
		return nil, err
	}

	for _, service := range annotations {
		g := middleware.NewGenerator(
			service.NameTypeInfo,
			"Log",
			f.Qual,
			[]jen.Code{
				jen.Id("logger"), jen.Qual(gomosaic.SpanPkg, "Logger"),
			},
		)

		for _, m := range service.Methods {
			g.GenerateMethod(m.Func, func(group *jen.Group) {
				spanFuncName := "StartLogSpan"

				group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, spanFuncName).CallFunc(func(group *jen.Group) {
					group.Id("m").Dot("logger")
					group.Lit(m.Func.ShortName)
				})
			},
				func(group *jen.Group) {
					if v, ok := gomosaic.HasError(m.Func.Results); ok {
						group.If(jen.Id(v.Name).Op("!=").Nil()).Block(
							jen.Id("span").Dot("FinishWithError").Call(jen.Id("ctx"), jen.Id(v.Name)),
						).Else().Block(
							jen.Id("span").Dot("Finish").Call(jen.Id("ctx")),
						)
					} else {
						group.Id("span").Dot("Finish").Call(jen.Id("ctx"))
					}
				})
		}

		code, err := g.Generate()
		if err != nil {
			return nil, err
		}

		f.Add(code)
	}

	return map[string]gomosaic.File{"log_middleware_gen.go": f}, errs
}
