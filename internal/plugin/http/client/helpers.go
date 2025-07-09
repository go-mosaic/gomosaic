package client

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/internal/plugin/http/annotation"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

func fullPrefixLowerCamel(methodOpt *annotation.MethodOpt) string {
	return strcase.ToLowerCamel(methodOpt.Iface.NameTypeInfo.Name) + methodOpt.Func.Name
}

func fullPrefixCamel(methodOpt *annotation.MethodOpt) string {
	return strcase.ToCamel(methodOpt.Iface.NameTypeInfo.Name) + methodOpt.Func.Name
}

func constShortName(methodOpt *annotation.MethodOpt) string {
	return fullPrefixLowerCamel(methodOpt) + "ShortName"
}

func constFullName(methodOpt *annotation.MethodOpt) string {
	return fullPrefixLowerCamel(methodOpt) + "FullName"
}

func clientStructName(ifaceOpt *annotation.IfaceOpt) string {
	return ifaceOpt.NameTypeInfo.Name + "Client"
}

func methodRequestName(methodOpt *annotation.MethodOpt) string {
	return fullPrefixCamel(methodOpt) + "Request"
}

func methodMakeRequestName(methodOpt *annotation.MethodOpt) string {
	return methodOpt.Func.Name + "Request"
}

func sprintfPath(methodOpt *annotation.MethodOpt) string {
	pathParamsMap := make(map[string]*annotation.MethodParamOpt, len(methodOpt.PathParams))
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

func pathParts(methodOpt *annotation.MethodOpt, fn func(name string) string) []string {
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
	return jen.Qual(annotation.IOPkg, "NopCloser").Call(code)
}
