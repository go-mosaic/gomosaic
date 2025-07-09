package annotation

import (
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/option"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

const defaultMemory = 32 << 20 // 32 MB

type UseOpt struct {
	// @docgen-title "Включить обработку запросов multipart/form-data"
	// @docgen-descr "Включает поддержу запросов в фрмате multipart"
	Multipart bool `option:"multipart"`
	// @docgen-title "Включить обработку запросов application/x-www-form-urlencoded"
	// @docgen-descr "Включает поддержу запросов в фрмате urlencoded"
	URLEncoded bool `option:"url-encoded"`
}

// @docgen
// @docgen-title "Аннотации метода"
type MethodOpt struct {
	// @docgen-title "Формат времени для запроса"
	// @docgen-descr "Устанавлиивает формат времени для JSON запросов"
	// @docgen-example "Базовый пример" "@http-time-format 2006-01-02T15:04:05Z07:00"
	// @docgen-option-descr "формат времени из пакета time"
	TimeFormat string `option:"time-format"`
	// @docgen-title "Метод запроса HTTP"
	// @docgen-option-descr "возможные значения GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH"
	// @docgen-example "Базовый пример" "@http-method GET"
	// @docgen-example "Базовый пример" "@http-method POST"
	Method string `option:"method" valid:"in,params:'GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH'"`
	// @docgen-title "Путь HTTP хендлера"
	// @docgen-descr "Путь HTTP хендлера для обработки запроса или отправки клиента"
	// @docgen-option-descr "HTTP путь, можно использовать именованный парамер, должен совпадать с именем параметра метода"
	// @docgen-example "Базовый пример" "@http-path /user"
	// @docgen-example "Пример с именованым параметром" "@http-path /user/{id}"
	Path    string           `option:"path"`
	Openapi MethodOpenapiOpt `option:"openapi"`
	// @docgen-title "Максимальный размер тела HTTP запроса"
	// @docgen-descr "Задает максимальный размер передаваймых данных для <code>multipart/form-data</code> и <code>application/x-www-form-urlencoded</code>"
	// @docgen-option-descr "Значение в байтах, по умолчанию 32 MB"
	FormMaxMemory int               `option:"form-max-memory"`
	Query         MethodQueryOpt    `option:"query"`
	WrapReq       MethodWrapReqOpt  `option:"wrap-req"`
	WrapResp      MethodWrapRespOpt `option:"wrap-resp"`
	Single        SingleOpt         `option:"single"`
	Default       DefaultOpt        `option:"default"`
	Use           UseOpt            `option:"use"`

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

type SingleOpt struct {
	// @docgen-title "Включает оборачивание тела запроса"
	// @docgen-descr "Если аннтотация установлена и в методе есть только один входящий параметр генератор его обернет в JSON вида <code>{\"paramNme\": paramValue}</code>"
	Req bool `option:"req,asFlag"`
	// @docgen-title "Включает оборачивание тела ответа"
	// @docgen-descr "Если аннтотация установлена и метод возвращает только одно значение генератор его обернет в JSON вида <code>{\"paramNme\": paramValue}</code>"
	Resp bool `option:"resp,asFlag"`
}

type MethodWrapReqOpt struct {
	// @docgen-title "Оборачивание тела запроса"
	// @docgen-descr "Позволяет обернуть JSON запрос во вложенный объект, например если необходимо обернуть запрос, чтобы получилось <code>{\"response\": {\"data\": {\"name\": \"test\"}}}</code> надо указать параметр path: <code>response.data</code>"
	// @docgen-option "path" "Путь через точку в который необходимо обернуть тело запроса"
	Path      string `option:"path"`
	PathParts []string
}

type MethodWrapRespOpt struct {
	// @docgen-title "Оборачивание тела ответа"
	// @docgen-option "path" "Путь через точку в который необходимо обернуть тело ответа"
	// @docgen-example "Базовый пример" "@http-wrap-req 'data.user'"
	Path      string `option:"path"`
	PathParts []string
}

type MethodXMLOpt struct {
	ReqName  string `option:"req-name"`
	RespName string `option:"resp-name"`
}

type MethodQueryOpt struct {
	// @docgen-title "Значение в query"
	// @docgen-descr "Используеться для передачи фиксированного значения в query, используеться только при генерации HTTP клиента"
	// @docgen-example "Базовый пример" "@http-query-value perpage 10"
	Values []QueryValueOpt `option:"value,inline"`
}

type QueryValueOpt struct {
	// @docgen-option-descr "Имя параметра"
	Name string `option:",fromValue" valid:"required"`
	// @docgen-option-descr "Значение параметра"
	Values []string `option:",fromOptions"`
}

type MethodOpenapiOpt struct {
	// @docgen-title "Теги OpenAPI"
	// @docgen-option-descr "Устанавливает теги для кнечной точки при генерации openapi документаци"
	// @docgen-example "Базовый пример" "@openapi-tags tag1 tag2 tag3 tag4"
	Tags []string `option:"tags"`
}

type MethodParamNameOpt struct {
	// @docgen-option-descr "Имя параметра в запросе в зависимости от типа (загловок, тело запроса)"
	Value     string `option:",fromValue"`
	Omitempty bool   `option:",fromOption,asFlag"`
	// @docgen-option-descr "Формат"
	Format string `option:",fromParam" valid:"in,params:'lowerCamel kebab screamingKebab snake screamingSnake'" default:"lowerCamel"`
}

// @docgen
// @docgen-title "Аннотации параметров метода"
// @docgen-descr "Аннотации параметров метода применяются только для параметров метода"
type MethodParamOpt struct {
	// @docgen-title "Имя параметра в запросе"
	// @docgen-descr "Имя параметра в запросе в зависимости от типа (загловок, тело запроса)"
	NameOpt MethodParamNameOpt `option:"name,inline"`
	// @docgen-title "Тип передачи значения параметра"
	// @docgen-descr "Тип передачи значения параметра определяет как параметр будет передаваться в запросе"
	// @docgen-option "type" "Тип парамера, возможные значения <code>body</code>, <code>header</code>, <code>query</code>, <code>cookie</code>"
	HTTPType string `option:"type"`
	// @docgen-title "Утсановка как обязательного"
	// @docgen-descr "На данных момент используется для генераци клиента и опередляет какие параметры необходимо указывать в сгенерированном методе клиента обязательно"
	Required bool `option:"required,asFlag"`

	Var            *gomosaic.VarInfo
	Name           string
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

type DefaultOpt struct {
	// @docgen-title "Content-Type по умолчанию"
	// @docgen-descr "Устанавливает Content-Type по умолчанию для запроса в котором его не передали"
	// @docgen-option-descr "Значение типа контента например application/json"
	ContentType string `option:"content-type"`
	// @docgen-title "Accept по умолчанию"
	// @docgen-descr "Устанавливает Accept по умолчанию для запроса в котором его не передали"
	// @docgen-option-descr "Значение типа контента например application/json"
	Accept string `option:"accept"`
}

// @docgen
// @docgen-title "Аннотации интерфейса"
type IfaceOpt struct {
	Default DefaultOpt `option:"default"`
	// @docgen-title "Включение копирования типов в сгенерированного клиента"
	CopyDTOTypes bool `option:"copy-dto-types,asFlag"`

	NameTypeInfo *gomosaic.NameTypeInfo
	Methods      []*MethodOpt
}

func Load(module *gomosaic.ModuleInfo, prefix string, types []*gomosaic.NameTypeInfo) (interfaces []*IfaceOpt, errs error) {
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
							methodOpt.Params[i].Required = true
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

		// for i, e := range ifaceOpt.Errors {
		// 	components := strings.Split(e.Expr, " ")
		// 	if len(components) != 2 && (e.StatusCode && len(components) != 1) {
		// 		errs = multierror.Append(errs, gomosaic.Error("не верный формат значения http-error", nameTypeInfo.Pos))
		// 		continue
		// 	}

		// 	if e.Type == "" {
		// 		ifaceOpt.Errors[i].Type = "string"
		// 	}

		// 	var tagName, methodName, fldName string
		// 	if e.StatusCode {
		// 		tagName = "-"
		// 		methodName = components[0]
		// 		fldName = strcase.ToCamel(methodName)
		// 	} else {
		// 		tagName = components[0]
		// 		methodName = components[1]
		// 		fldName = strcase.ToCamel(methodName)
		// 	}

		// 	ifaceOpt.Errors[i].MethodName = methodName
		// 	ifaceOpt.Errors[i].FldName = fldName
		// 	ifaceOpt.Errors[i].TagName = tagName
		// }

		interfaces = append(interfaces, ifaceOpt)
	}
	if errs != nil {
		return nil, errs
	}
	return interfaces, nil
}
