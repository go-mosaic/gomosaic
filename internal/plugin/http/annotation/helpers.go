package annotation

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/pkg/strcase"
)

const (
	CTXPkg           = "context"
	HTTPPkg          = "net/http"
	URLPkg           = "net/url"
	StringsPkg       = "strings"
	BytesPkg         = "bytes"
	JSONPkg          = "encoding/json"
	ContextPkg       = "context"
	FmtPkg           = "fmt"
	SyncPkg          = "sync"
	IOPkg            = "io"
	NetPkg           = "net"
	TLSPkg           = "crypto/tls"
	StrconvPkg       = "strconv"
	MimeMultipartPkg = "mime/multipart"

	PrometheusPkg      = "github.com/prometheus/client_golang/prometheus"
	PromautoPkg        = "github.com/prometheus/client_golang/prometheus/promauto"
	PromHTTPPkg        = "github.com/prometheus/client_golang/prometheus/promhttp"
	CleanHTTPPkg       = "github.com/hashicorp/go-cleanhttp"
	ChiPkg             = "github.com/go-chi/chi/v5"
	JSONRPCPkg         = "github.com/555f/jsonrpc"
	EchoPkg            = "github.com/labstack/echo/v4"
	NullPkg            = "gopkg.in/guregu/null.v4"
	MimeheaderPkg      = "github.com/aohorodnyk/mimeheader"
	OtelTracePkg       = "go.opentelemetry.io/otel/trace"
	OtelTraceAttrPkg   = "go.opentelemetry.io/otel/attribute"
	OtelCodesPkg       = "go.opentelemetry.io/otel/codes"
	OtelPropagationPkg = "go.opentelemetry.io/otel/propagation"
	TemplPkg           = "github.com/a-h/templ"
)

const paramNameDefaultFormatter = "lowerCamel"

const (
	PathHTTPType   string = "path"
	CookieHTTPType string = "cookie"
	QueryHTTPType  string = "query"
	HeaderHTTPType string = "header"
	BodyHTTPType   string = "body"
)

var paramNameFormatters = map[string]func(string) string{
	"lowerCamel":     strcase.ToLowerCamel,
	"kebab":          strcase.ToKebab,
	"screamingKebab": strcase.ToScreamingKebab,
	"snake":          strcase.ToSnake,
	"screamingSnake": strcase.ToScreamingSnake,
}

func formatName(name, defaultName string, format string) string {
	if name != "" {
		return name
	}
	_, isNameParamFormatExists := paramNameFormatters[format]
	if format == "" || !isNameParamFormatExists {
		format = paramNameDefaultFormatter
	}
	return paramNameFormatters[format](defaultName)
}

func WrapStruct(names []string, wrappedCode jen.Code) jen.Code {
	code := wrappedCode

	for i := len(names) - 1; i >= 0; i-- {
		code = jen.Id(strcase.ToCamel(names[i])).Struct(code).Tag(map[string]string{"json": names[i]})
	}

	return code
}

func MakeStructFieldsFromParams(params []*MethodParamOpt, qual jenutils.QualFunc) jen.Code {
	structFields := jen.NewFile("")

	for _, param := range params {
		jsonTag := param.Name
		fld := structFields.Id(strcase.ToCamel(param.Var.Name))
		if !param.Required {
			jsonTag += ",omitempty"
		}
		fld.Add(jenutils.TypeInfoQual(param.Var.Type, qual)).Tag(map[string]string{"json": jsonTag})
	}

	return structFields
}

func MakeStructFieldsFromResults(params []*MethodResultOpt, qual jenutils.QualFunc) jen.Code {
	structFields := jen.NewFile("")

	for _, param := range params {
		jsonTag := param.Name
		fld := structFields.Id(strcase.ToCamel(param.Var.Name))
		fld.Add(jenutils.TypeInfoQual(param.Var.Type, qual)).Tag(map[string]string{"json": jsonTag})
	}

	return structFields
}

func IsObjectType(typeInfo *gomosaic.TypeInfo) (ok bool) {
	if typeInfo.IsPtr {
		typeInfo = typeInfo.ElemType
	}

	if typeInfo.IsNamed {
		typeInfo = typeInfo.ElemType
	}

	return typeInfo.Struct != nil || typeInfo.Interface != nil || typeInfo.IsMap
}

func MakeEmptyResults(results []*MethodResultOpt, qualFunc jenutils.QualFunc, addinCodes ...jen.Code) (codes []jen.Code) {
	for _, r := range results {
		codes = append(codes, jenutils.ZeroValue(r.Var.Type, qualFunc))
	}

	return append(codes, addinCodes...)
}

func Dot(parts ...string) jen.Code {
	group := jen.Null()
	for _, p := range parts {
		group.Dot(strcase.ToCamel(p))
	}

	return group
}
