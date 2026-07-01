// Package annotation предоставляет загрузку и парсинг HTTP-аннотаций для v2.
package annotation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/option"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

// Константы HTTP-типов параметров.
const (
	PathHTTPType   = "path"
	CookieHTTPType = "cookie"
	QueryHTTPType  = "query"
	HeaderHTTPType = "header"
	BodyHTTPType   = "body"
)

const defaultMemory = 32 << 20 // 32 MB

// IfaceOpt содержит аннотации интерфейса.
type IfaceOpt struct {
	Default      DefaultOpt `option:"default"`
	CopyTypes    bool       `option:"copy-types,asFlag"`
	ClientEnable bool       `option:"client-enable,asFlag"`

	NameTypeInfo *gomosaic.NameTypeInfo
	Methods      []*MethodOpt
}

// DefaultOpt — опции по умолчанию.
type DefaultOpt struct {
	ContentType string `option:"content-type"`
	Accept      string `option:"accept"`
}

// MethodOpt содержит аннотации метода.
type MethodOpt struct {
	Method        string        `option:"method" valid:"in,params:'GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH'"`
	Path          string        `option:"path"`
	TimeFormat    string        `option:"time-format"`
	FormMaxMemory int           `option:"form-max-memory"`
	WrapReq       MethodWrapOpt `option:"wrap-req"`
	WrapResp      MethodWrapOpt `option:"wrap-resp"`
	Single        SingleOpt     `option:"single"`
	Use           UseOpt        `option:"use"`
	Default       DefaultOpt    `option:"default"`

	// Вычисляемые поля
	Iface         *IfaceOpt
	Func          *gomosaic.MethodInfo
	Context       *gomosaic.VarInfo
	Error         *gomosaic.VarInfo
	Params        []*MethodParamOpt
	Results       []*MethodResultOpt
	BodyParams    []*MethodParamOpt
	QueryParams   []*MethodParamOpt
	HeaderParams  []*MethodParamOpt
	CookieParams  []*MethodParamOpt
	PathParams    []*MethodParamOpt
	BodyResults   []*MethodResultOpt
	HeaderResults []*MethodResultOpt
	CookieResults []*MethodResultOpt
}

// MethodWrapOpt — опции оборачивания.
type MethodWrapOpt struct {
	Path      string   `option:"path"`
	PathParts []string // вычисляется из Path
}

// SingleOpt — опции одиночного параметра.
type SingleOpt struct {
	Req  bool `option:"req,asFlag"`
	Resp bool `option:"resp,asFlag"`
}

// UseOpt — опции использования.
type UseOpt struct {
	Multipart  bool `option:"multipart"`
	URLEncoded bool `option:"url-encoded"`
}

// MethodParamOpt содержит аннотации параметра метода.
type MethodParamOpt struct {
	NameOpt  MethodParamNameOpt `option:"name,inline"`
	HTTPType string             `option:"type"`
	Required bool               `option:"required,asFlag"`

	Var            *gomosaic.VarInfo
	Name           string
	PathParamIndex int
	PathParamName  string
}

// MethodParamNameOpt — опции имени параметра.
type MethodParamNameOpt struct {
	Value     string `option:",fromValue"`
	Omitempty bool   `option:",fromOption,asFlag"`
	Format    string `option:",fromParam" valid:"in,params:'lowerCamel kebab screamingKebab snake screamingSnake'" default:"lowerCamel"`
}

// MethodResultOpt содержит аннотации результата метода.
type MethodResultOpt struct {
	Var      *gomosaic.VarInfo
	Name     string
	NameOpt  MethodParamNameOpt `option:"name,inline"`
	HTTPType string             `option:"type"`
	Required bool               `option:"required"`
	Flat     bool               `option:"flat"`
}

