// Package middleware предоставляет унифицированный конструктор middleware-плагинов.
//
// В отличие от v1, где logmiddleware и metricmiddleware были отдельными плагинами
// с практически идентичным кодом, v2 предоставляет единый MiddlewarePlugin,
// который конфигурируется через Config.
//
// Пример использования:
//
//	// Log middleware
//	logPlugin := middleware.NewPlugin(middleware.Config{
//	    Name:       "log-middleware",
//	    MiddlewareName: "Log",
//	    Annotation: "log",
//	    Fields: []jen.Code{
//	        jen.Id("logger"), jen.Qual(gomosaic.SpanPkg, "Logger"),
//	    },
//	    BeforeFn: func(group *jen.Group, m *middleware.MethodOpt) {
//	        group.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartLogSpan").Call(
//	            jen.Id("m").Dot("logger"),
//	            jen.Lit(m.Func.ShortName),
//	        )
//	    },
//	    AfterFn: func(group *jen.Group, m *middleware.MethodOpt) {
//	        // ...
//	    },
//	})
//
//	// Metric middleware
//	metricPlugin := middleware.NewPlugin(middleware.Config{
//	    Name:       "metric-middleware",
//	    MiddlewareName: "Metric",
//	    Annotation: "metric",
//	    Fields: []jen.Code{
//	        jen.Id("metricCollector"), jen.Qual(gomosaic.SpanPkg, "MetricsCollector"),
//	    },
//	    BeforeFn: func(group *jen.Group, m *middleware.MethodOpt) { /* ... */ },
//	    AfterFn:  func(group *jen.Group, m *middleware.MethodOpt) { /* ... */ },
//	})
package middleware

import (
	"context"

	"github.com/dave/jennifer/jen"
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// BodyFn — функция, генерирующая тело метода.
type BodyFn func(group *jen.Group)

// Config содержит конфигурацию middleware-плагина.
type Config struct {
	// Name — уникальное имя плагина (например, "log-middleware").
	Name string

	// MiddlewareName — имя middleware (например, "Log", "Metric"),
	// используется для имён структур и конструкторов.
	MiddlewareName string

	// Fields — дополнительные поля структуры middleware (пары имя-тип).
	Fields []jen.Code

	// BeforeFn вызывается перед вызовом next.SomeMethod().
	BeforeFn BodyFn

	// AfterFn вызывается после вызова next.SomeMethod().
	AfterFn BodyFn

	// OutputFile — имя выходного файла (по умолчанию Name_gen.go).
	OutputFile string
}

type Plugin struct {
	cfg Config
}

func NewPlugin(cfg Config) *Plugin {
	if cfg.OutputFile == "" {
		cfg.OutputFile = cfg.Name + "_gen.go"
	}
	return &Plugin{cfg: cfg}
}

func (p *Plugin) Name() string { return p.cfg.Name }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (files map[string]gomosaic.File, errs error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	services, err := p.loadAnnotations(module, types)
	if err != nil {
		return nil, err
	}

	f := gomosaic.NewGoFile(module, outputDir)

	for _, service := range services {
		g := &Generator{
			nameTypeInfo:  service,
			structName:    strcase.ToCamel(service.Name) + p.cfg.MiddlewareName + "Middleware",
			constructName: p.cfg.MiddlewareName + strcase.ToCamel(service.Name) + "Middleware",
			qualFunc:      f.Qual,
			params:        append([]jen.Code{}, p.cfg.Fields...),
		}

		for _, m := range service.Type.Interface.Methods {
			g.GenerateMethod(m, func(group *jen.Group) {
				if p.cfg.BeforeFn != nil {
					p.cfg.BeforeFn(group)
				}
			}, func(group *jen.Group) {
				if p.cfg.AfterFn != nil {
					p.cfg.AfterFn(group)
				}
			})
		}

		code, err := g.Generate()
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		f.Add(code)
	}

	return map[string]gomosaic.File{p.cfg.OutputFile: f}, errs
}

func (p *Plugin) loadAnnotations(module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) ([]*gomosaic.NameTypeInfo, error) {
	var services []*gomosaic.NameTypeInfo

	annotations, err := annotation.Load(module, types)
	if err != nil {
		return nil, err
	}

	for _, annotation := range annotations {
		services = append(services, annotation.NameTypeInfo)
	}

	return services, nil
}

type Generator struct {
	nameTypeInfo  *gomosaic.NameTypeInfo
	structName    string
	constructName string
	params        []jen.Code
	methods       []jen.Code
	qualFunc      gomosaic.QualFunc
}

func (g *Generator) Generate() (jen.Code, error) {
	group := jen.NewFile("")

	if g.nameTypeInfo.Type == nil || g.nameTypeInfo.Type.Interface == nil {
		return group.Null(), nil
	}

	ifaceType := jen.Do(g.qualFunc(g.nameTypeInfo.Package.Path, g.nameTypeInfo.Name))

	group.Type().Id(g.structName).StructFunc(func(group *jen.Group) {
		group.Id("next").Do(g.qualFunc(g.nameTypeInfo.Package.Path, g.nameTypeInfo.Name))

		for i := 0; i < len(g.params); i += 2 {
			group.Add(g.params[i]).Add(g.params[i+1])
		}
	})

	structValues := jen.Dict{
		jen.Id("next"): jen.Id("next"),
	}

	for i := 0; i < len(g.params); i += 2 {
		structValues[g.params[i]] = g.params[i]
	}

	group.Func().
		Id(g.constructName).
		ParamsFunc(func(group *jen.Group) {
			for i := 0; i < len(g.params); i += 2 {
				group.Add(g.params[i]).Add(g.params[i+1])
			}
		}).
		Do(g.qualFunc(gomosaic.RuntimePkg, "Middleware")).Index(ifaceType).
		Block(
			jen.Return(
				jen.Func().Params(jen.Id("next").Add(ifaceType)).Add(ifaceType).Block(
					jen.Return(jen.Op("&").Id(g.structName).Values(structValues)),
				),
			),
		)

	for _, m := range g.methods {
		group.Add(m)
	}

	return group, nil
}

func (g *Generator) GenerateMethod(m *gomosaic.MethodInfo, beforeNextBodyFn, afterNextBodyFn BodyFn) {
	resultList := jen.Null()

	callFunc := jen.Id("m").Dot("next").Dot(m.Name).CallFunc(func(group *jen.Group) {
		for _, p := range m.Params {
			group.Id(p.Name)
		}
	})

	if len(m.Results) > 0 {
		resultList = jen.ListFunc(func(group *jen.Group) {
			for _, r := range m.Results {
				group.Id(r.Name)
			}
		})

		callFunc = jen.Add(resultList).Op(":=").Add(callFunc)
	}

	code := jen.Func().
		Params(
			jen.Id("m").Op("*").Id(g.structName),
		).
		Id(m.Name).
		ParamsFunc(func(group *jen.Group) {
			for _, p := range m.Params {
				group.Id(p.Name).Add(jenutils.TypeInfoQual(p.Type, g.qualFunc))
			}
		}).
		ParamsFunc(func(group *jen.Group) {
			for _, r := range m.Results {
				group.Add(jenutils.TypeInfoQual(r.Type, g.qualFunc))
			}
		}).
		BlockFunc(func(group *jen.Group) {
			beforeNextBodyFn(group)

			group.Add(callFunc)

			afterNextBodyFn(group)

			if len(m.Results) > 0 {
				group.Return(resultList)
			}
		})

	g.methods = append(g.methods, code)
}

var _ gomosaic.Generator = (*Plugin)(nil)
