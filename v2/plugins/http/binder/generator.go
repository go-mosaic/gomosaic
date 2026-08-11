package binder

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

type generator struct {
	module            *gomosaic.ModuleInfo
	strategy          Strategy
	qual              gomosaic.QualFunc
	transformRegistry *gomosaic.TransformRegistry
}

func (g *generator) Generate(structs []*StructOpt) (jen.Code, error) {
	group := jen.NewFile("")
	for _, st := range structs {
		group.Add(g.genBindMethod(st))
	}
	return group, nil
}

func (g *generator) genBindMethod(st *StructOpt) jen.Code {
	recvName := "p"

	fn := jen.Func().Params(
		jen.Id(recvName).Op("*").Do(g.qual(st.NameTypeInfo.Package.Path, st.NameTypeInfo.Name)),
	).Id("Bind").Params(
		jen.Id("r").Op("*").Qual("net/http", "Request"),
	).Error()

	fn.BlockFunc(func(body *jen.Group) {
		if st.hasFormFields {
			g.genParseFormPreamble(body, st)
		}

		for _, fo := range st.Fields {
			g.genFieldBinding(body, recvName, fo)
		}
		body.Return(jen.Nil())
	})

	return fn
}

func (g *generator) genParseFormPreamble(body *jen.Group, st *StructOpt) {
	body.If(
		jen.Err().Op(":=").Id("r").Dot("ParseMultipartForm").Call(jen.Lit(int64(st.FormMaxMemory))),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Err()),
	)
}

func (g *generator) genFieldBinding(body *jen.Group, recvName string, fo *FieldOpt) {
	paramName := fo.NameOverride
	fieldName := strcase.ToCamel(fo.Var.Name)

	switch fo.Source {
	case SourceQuery:
		g.genQueryBinding(body, recvName, fieldName, paramName, fo)
	case SourcePath:
		g.genPathBinding(body, recvName, fieldName, paramName, fo)
	case SourceHeader:
		g.genHeaderBinding(body, recvName, fieldName, paramName, fo)
	case SourceCookie:
		g.genCookieBinding(body, recvName, fieldName, paramName, fo)
	case SourceForm:
		g.genFormBinding(body, recvName, fieldName, paramName, fo)
	case SourceFile:
		g.genFileBinding(body, recvName, fieldName, paramName, fo)
	}
}

func (g *generator) genQueryBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	valueID := jen.Id("r").Dot("URL").Dot("Query").Call().Dot("Get").Call(jen.Lit(paramName))

	varElseBlock := func(elseBody *jen.Group) {
		g.genParseAndAssign(elseBody, recvName, fieldName, valueID.Clone(), fo)
	}

	isEmpty := jen.Id("r").Dot("URL").Dot("Query").Call().Dot("Get").Call(jen.Lit(paramName)).Op("==").Lit("")

	if fo.Default != "" {
		body.If(isEmpty).Block(
			g.genDefaultAssign(recvName, fieldName, fo),
		).Else().BlockFunc(varElseBlock)
		return
	}

	if fo.Required {
		body.If(isEmpty).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required query parameter %q is missing", paramName)),
			)),
		)
	}

	g.genParseAndAssign(body, recvName, fieldName, valueID, fo)
}

func (g *generator) genPathBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	valueID := g.strategy.PathParamExtract("r", paramName)

	if fo.Required {
		body.If(valueID.Clone().Op("==").Lit("")).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required path parameter %q is missing", paramName)),
			)),
		)
	}

	g.genParseAndAssign(body, recvName, fieldName, valueID, fo)
}

func (g *generator) genHeaderBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	valueID := jen.Id("r").Dot("Header").Dot("Get").Call(jen.Lit(paramName))

	varElseBlock := func(elseBody *jen.Group) {
		g.genParseAndAssign(elseBody, recvName, fieldName, valueID.Clone(), fo)
	}

	isEmpty := jen.Id("r").Dot("Header").Dot("Get").Call(jen.Lit(paramName)).Op("==").Lit("")

	if fo.Default != "" {
		body.If(isEmpty).Block(
			g.genDefaultAssign(recvName, fieldName, fo),
		).Else().BlockFunc(varElseBlock)
		return
	}

	if fo.Required {
		body.If(isEmpty).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required header %q is missing", paramName)),
			)),
		)
	}

	g.genParseAndAssign(body, recvName, fieldName, valueID, fo)
}

