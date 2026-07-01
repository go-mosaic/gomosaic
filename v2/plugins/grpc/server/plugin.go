// Package server предоставляет плагин для генерации gRPC-сервера.
//
// Генерирует код регистрации gRPC-сервера, адаптеры и middleware
// на основе аннотированных интерфейсов.
//
// Аннотации:
//
//	@grpc-service                  — пометить интерфейс как gRPC-сервис
//	@grpc-method                   — gRPC-метод (унарный)
//	@grpc-method server-stream     — серверный стриминг
//	@grpc-method client-stream     — клиентский стриминг
//	@grpc-method bidi-stream       — двунаправленный стриминг
//
// Пример:
//
//	// @grpc-service
//	type UserService interface {
//	    // @grpc-method
//	    GetUser(ctx context.Context, req *GetUserRequest) (*User, error)
//
//	    // @grpc-method server-stream
//	    ListUsers(ctx context.Context, req *ListRequest, stream UserService_ListUsersServer) error
//	}
//
// Генерирует:
//   - Регистрацию сервера
//   - gRPC-хендлеры
//   - Цепочку middleware
package server

import (
	"context"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

// MethodType определяет тип gRPC-метода.
type MethodType string

const (
	Unary        MethodType = "unary"
	ServerStream MethodType = "server-stream"
	ClientStream MethodType = "client-stream"
	BidiStream   MethodType = "bidi-stream"
)

// MethodOpt содержит аннотации gRPC-метода.
type MethodOpt struct {
	Func       *gomosaic.MethodInfo
	MethodType MethodType
}

// ServiceOpt содержит аннотации gRPC-сервиса.
type ServiceOpt struct {
	NameTypeInfo *gomosaic.NameTypeInfo
	Methods      []*MethodOpt
}

// Plugin — плагин генерации gRPC-сервера.
type Plugin struct{}

// NewPlugin создает новый экземпляр плагина.
func NewPlugin() *Plugin { return &Plugin{} }

// Name возвращает имя плагина.
func (p *Plugin) Name() string { return "grpc-server" }

// Generate генерирует код gRPC-сервера.
func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)

	services := loadGRPCServices(types)
	if len(services) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := &generator{qual: f.Qual, module: module}

	for _, svc := range services {
		f.Add(gen.generateService(svc))
	}

	return map[string]gomosaic.File{"grpc_server_gen.go": f}, nil
}

func loadGRPCServices(types []*gomosaic.NameTypeInfo) []*ServiceOpt {
	var services []*ServiceOpt
	for _, nt := range types {
		if nt.Type.Interface == nil {
			continue
		}
		if !nt.Annotations.Has("grpc-service") {
			continue
		}

		svc := &ServiceOpt{NameTypeInfo: nt}
		for _, m := range nt.Type.Interface.Methods {
			mt := Unary
			if ann, ok := m.Annotations.Get("grpc-method"); ok && len(ann.Params) > 0 {
				mt = MethodType(ann.Params[0])
			}
			svc.Methods = append(svc.Methods, &MethodOpt{Func: m, MethodType: mt})
		}
		services = append(services, svc)
	}
	return services
}

type generator struct {
	qual   gomosaic.QualFunc
	module *gomosaic.ModuleInfo
}

func (g *generator) generateService(svc *ServiceOpt) jen.Code {
	group := jen.NewFile("")
	serviceName := svc.NameTypeInfo.Name

	// 1. Регистрация сервера
	group.Comment("Register" + serviceName + "Server регистрирует gRPC-сервер.")
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

	// 2. Структура сервера
	group.Type().Id(serviceName+"Server").Struct(
		jen.Id("svc").Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name)),
		jen.Id("middleware").Index().Qual(gomosaic.RuntimePkg, "Middleware").Index(jen.Do(g.qual(svc.NameTypeInfo.Package.Path, svc.NameTypeInfo.Name))),
	)

	// 3. Методы сервера
	for _, m := range svc.Methods {
		group.Add(g.generateMethod(serviceName, svc, m))
	}

	return group
}

func (g *generator) generateMethod(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	switch m.MethodType {
	case Unary:
		return g.generateUnaryMethod(serviceName, m)
	case ServerStream, ClientStream, BidiStream:
		return g.generateStreamMethod(serviceName, m)
	}
	return jen.NewFile("")
}

func (g *generator) generateUnaryMethod(serviceName string, m *MethodOpt) jen.Code {
	group := jen.NewFile("")
	params := m.Func.Params
	results := m.Func.Results

	// Проверяем сигнатуру: ctx context.Context, req *Request
	if len(params) < 2 {
		return group
	}
	reqParam := params[1]
	respParam := results[0] // Первый результат (не ошибка)

	group.Func().Params(
		jen.Id("s").Op("*").Id(serviceName+"Server"),
	).Id(m.Func.Name).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Add(jenutils.TypeInfoQual(reqParam.Type, g.qual)),
	).Params(
		jenutils.TypeInfoQual(respParam.Type, g.qual),
		jen.Error(),
	).BlockFunc(func(body *jen.Group) {
		// Вызов сервиса
		body.List(jen.Id("resp"), jen.Err()).Op(":=").Id("s").Dot("svc").Dot(m.Func.Name).Call(
			jen.Id("ctx"),
			jen.Id("req"),
		)
		body.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		)
		body.Return(jen.Id("resp"), jen.Nil())
	})

	return group
}

func (g *generator) generateStreamMethod(serviceName string, m *MethodOpt) jen.Code {
	group := jen.NewFile("")
	params := m.Func.Params

	group.Func().Params(
		jen.Id("s").Op("*").Id(serviceName + "Server"),
	).Id(m.Func.Name).ParamsFunc(func(group *jen.Group) {
		for _, p := range params {
			switch {
			case p.IsContext:
				group.Id("ctx").Qual("context", "Context")
			case !isStreamType(p):
				group.Id(p.Name).Add(g.qualType(p.Type))
			default:
				group.Id(p.Name).Add(g.qualType(p.Type))
			}
		}
	}).Error().BlockFunc(func(body *jen.Group) {
		callArgs := make([]jen.Code, 0)
		for _, p := range params {
			callArgs = append(callArgs, jen.Id(p.Name))
		}
		body.Return(jen.Id("s").Dot("svc").Dot(m.Func.Name).Call(callArgs...))
	})

	return group
}

func (g *generator) qualType(t *gomosaic.TypeInfo) *jen.Statement {
	if t.Package != "" {
		return jen.Qual(t.Package, t.Name)
	}
	return jen.Id(t.Name)
}

func isStreamType(v *gomosaic.VarInfo) bool {
	name := v.Type.Name
	return strings.Contains(name, "Server") || strings.Contains(name, "Client")
}

// Ensure compilation
var _ = multierror.Append
var _ gomosaic.Generator = (*Plugin)(nil)
