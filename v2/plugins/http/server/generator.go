package server

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

// generator генерирует код HTTP-сервера.
type generator struct {
	module            *gomosaic.ModuleInfo
	strategy          Strategy
	qual              gomosaic.QualFunc
	transformRegistry *gomosaic.TransformRegistry
}

// Generate генерирует весь код сервера.
func (g *generator) Generate(services []*annotation.IfaceOpt) (jen.Code, error) {
	group := jen.NewFile("")
	group.Add(g.genServiceOptions(services))
	for _, s := range services {
		group.Add(g.genRegisterHandlers(s))
	}
	return group, nil
}

func (g *generator) genServiceOptions(services []*annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")
	for _, s := range services {
		middlewareType := jen.Qual(gomosaic.TransportPkg, "Middleware")
		optionsName := s.NameTypeInfo.Name + "Options"
		group.Add(g.genTypeOptions(optionsName, middlewareType, s.Methods))
	}
	return group
}

func (g *generator) genTypeOptions(optionsName string, middlewareType jen.Code, methods []*annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")
	transportOptions := jen.Do(g.qual(gomosaic.TransportPkg, "TransportOption"))

	group.Type().Id(optionsName).StructFunc(func(group *jen.Group) {
		group.Id("transportOptions").Index().Add(transportOptions)
		group.Id("middleware").Index().Add(middlewareType)
		for _, m := range methods {
			group.Id("middleware" + m.Func.Name).Index().Add(middlewareType)
		}
	})

	group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("TransportOptions").Params(jen.Id("opts").Op("...").Add(transportOptions)).Op("*").Id(optionsName).Block(
		jen.Id("o").Dot("transportOptions").Op("=").Append(jen.Id("o").Dot("transportOptions"), jen.Id("opts").Op("...")),
		jen.Return(jen.Id("o")),
	).Line()

	group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("Middleware").Params(jen.Id("middleware").Op("...").Add(middlewareType)).Op("*").Id(optionsName).Block(
		jen.Id("o").Dot("middleware").Op("=").Append(jen.Id("o").Dot("middleware"), jen.Id("middleware").Op("...")),
		jen.Return(jen.Id("o")),
	).Line()

	for _, m := range methods {
		group.Func().Params(jen.Id("o").Op("*").Id(optionsName)).Id("Middleware"+m.Func.Name).Params(jen.Id("middleware").Op("...").Add(middlewareType)).Op("*").Id(optionsName).Block(
			jen.Id("o").Dot("middleware"+m.Func.Name).Op("=").Append(jen.Id("o").Dot("middleware"+m.Func.Name), jen.Id("middleware").Op("...")),
			jen.Return(jen.Id("o")),
		).Line()
	}

	return group
}