func (g *generator) genCookieBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	cookieVar := "c"
	body.List(jen.Id(cookieVar), jen.Err()).Op(":=").Id("r").Dot("Cookie").Call(jen.Lit(paramName))

	valueID := jen.Id(cookieVar).Dot("Value")

	varElseBlock := func(elseBody *jen.Group) {
		g.genParseAndAssign(elseBody, recvName, fieldName, valueID.Clone(), fo)
	}

	if fo.Default != "" {
		body.If(jen.Err().Op("!=").Nil()).Block(
			g.genDefaultAssign(recvName, fieldName, fo),
		).Else().BlockFunc(varElseBlock)
		return
	}

	if fo.Required {
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required cookie %q is missing", paramName)),
			)),
		)
	}

	body.If(jen.Err().Op("==").Nil()).BlockFunc(func(cookieBody *jen.Group) {
		g.genParseAndAssign(cookieBody, recvName, fieldName, valueID, fo)
	})
}

func (g *generator) genFormBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	valueID := jen.Id("r").Dot("FormValue").Call(jen.Lit(paramName))

	varElseBlock := func(elseBody *jen.Group) {
		g.genParseAndAssign(elseBody, recvName, fieldName, valueID.Clone(), fo)
	}

	isEmpty := valueID.Clone().Op("==").Lit("")

	if fo.Default != "" {
		body.If(isEmpty).Block(
			g.genDefaultAssign(recvName, fieldName, fo),
		).Else().BlockFunc(varElseBlock)
		return
	}

	if fo.Required {
		body.If(isEmpty).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required form field %q is missing", paramName)),
			)),
		)
	}

	g.genParseAndAssign(body, recvName, fieldName, valueID, fo)
}

func (g *generator) genFileBinding(body *jen.Group, recvName, fieldName, paramName string, fo *FieldOpt) {
	fileVar := "f"
	hdrVar := "hdr"
	body.List(
		jen.Id(fileVar),
		jen.Id(hdrVar),
		jen.Err(),
	).Op(":=").Id("r").Dot("FormFile").Call(jen.Lit(paramName))

	if fo.Required {
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("required file %q is missing", paramName)),
			)),
		)
		body.Defer().Id(fileVar).Dot("Close").Call()
		body.Add(jen.Id(recvName).Dot(fieldName)).Op("=").Id(hdrVar)
		return
	}

	body.If(jen.Err().Op("!=").Nil()).Block(
		jen.Id(recvName).Dot(fieldName).Op("=").Nil(),
	).Else().Block(
		jen.Defer().Id(fileVar).Dot("Close").Call(),
		jen.Id(recvName).Dot(fieldName).Op("=").Id(hdrVar),
	)
}

func (g *generator) genParseAndAssign(body *jen.Group, recvName, fieldName string, valueID jen.Code, fo *FieldOpt) {
	assignID := jen.Id(recvName).Dot(fieldName)

	body.Add(g.transformRegistry.For(fo.Var.Type).
		SetValueID(valueID).
		SetAssignID(assignID).
		SetQualFunc(g.qual).
		SetErrStatements(jen.Return(jen.Err())).
		Parse())
}

func (g *generator) genDefaultAssign(recvName, fieldName string, fo *FieldOpt) jen.Code {
	assignID := jen.Id(recvName).Dot(fieldName)
	t := fo.Var.Type

	if t.IsBasic {
		switch {
		case t.BasicInfo == gomosaic.IsString:
			return jen.Add(assignID).Op("=").Lit(fo.Default)
		case (t.BasicInfo & gomosaic.IsInteger) != 0:
			return jen.Add(assignID).Op("=").Lit(defaultInt(fo.Default))
		case (t.BasicInfo & gomosaic.IsFloat) != 0:
			return jen.Add(assignID).Op("=").Lit(defaultFloat(fo.Default))
		case t.BasicInfo == gomosaic.IsBoolean:
			return jen.Add(assignID).Op("=").Lit(defaultBool(fo.Default))
		}
	}

	return jen.Null()
}

func defaultInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func defaultFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func defaultBool(s string) bool {
	return s == "true" || s == "1"
}
