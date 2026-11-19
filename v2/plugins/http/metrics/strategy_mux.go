package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// MuxStrategy генерирует middleware для стандартного net/http ServeMux.
type MuxStrategy struct{}

// Name реализует Strategy.
func (s *MuxStrategy) Name() string { return "http-metrics-mux" }

// FileName реализует Strategy.
func (s *MuxStrategy) FileName() string { return "metrics_mux_gen.go" }

// ConstructorName реализует Strategy.
func (s *MuxStrategy) ConstructorName() string { return "NewMuxMetricsMiddleware" }

// StructName реализует Strategy.
func (s *MuxStrategy) StructName() string { return "muxMetrics" }

func (s *MuxStrategy) recorderName() string { return s.StructName() + "StatusRecorder" }

// MiddlewareType реализует Strategy.
func (s *MuxStrategy) MiddlewareType(_ *gomosaic.GoFile) jen.Code {
	return jen.Func().Params(jen.Qual(netHTTPPkg, "Handler")).Qual(netHTTPPkg, "Handler")
}

// ExtraTypes реализует Strategy.
func (s *MuxStrategy) ExtraTypes(_ *gomosaic.GoFile) jen.Code {
	group := jen.NewFile("")
	recName := s.recorderName()

	group.Type().Id(recName).Struct(
		jen.Qual(netHTTPPkg, "ResponseWriter"),
		jen.Id("statusCode").Int(),
	)

	group.Func().Params(
		jen.Id("r").Op("*").Id(recName),
	).Id("WriteHeader").Params(jen.Id("code").Int()).Block(
		jen.Id("r").Dot("statusCode").Op("=").Id("code"),
		jen.Id("r").Dot("ResponseWriter").Dot("WriteHeader").Call(jen.Id("code")),
	)

	return group
}

// GenerateReturn реализует Strategy.
func (s *MuxStrategy) GenerateReturn(_ *gomosaic.GoFile) jen.Code {
	recName := s.recorderName()

	return jen.Return(
		jen.Func().Params(
			jen.Id("next").Qual(netHTTPPkg, "Handler"),
		).Qual(netHTTPPkg, "Handler").Block(
			jen.Return(jen.Qual(netHTTPPkg, "HandlerFunc").Call(
				jen.Func().Params(
					jen.Id("w").Qual(netHTTPPkg, "ResponseWriter"),
					jen.Id("r").Op("*").Qual(netHTTPPkg, "Request"),
				).Block(
					jen.Id("start").Op(":=").Qual(timePkg, "Now").Call(),
					jen.Id("rec").Op(":=").Op("&").Id(recName).Values(jen.Dict{
						jen.Id("ResponseWriter"): jen.Id("w"),
						jen.Id("statusCode"):     jen.Qual(netHTTPPkg, "StatusOK"),
					}),
					jen.Id("next").Dot("ServeHTTP").Call(jen.Id("rec"), jen.Id("r")),
					jen.Id("path").Op(":=").Id("r").Dot("Pattern"),
					jen.If(jen.Id("path").Op("==").Lit("")).Block(
						jen.Id("path").Op("=").Id("r").Dot("URL").Dot("Path"),
					),
					jen.Id("m").Dot("observe").Call(
						jen.Id("r").Dot("Method"),
						jen.Id("path"),
						jen.Id("rec").Dot("statusCode"),
						jen.Qual(timePkg, "Since").Call(jen.Id("start")),
					),
				),
			)),
		),
	)
}

var _ Strategy = (*MuxStrategy)(nil)
