package server

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

func (g *generator) genRegisterHandlers(s *annotation.IfaceOpt) jen.Code {
	group := jen.NewFile("")

	group.Func().Id(s.NameTypeInfo.Name+"RegisterHandlers").Params(
		jen.Id("router").Do(func(s *jen.Statement) {
			if g.strategy.UsePtrType() {
				s.Op("*")
			}
		}).Qual(g.strategy.RouterPkg(), g.strategy.RouterType()),
		jen.Id("svc").Do(g.qual(s.NameTypeInfo.Package.Path, s.NameTypeInfo.Name)),
		jen.Id("opt").Op("*").Id(s.NameTypeInfo.Name+"Options"),
	).BlockFunc(func(group *jen.Group) {
		group.If(jen.Id("opt").Op("==").Nil()).Block(
			jen.Id("opt").Op("=").Op("&").Id(s.NameTypeInfo.Name + "Options").Values(),
		)

		group.Id("transportFactory").Op(":=").Qual(gomosaic.TransportFactoryPkg, "NewFactory").Call(
			jen.Id("opt").Dot("transportOptions").Op("..."),
		)

		group.List(jen.Id("tr"), jen.Err()).Op(":=").Id("transportFactory").Dot("Create").Call(
			jen.Qual(gomosaic.TransportFactoryPkg, g.strategy.TransportFactoryType()),
			jen.Id("router"),
		)
		group.If(jen.Err().Op("!=").Nil()).Block(jen.Panic(jen.Err()))

		for _, m := range s.Methods {
			pathParts := strings.Split(m.Path, "/")
			for _, pp := range m.PathParams {
				pathParts[pp.PathParamIndex] = g.strategy.PathParamWrap(pp.PathParamName)
			}

			group.Id("tr").Dot("AddRoute").Call(
				jen.Lit(m.Method),
				jen.Lit(strings.Join(pathParts, "/")),
				jen.Func().Params(
					jen.Id("req").Qual(gomosaic.TransportPkg, "Request"),
					jen.Id("resp").Qual(gomosaic.TransportPkg, "Response"),
				).Error().BlockFunc(func(group *jen.Group) {
					if len(m.BodyParams) > 0 {
						group.Add(g.genDecodeBodyParams(m, m.BodyParams))
					}
					if len(m.HeaderParams) > 0 {
						group.Add(g.genNonBodyParams(m.HeaderParams, func(name string) jen.Code {
							return jen.Id("req").Dot("Header").Call(jen.Lit(name))
						}))
					}
					if len(m.QueryParams) > 0 {
						group.Add(g.genQueryParams(m.QueryParams))
					}
					if len(m.PathParams) > 0 {
						group.Add(g.genNonBodyParams(m.PathParams, func(name string) jen.Code {
							return jen.Id("req").Dot("PathValue").Call(jen.Lit(name))
						}))
					}

					group.Add(g.genCallServiceMethod(m))

					respName := "respData"
					if len(m.BodyResults) == 1 && m.Single.Resp {
						respName = m.BodyResults[0].Var.Name
					} else {
						structFields := annotation.MakeStructFieldsFromResults(m.BodyResults, g.qual)
						if len(m.WrapResp.PathParts) > 0 {
							structFields = annotation.WrapStruct(m.WrapResp.PathParts, structFields)
						}
						group.Var().Id(respName).Struct(structFields)
						for _, result := range m.BodyResults {
							group.Id(respName).Do(func(s *jen.Statement) {
								for _, name := range m.WrapResp.PathParts {
									s.Dot(strcase.ToCamel(name))
								}
							}).Dot(strcase.ToCamel(result.Var.Name)).Op("=").Id(result.Var.Name)
						}
					}

					group.Id("resp").Dot("WriteData").Call(jen.Id("req"), jen.Id(respName))
					group.Return(jen.Nil())
				}),
				jen.Append(jen.Id("opt").Dot("middleware"), jen.Id("opt").Dot("middleware"+m.Func.Name).Op("...")).Op("..."),
			)
		}
	})

	return group
}
