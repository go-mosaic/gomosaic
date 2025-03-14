package service

import (
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/option"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

type ErrorOpt struct {
	Expr       string `option:",fromValue"`
	Type       string `option:"type,fromParam"`
	StatusCode bool   `option:"statusCode,fromParam"`
	Required   bool   `option:"required,fromParam"`
	TagName    string
	FldName    string
	MethodName string
}

type MethodTemplOpt struct {
	Path     string `option:"path"`
	PkgPath  string
	FuncName string
	Params   []string
}

type MethodOpt struct {
	Iface *IfaceOpt
	Func  *gomosaic.MethodInfo

	TimeFormat         string           `option:"time-format"`
	Method             string           `option:"method" valid:"in,params:'GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH'"`
	Path               string           `option:"path"`
	Openapi            MethodOpenapiOpt `option:"openapi"`
	MultipartMaxMemory int64            `option:"multipart-max-memory"`
	Query              MethodQueryOpt   `option:"query"`
	WrapReq            MethodWrapOpt    `option:"wrap-req"`
	WrapResp           MethodWrapOpt    `option:"wrap-resp"`
	Single             SingleOpt        `option:"single"`
	Templ              MethodTemplOpt   `option:"templ"`
	Context            *gomosaic.VarInfo
	Error              *gomosaic.VarInfo
	Params             []*MethodParamOpt
	Results            []*MethodResultOpt
	BodyParams         []*MethodParamOpt
	QueryParams        []*MethodParamOpt
	HeaderParams       []*MethodParamOpt
	CookieParams       []*MethodParamOpt
	PathParams         []*MethodParamOpt
	BodyResults        []*MethodResultOpt
	HeaderResults      []*MethodResultOpt
	CookieResults      []*MethodResultOpt
}

type SingleOpt struct {
	Req  bool `option:"req,asFlag"`
	Resp bool `option:"resp,asFlag"`
}

type MethodWrapOpt struct {
	Path      string `option:"path"`
	PathParts []string
}

type MethodXMLOpt struct {
	ReqName  string `option:"req-name"`
	RespName string `option:"resp-name"`
}

type MethodQueryOpt struct {
	Values []QueryValueOpt `option:"value,inline"`
}

type QueryValueOpt struct {
	Name   string   `option:",fromValue" valid:"required"`
	Values []string `option:",fromOptions"`
}

type MethodOpenapiOpt struct {
	Tags []string `option:"tags"`
}

type MethodParamNameOpt struct {
	Value     string `option:",fromValue"`
	Omitempty bool   `option:",fromOption"`
	Format    string `option:",fromParam"`
}

type MethodParamOpt struct {
	Var            *gomosaic.VarInfo
	Name           string
	NameOpt        MethodParamNameOpt `option:"name,inline"`
	HTTPType       string             `option:"type"`
	Required       bool               `option:"required,asFlag"`
	PathParamIndex int
	PathParamName  string
}

type MethodResultOpt struct {
	Var      *gomosaic.VarInfo
	Name     string
	NameOpt  MethodParamNameOpt `option:"name,inline"`
	HTTPType string             `option:"type"`
	Required bool               `option:"required"`
	Flat     bool               `option:"flat"`
}

type IfaceOpt struct {
	NameTypeInfo       *gomosaic.NameTypeInfo
	Errors             []ErrorOpt `option:"error,inline"`
	ErrorText          string     `option:"error-text"`
	Example            string     `option:"example" valid:"in,params:'http curl'"`
	DefaultContentType string     `option:"default-content-type"`
	Methods            []*MethodOpt
}

func ServiceLoad(module *gomosaic.ModuleInfo, prefix string, types []*gomosaic.NameTypeInfo) (interfaces []*IfaceOpt, errs error) {
	for _, nameTypeInfo := range types {
		if nameTypeInfo.Type.Interface == nil {
			continue
		}

		ifaceOpt := &IfaceOpt{NameTypeInfo: nameTypeInfo}

		err := option.Unmarshal(prefix, nameTypeInfo.Annotations, ifaceOpt)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		for _, m := range nameTypeInfo.Type.Interface.Methods {
			methodOpt := &MethodOpt{Iface: ifaceOpt, Func: m}

			err := option.Unmarshal(prefix, m.Annotations, methodOpt)
			if err != nil {
				errs = multierror.Append(errs, err)
			}

			if methodOpt.Templ.Path != "" {
				pkgPath, funcName, err := module.ParsePath(methodOpt.Templ.Path)
				if err != nil {
					errs = multierror.Append(errs, err)
				} else {
					funcName, params, err := gomosaic.ParseFunctionCall(funcName)
					if err != nil {
						errs = multierror.Append(errs, err)
					} else {
						methodOpt.Templ.PkgPath = pkgPath
						methodOpt.Templ.FuncName = funcName
						methodOpt.Templ.Params = params
					}
				}
			}

			if methodOpt.WrapReq.Path != "" {
				methodOpt.WrapReq.PathParts = strings.Split(methodOpt.WrapReq.Path, ".")
			}

			if methodOpt.WrapResp.Path != "" {
				methodOpt.WrapResp.PathParts = strings.Split(methodOpt.WrapResp.Path, ".")
			}

			if len(m.Params) == 0 || !m.Params[0].IsContext {
				errs = multierror.Append(errs, gomosaic.Error("Не верная сигнатура метода, первым параметром обязателен тип context.Context", m.Pos))
				continue
			}

			for _, param := range m.Params {
				methodParamOpt := &MethodParamOpt{Var: param}
				err := option.Unmarshal(prefix, param.Annotations, methodParamOpt)
				if err != nil {
					errs = multierror.Append(errs, err)
				}

				if param.IsContext {
					methodOpt.Context = param
				}

				if methodParamOpt.HTTPType == "" {
					methodParamOpt.HTTPType = BodyHTTPType
				}

				methodParamOpt.Name = formatName(methodParamOpt.NameOpt.Value, methodParamOpt.Var.Name, methodParamOpt.NameOpt.Format)

				methodOpt.Params = append(methodOpt.Params, methodParamOpt)
			}

			parts := strings.Split(methodOpt.Path, "/")
			for idx, part := range parts {
				if strings.HasPrefix(part, ":") {
					pathParamName := part[1:]
					for i, p := range methodOpt.Params {
						if pathParamName == strcase.ToLowerCamel(p.Var.Name) {
							methodOpt.Params[i].HTTPType = PathHTTPType
							methodOpt.Params[i].PathParamIndex = idx
							methodOpt.Params[i].PathParamName = pathParamName
						}
					}
				}
			}

			for _, param := range methodOpt.Params {
				if param.Var.IsContext {
					continue
				}
				switch param.HTTPType {
				default:
					methodOpt.BodyParams = append(methodOpt.BodyParams, param)
				case QueryHTTPType:
					methodOpt.QueryParams = append(methodOpt.QueryParams, param)
				case HeaderHTTPType:
					methodOpt.HeaderParams = append(methodOpt.HeaderParams, param)
				case CookieHTTPType:
					methodOpt.CookieParams = append(methodOpt.CookieParams, param)
				case PathHTTPType:
					methodOpt.PathParams = append(methodOpt.PathParams, param)
				}
			}

			if len(m.Results) == 0 || !m.Results[len(m.Results)-1].IsError {
				errs = multierror.Append(errs, gomosaic.Error("Не верная сигнатура метода, последим параметром результата обязателен тип error", m.Pos))
				continue
			}

			for _, result := range m.Results {
				MethodResultOpt := &MethodResultOpt{Var: result}
				err := option.Unmarshal(prefix, result.Annotations, MethodResultOpt)
				if err != nil {
					errs = multierror.Append(errs, err)
				}

				if result.IsError {
					methodOpt.Error = result
				} else {
					switch MethodResultOpt.HTTPType {
					default:
						methodOpt.BodyResults = append(methodOpt.BodyResults, MethodResultOpt)
					case HeaderHTTPType:
						methodOpt.HeaderResults = append(methodOpt.HeaderResults, MethodResultOpt)
					case CookieHTTPType:
						methodOpt.CookieResults = append(methodOpt.CookieResults, MethodResultOpt)
					}
				}

				MethodResultOpt.Name = formatName(MethodResultOpt.NameOpt.Value, MethodResultOpt.Var.Name, MethodResultOpt.NameOpt.Format)

				methodOpt.Results = append(methodOpt.Results, MethodResultOpt)
			}

			ifaceOpt.Methods = append(ifaceOpt.Methods, methodOpt)
		}

		for i, e := range ifaceOpt.Errors {
			components := strings.Split(e.Expr, " ")
			if len(components) != 2 && (e.StatusCode && len(components) != 1) {
				errs = multierror.Append(errs, gomosaic.Error("не верный формат значения http-error", nameTypeInfo.Pos))
				continue
			}

			if e.Type == "" {
				ifaceOpt.Errors[i].Type = "string"
			}

			var tagName, methodName, fldName string
			if e.StatusCode {
				tagName = "-"
				methodName = components[0]
				fldName = strcase.ToCamel(methodName)
			} else {
				tagName = components[0]
				methodName = components[1]
				fldName = strcase.ToCamel(methodName)
			}

			ifaceOpt.Errors[i].MethodName = methodName
			ifaceOpt.Errors[i].FldName = fldName
			ifaceOpt.Errors[i].TagName = tagName
		}

		interfaces = append(interfaces, ifaceOpt)
	}
	if errs != nil {
		return nil, errs
	}
	return interfaces, nil
}
