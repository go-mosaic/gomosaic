package client

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/service"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

func errorTypeName(ifaceOpt *service.IfaceOpt) string {
	return ifaceOpt.NameTypeInfo.Name + "Error"
}

func clientStructName(ifaceOpt *service.IfaceOpt) string {
	return ifaceOpt.NameTypeInfo.Name + "Client"
}

func methodRequestName(methodOpt *service.MethodOpt) string {
	return methodOpt.Iface.NameTypeInfo.Name + methodOpt.Func.Name + "Request"
}

func methodReqName(methodOpt *service.MethodOpt) string {
	return methodOpt.Func.Name + "Request"
}

func sprintfPath(methodOpt *service.MethodOpt) string {
	pathParamsMap := make(map[string]*service.MethodParamOpt, len(methodOpt.PathParams))
	for _, param := range methodOpt.PathParams {
		pathParamsMap[param.Name] = param
	}

	parts := pathParts(methodOpt, func(name string) (result string) {
		result = "%s"
		if param, ok := pathParamsMap[name]; ok {
			if param.Var.Type.IsBasic && (param.Var.Type.BasicInfo == gomosaic.IsInteger || param.Var.Type.BasicInfo == gomosaic.IsInteger|gomosaic.IsUnsigned) {
				result = "%d"
			} else if param.Var.Type.BasicInfo == gomosaic.IsFloat {
				result = "%f"
			}
		}
		return
	})

	return strings.Join(parts, "/")
}

func pathParts(methodOpt *service.MethodOpt, fn func(name string) string) []string {
	pathParts := strings.Split(methodOpt.Path, "/")
	for i := range pathParts {
		s := pathParts[i]
		if strings.HasPrefix(s, ":") {
			pathParts[i] = fn(s[1:])
		}
	}

	return pathParts
}

func wrapIOCloser(code jen.Code) jen.Code {
	return jen.Qual(service.IOPkg, "NopCloser").Call(code)
}
