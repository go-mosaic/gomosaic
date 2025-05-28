package middleware

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

type BodyFn func(group *jen.Group)

type Generator struct {
	nameTypeInfo  *gomosaic.NameTypeInfo
	structName    string
	constructName string
	params        []jen.Code
	methods       []jen.Code
	qualFunc      jenutils.QualFunc
}

func NewGenerator(
	nameTypeInfo *gomosaic.NameTypeInfo,
	name string,
	qualFunc jenutils.QualFunc,
	params []jen.Code,

) *Generator {
	return &Generator{
		nameTypeInfo:  nameTypeInfo,
		structName:    strcase.ToCamel(nameTypeInfo.Name) + name + "Middleware",
		constructName: name + strcase.ToCamel(nameTypeInfo.Name) + "Middleware",
		qualFunc:      qualFunc,
		params:        params,
	}
}

func (g *Generator) AddParam(paramName jen.Code, paramType jen.Code) {
	g.params = append(g.params, paramName, paramType)
}

func (g *Generator) Generate() (jen.Code, error) {
	group := jen.NewFile("")

	if g.nameTypeInfo.Type == nil || g.nameTypeInfo.Type.Interface == nil {
		return group.Null(), nil
	}

	middlewareTypeName := g.nameTypeInfo.Name + "Middleware"
	ifaceType := jen.Qual(g.nameTypeInfo.Package.Path, g.nameTypeInfo.Name)

	group.Type().Id(g.structName).StructFunc(func(group *jen.Group) {
		group.Id("next").Qual(g.nameTypeInfo.Package.Path, g.nameTypeInfo.Name)

		for i := 0; i < len(g.params); i += 2 {
			group.Add(g.params[i]).Add(g.params[i+1])
		}
	})

	group.Type().Id(middlewareTypeName).Op("=").Qual(gomosaic.RuntimePkg, "Middleware").Index(ifaceType)

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
		Id(middlewareTypeName).
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
