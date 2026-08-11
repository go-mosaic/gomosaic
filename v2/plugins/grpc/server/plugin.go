// Package server предоставляет плагин для генерации gRPC-сервера.
//
// Генерирует код регистрации gRPC-сервера, адаптеры и middleware
// на основе интерфейсов.
//
// Аннотации:
//
//	@proto-package <package>       — указывает пакет с protobuf-типами (например: @proto-package github.com/project/api/proto)
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
package server

import (
	"context"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

type MethodType string

const (
	Unary        MethodType = "unary"
	ServerStream MethodType = "server-stream"
	ClientStream MethodType = "client-stream"
	BidiStream   MethodType = "bidi-stream"
)

type MethodOpt struct {
	Func       *gomosaic.MethodInfo
	MethodType MethodType
}

type ServiceOpt struct {
	NameTypeInfo *gomosaic.NameTypeInfo
	ProtoPackage string // Пакет с protobuf-типами
	Methods      []*MethodOpt
}

type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "grpc-server" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
	outputDir := gomosaic.OutputDirFromContext(ctx)
	transformReg := gomosaic.TransformRegistryFromContext(ctx)

	services := loadGRPCServices(types)
	if len(services) == 0 {
		return nil, nil
	}

	f := gomosaic.NewGoFile(module, outputDir)
	gen := &generator{
		qual:              f.Qual,
		module:            module,
		transformRegistry: transformReg,
		allTypes:          buildTypeIndex(types),
	}

	for _, svc := range services {
		f.Add(gen.generateService(svc))
	}

	return map[string]gomosaic.File{"grpc_server_gen.go": f}, nil
}

// buildTypeIndex строит индекс типов из NameTypeInfo.
func buildTypeIndex(types []*gomosaic.NameTypeInfo) map[string]*gomosaic.NameTypeInfo {
	idx := make(map[string]*gomosaic.NameTypeInfo)
	for _, nt := range types {
		idx[nt.Name] = nt
	}
	return idx
}

func loadGRPCServices(types []*gomosaic.NameTypeInfo) []*ServiceOpt {
	var services []*ServiceOpt
	for _, nt := range types {
		if nt.Type.Interface == nil {
			continue
		}

		svc := &ServiceOpt{NameTypeInfo: nt}

		if ann, ok := nt.Annotations.Get("proto-package"); ok && len(ann.Params) > 0 {
			svc.ProtoPackage = ann.Params[0]
		}

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

var _ gomosaic.Generator = (*Plugin)(nil)
