package testclient

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/flatten"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

func (g *generator) genCheckError(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("").Null()

	errVar, hasErr := gomosaic.HasError(methodOpt.Func.Results)
	if !hasErr {
		return group
	}
	if !cfg.CheckError {
		group.If(jen.Id(errVar.Name).Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatalf").Call(jen.Lit("%s: %s"), jen.Lit("failed execute method "+methodOpt.Func.ShortName), jen.Id(errVar.Name)),
		)
		return group
	}
	group.If(jen.Id(errVar.Name).Op("==").Nil()).Block(
		jen.Id("t").Dot("Fatal").Call(jen.Lit("failed execute method " + methodOpt.Func.ShortName + " error is nil")),
	)
	return group
}

func (g *generator) genCheckBodyResult(methodOpt *annotation.MethodOpt, cfg Config) jen.Code {
	group := jen.NewFile("")

	if cfg.CheckError || len(methodOpt.BodyResults) == 0 {
		return jen.Null()
	}
	for _, r := range methodOpt.BodyResults {
		if !r.Var.Type.IsNamed {
			continue
		}
		if r.Var.Type.ElemType.Struct == nil {
			continue
		}
		st := r.Var.Type.ElemType.Struct
		for _, f := range st.Fields {
			for _, v := range flatten.Flatten(f) {
				if v.IsArray {
					continue
				}
				fieldPath := v.Paths.String()
				group.If(jen.Id(r.Var.Name).Op(".").Add(v.Path).Op("!=").Id("serverResponse").Op(".").Add(v.Path)).Block(
					jen.Id("t").Dot("Fatal").Call(jen.Lit("failed equal method " + methodOpt.Func.ShortName + " " + fieldPath + " not equal")),
				)
			}
		}
	}
	return group
}
