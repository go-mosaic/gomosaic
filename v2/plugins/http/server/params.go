package server

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
	"github.com/go-mosaic/gomosaic/v2/plugins/http/annotation"
)

func (g *generator) genDecodeBodyParams(opt *annotation.MethodOpt, params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("")

	for _, p := range params {
		group.Var().Id(strcase.ToLowerCamel(p.Var.Name)).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))
	}

	httpMethod := strings.ToUpper(opt.Method)
	reqName := "reqBody"
	varContentType := "contentType"

	switch httpMethod {
	case "POST", "PUT", "PATCH", "DELETE":
		group.Id(varContentType).Op(":=").Id("req").Dot("Header").Call(jen.Lit("Content-Type"))
		group.If(jen.Id(varContentType).Op("==").Lit("")).Block(
			jen.Id(varContentType).Op("=").Lit(opt.Default.ContentType),
		)
		group.Id("parts").Op(":=").Qual("strings", "Split").Call(jen.Id(varContentType), jen.Lit(";"))
		group.If(jen.Len(jen.Id("parts")).Op(">").Lit(0)).Block(
			jen.Id(varContentType).Op("=").Id("parts").Index(jen.Lit(0)),
		)

		group.Switch(jen.Id(varContentType)).BlockFunc(func(group *jen.Group) {
			group.Case(jen.Lit("application/json")).BlockFunc(func(group *jen.Group) {
				if len(params) == 1 && opt.Single.Req {
					group.Var().Id(reqName).Add(jenutils.TypeInfoQual(params[0].Var.Type, g.qual))
				} else {
					structFields := annotation.MakeStructFieldsFromParams(params, g.qual)
					if len(opt.WrapReq.PathParts) > 0 {
						structFields = annotation.WrapStruct(opt.WrapReq.PathParts, structFields)
					}
					group.Var().Id(reqName).Struct(structFields)
				}

				group.If(jen.Err().Op(":=").Id("req").Dot("ReadData").Call(jen.Op("&").Id(reqName)), jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Err()),
				)

				if len(params) == 1 && opt.Single.Req {
					group.Id(strcase.ToLowerCamel(params[0].Var.Name)).Op("=").Id(reqName)
				} else {
					for _, p := range params {
						group.Id(strcase.ToLowerCamel(p.Var.Name)).Op("=").Id(reqName).Add(annotation.Dot(append(opt.WrapReq.PathParts, strcase.ToCamel(p.Var.Name))...))
					}
				}
			})
		})
	}

	return group
}

func (g *generator) genQueryParams(params []*annotation.MethodParamOpt) jen.Code {
	group := jen.NewFile("")
	group.Id("q").Op(":=").Id("req").Dot("Queries").Call()

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)
		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

		queryName := p.Name
		if p.NameOpt.Value != "" {
			queryName = p.NameOpt.Value
		}

		valueID := jen.Id("q").Dot("Get").Call(jen.Lit(queryName))
		group.Add(g.transformRegistry.For(p.Var.Type).
			SetValueID(valueID).
			SetAssignID(jen.Id(name)).
			SetQualFunc(g.qual).
			SetErrStatements(jen.Return(jen.Err())).
			Parse())
	}

	return group
}

func (g *generator) genNonBodyParams(params []*annotation.MethodParamOpt, valueFn func(name string) jen.Code) jen.Code {
	group := jen.NewFile("")

	for _, p := range params {
		name := "param" + strcase.ToCamel(p.Name)
		group.Var().Id(name).Add(jenutils.TypeInfoQual(p.Var.Type, g.qual))

		valueName := strcase.ToLowerCamel(p.Name)
		if p.NameOpt.Value != "" {
			valueName = p.NameOpt.Value
		}

		group.Add(g.transformRegistry.For(p.Var.Type).
			SetValueID(valueFn(valueName)).
			SetAssignID(jen.Id(name)).
			SetQualFunc(g.qual).
			SetErrStatements(jen.Return(jen.Err())).
			Parse())
	}

	return group
}

func (g *generator) genCallServiceMethod(m *annotation.MethodOpt) jen.Code {
	group := jen.NewFile("")

	svcCall := jen.Do(func(s *jen.Statement) {
		s.ListFunc(func(group *jen.Group) {
			for _, r := range m.Results {
				group.Id(strcase.ToLowerCamel(r.Var.Name))
			}
		})
		if len(m.Results) > 0 {
			s.Op(":=")
		} else {
			s.Op("=")
		}
	}).Id("svc").Dot(m.Func.Name).CallFunc(func(group *jen.Group) {
		group.Id("req").Dot("Context").Call()
		for _, p := range m.Params {
			if p.Var.IsContext {
				continue
			}
			switch p.HTTPType {
			case annotation.PathHTTPType, annotation.CookieHTTPType, annotation.QueryHTTPType, annotation.HeaderHTTPType:
				group.Id("param" + strcase.ToCamel(p.Name))
			default:
				group.Id(strcase.ToLowerCamel(p.Var.Name))
			}
		}
	})

	group.Add(svcCall)
	group.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err()))

	return group
}
