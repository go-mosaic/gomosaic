package server

import (
	"github.com/dave/jennifer/jen"
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

type generator struct {
	qual              gomosaic.QualFunc
	module            *gomosaic.ModuleInfo
	transformRegistry *gomosaic.TransformRegistry
	allTypes          map[string]*gomosaic.NameTypeInfo
}

func (g *generator) generateService(svc *ServiceOpt) jen.Code {
	group := jen.NewFile("")
	serviceName := svc.NameTypeInfo.Name

	group.Func().Id("Register"+serviceName+"Server").Params(
		jen.Id("s").Op("*").Qual("google.golang.org/grpc", "Server"),
		jen.Id("svc").Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name)),
		jen.Id("mw").Op("...").Qual(gomosaic.RuntimePkg, "Middleware").Index(jen.Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name))),
	).BlockFunc(func(body *jen.Group) {
		serverName := strcase.ToLowerCamel(serviceName) + "Server"
		body.Id(serverName).Op(":=").Op("&").Id(serviceName + "Server").Values(jen.Dict{
			jen.Id("svc"): jen.Id("svc"),
		})
		body.Qual("google.golang.org/grpc", "Register"+serviceName+"Server").Call(
			jen.Id("s"),
			jen.Id(serverName),
		)
	})

	group.Type().Id(serviceName+"Server").Struct(
		jen.Id("svc").Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name)),
		jen.Id("middleware").Index().Qual(gomosaic.RuntimePkg, "Middleware").Index(jen.Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name))),
	)

	for _, m := range svc.Methods {
		group.Add(g.generateMethod(serviceName, svc, m))
	}

	if svc.ProtoPackage != "" {
		group.Add(g.generateConversionFunctions(svc))
		group.Add(g.generateStreamAdapters(serviceName, svc))
	}

	return group
}

func (g *generator) generateMethod(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	switch m.MethodType {
	case Unary:
		return g.generateUnaryMethod(serviceName, svc, m)
	case ServerStream, ClientStream, BidiStream:
		return g.generateStreamMethod(serviceName, svc, m)
	}
	return jen.NewFile("")
}

func (g *generator) qualType(t *gomosaic.TypeInfo) *jen.Statement {
	if t.Package != "" {
		return jen.Qual(t.Package, t.Name)
	}
	return jen.Id(t.Name)
}

// Ensure compilation
var _ = multierror.Append