// Load загружает HTTP-аннотации из типов.
func Load(module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (interfaces []*IfaceOpt, errs error) {
	for _, nameTypeInfo := range types {
		if nameTypeInfo.Type.Interface == nil {
			continue
		}

		ifaceOpt := &IfaceOpt{NameTypeInfo: nameTypeInfo}

		if err := option.Unmarshal("http", nameTypeInfo.Annotations, ifaceOpt); err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		for _, m := range nameTypeInfo.Type.Interface.Methods {
			methodOpt := &MethodOpt{Iface: ifaceOpt, Func: m}

			if err := option.Unmarshal("http", m.Annotations, methodOpt); err != nil {
				errs = multierror.Append(errs, err)
			}

			if methodOpt.FormMaxMemory == 0 {
				methodOpt.FormMaxMemory = defaultMemory
			}
			if methodOpt.Default.Accept == "" {
				methodOpt.Default.Accept = ifaceOpt.Default.Accept
			}
			if methodOpt.Default.ContentType == "" {
				methodOpt.Default.ContentType = ifaceOpt.Default.ContentType
			}
			if methodOpt.WrapReq.Path != "" {
				methodOpt.WrapReq.PathParts = strings.Split(methodOpt.WrapReq.Path, ".")
			}
			if methodOpt.WrapResp.Path != "" {
				methodOpt.WrapResp.PathParts = strings.Split(methodOpt.WrapResp.Path, ".")
			}

			if len(m.Params) == 0 || !m.Params[0].IsContext {
				errs = multierror.Append(errs, fmt.Errorf("не верная сигнатура метода %s: первый параметр должен быть context.Context", m.Name))
				continue
			}

			// Парсим параметры
			for _, param := range m.Params {
				mp := &MethodParamOpt{Var: param}
				if err := option.Unmarshal("http", param.Annotations, mp); err != nil {
					errs = multierror.Append(errs, err)
				}
				if param.IsContext {
					methodOpt.Context = param
				}
				if mp.HTTPType == "" {
					mp.HTTPType = BodyHTTPType
				}
				mp.Name = formatName(mp.NameOpt.Value, mp.Var.Name, mp.NameOpt.Format)
				methodOpt.Params = append(methodOpt.Params, mp)
			}

			// Парсим path params (поддержка :param и {param})
			parts := strings.Split(methodOpt.Path, "/")
			for idx, part := range parts {
				pathParamName := ""
				if strings.HasPrefix(part, ":") {
					pathParamName = part[1:]
				} else if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
					pathParamName = part[1 : len(part)-1]
				}
				if pathParamName != "" {
					for i, p := range methodOpt.Params {
						if pathParamName == strcase.ToLowerCamel(p.Var.Name) {
							methodOpt.Params[i].HTTPType = PathHTTPType
							methodOpt.Params[i].Required = true
							methodOpt.Params[i].PathParamIndex = idx
							methodOpt.Params[i].PathParamName = pathParamName
						}
					}
				}
			}

			// Категоризация параметров
			for _, p := range methodOpt.Params {
				if p.Var.IsContext {
					continue
				}
				switch p.HTTPType {
				case QueryHTTPType:
					methodOpt.QueryParams = append(methodOpt.QueryParams, p)
				case HeaderHTTPType:
					methodOpt.HeaderParams = append(methodOpt.HeaderParams, p)
				case CookieHTTPType:
					methodOpt.CookieParams = append(methodOpt.CookieParams, p)
				case PathHTTPType:
					methodOpt.PathParams = append(methodOpt.PathParams, p)
				default:
					methodOpt.BodyParams = append(methodOpt.BodyParams, p)
				}
			}

			// Результаты
			if len(m.Results) == 0 || !m.Results[len(m.Results)-1].IsError {
				errs = multierror.Append(errs, fmt.Errorf("не верная сигнатура метода %s: последний результат должен быть error", m.Name))
				continue
			}

			for _, result := range m.Results {
				mr := &MethodResultOpt{Var: result}
				if err := option.Unmarshal("http", result.Annotations, mr); err != nil {
					errs = multierror.Append(errs, err)
				}
				if result.IsError {
					methodOpt.Error = result
				} else {
					switch mr.HTTPType {
					case HeaderHTTPType:
						methodOpt.HeaderResults = append(methodOpt.HeaderResults, mr)
					case CookieHTTPType:
						methodOpt.CookieResults = append(methodOpt.CookieResults, mr)
					default:
						methodOpt.BodyResults = append(methodOpt.BodyResults, mr)
					}
				}
				mr.Name = formatName(mr.NameOpt.Value, mr.Var.Name, mr.NameOpt.Format)
				methodOpt.Results = append(methodOpt.Results, mr)
			}

			ifaceOpt.Methods = append(ifaceOpt.Methods, methodOpt)
		}

		interfaces = append(interfaces, ifaceOpt)
	}

	if errs != nil {
		return nil, errs
	}
	return interfaces, nil
}

var paramNameFormatters = map[string]func(string) string{
	"lowerCamel":     strcase.ToLowerCamel,
	"kebab":          strcase.ToKebab,
	"screamingKebab": strcase.ToScreamingKebab,
	"snake":          strcase.ToSnake,
	"screamingSnake": strcase.ToScreamingSnake,
}

const paramNameDefaultFormatter = "lowerCamel"

func formatName(name, defaultName, format string) string {
	if name != "" {
		return name
	}
	if fn, ok := paramNameFormatters[format]; ok {
		return fn(defaultName)
	}
	return paramNameFormatters[paramNameDefaultFormatter](defaultName)
}
