package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

const (
	prometheusPkg = "github.com/prometheus/client_golang/prometheus"
	timePkg       = "time"
	strconvPkg    = "strconv"

	labelMethod = "method"
	labelPath   = "path"
	labelStatus = "status"

	statusServerError = 500
)

// defaultBuckets задают границы гистограммы длительности запросов (в секундах):
// от 5 мс до 10 с, что покрывает p95/p99 для HTTP-запросов.
const (
	bucket5ms    = 0.005
	bucket10ms   = 0.01
	bucket25ms   = 0.025
	bucket50ms   = 0.05
	bucket100ms  = 0.1
	bucket250ms  = 0.25
	bucket500ms  = 0.5
	bucket1s     = 1
	bucket2500ms = 2.5
	bucket5s     = 5
	bucket10s    = 10
)

var defaultBuckets = []float64{
	bucket5ms, bucket10ms, bucket25ms, bucket50ms, bucket100ms,
	bucket250ms, bucket500ms, bucket1s, bucket2500ms, bucket5s, bucket10s,
}

// generator генерирует общую часть файла метрик: buckets, структуру-хранилище
// векторов, конструктор и метод observe. Различия между роутерами выносятся в
// Strategy.GenerateReturn.
type generator struct {
	f        *gomosaic.GoFile
	strategy Strategy
}

func (g *generator) Generate() jen.Code {
	group := jen.NewFile("")

	buckets := make([]jen.Code, len(defaultBuckets))
	for i, b := range defaultBuckets {
		buckets[i] = jen.Lit(b)
	}

	group.Var().Id(g.strategy.StructName() + "Buckets").Op("=").Index().Float64().Values(buckets...)

	group.Type().Id(g.strategy.StructName()).Struct(
		jen.Id("requests").Op("*").Qual(prometheusPkg, "CounterVec"),
		jen.Id("duration").Op("*").Qual(prometheusPkg, "HistogramVec"),
		jen.Id("errors").Op("*").Qual(prometheusPkg, "CounterVec"),
	)

	if extra := g.strategy.ExtraTypes(g.f); extra != nil {
		group.Add(extra)
	}

	group.Add(g.genConstructor())
	group.Add(g.genObserve())

	return group
}

func (g *generator) genConstructor() jen.Code {
	group := jen.NewFile("")

	group.Commentf(
		"%s creates an HTTP middleware that records request metrics for Prometheus.",
		g.strategy.ConstructorName(),
	)

	group.Func().Id(g.strategy.ConstructorName()).Params(
		jen.Id("reg").Qual(prometheusPkg, "Registerer"),
		jen.Id("namespace").String(),
		jen.Id("constLabels").Qual(prometheusPkg, "Labels"),
	).Params(g.strategy.MiddlewareType(g.f)).Block(
		jen.Id("m").Op(":=").Op("&").Id(g.strategy.StructName()).Values(jen.Dict{
			jen.Id("requests"): g.genRequests(),
			jen.Id("duration"): g.genDuration(),
			jen.Id("errors"):   g.genErrors(),
		}),
		jen.Id("reg").Dot("MustRegister").Call(
			jen.Id("m").Dot("requests"),
			jen.Id("m").Dot("duration"),
			jen.Id("m").Dot("errors"),
		),
		g.strategy.GenerateReturn(g.f),
	)

	return group
}

func (g *generator) genRequests() jen.Code {
	return jen.Qual(prometheusPkg, "NewCounterVec").Call(
		jen.Qual(prometheusPkg, "CounterOpts").Values(jen.Dict{
			jen.Id("Namespace"):   jen.Id("namespace"),
			jen.Id("Name"):        jen.Lit("http_requests_total"),
			jen.Id("Help"):        jen.Lit("Total number of HTTP requests."),
			jen.Id("ConstLabels"): jen.Id("constLabels"),
		}),
		jen.Index().String().Values(jen.Lit(labelMethod), jen.Lit(labelPath), jen.Lit(labelStatus)),
	)
}

func (g *generator) genDuration() jen.Code {
	return jen.Qual(prometheusPkg, "NewHistogramVec").Call(
		jen.Qual(prometheusPkg, "HistogramOpts").Values(jen.Dict{
			jen.Id("Namespace"):   jen.Id("namespace"),
			jen.Id("Name"):        jen.Lit("http_request_duration_seconds"),
			jen.Id("Help"):        jen.Lit("Duration of HTTP request processing in seconds."),
			jen.Id("Buckets"):     jen.Id(g.strategy.StructName() + "Buckets"),
			jen.Id("ConstLabels"): jen.Id("constLabels"),
		}),
		jen.Index().String().Values(jen.Lit(labelMethod), jen.Lit(labelPath), jen.Lit(labelStatus)),
	)
}

func (g *generator) genErrors() jen.Code {
	return jen.Qual(prometheusPkg, "NewCounterVec").Call(
		jen.Qual(prometheusPkg, "CounterOpts").Values(jen.Dict{
			jen.Id("Namespace"):   jen.Id("namespace"),
			jen.Id("Name"):        jen.Lit("http_errors_total"),
			jen.Id("Help"):        jen.Lit("Total number of HTTP responses with an error (status >= 500)."),
			jen.Id("ConstLabels"): jen.Id("constLabels"),
		}),
		jen.Index().String().Values(jen.Lit(labelMethod), jen.Lit(labelPath)),
	)
}

func (g *generator) genObserve() jen.Code {
	group := jen.NewFile("")

	group.Func().Params(
		jen.Id("m").Op("*").Id(g.strategy.StructName()),
	).Id("observe").Params(
		jen.Id("method").String(),
		jen.Id("path").String(),
		jen.Id("status").Int(),
		jen.Id("d").Qual(timePkg, "Duration"),
	).Block(
		jen.Id("statusStr").Op(":=").Qual(strconvPkg, "Itoa").Call(jen.Id("status")),
		jen.Id("m").Dot("requests").Dot("WithLabelValues").Call(
			jen.Id("method"), jen.Id("path"), jen.Id("statusStr"),
		).Dot("Inc").Call(),
		jen.Id("m").Dot("duration").Dot("WithLabelValues").Call(
			jen.Id("method"), jen.Id("path"), jen.Id("statusStr"),
		).Dot("Observe").Call(jen.Id("d").Dot("Seconds").Call()),
		jen.If(jen.Id("status").Op(">=").Lit(statusServerError)).Block(
			jen.Id("m").Dot("errors").Dot("WithLabelValues").Call(
				jen.Id("method"), jen.Id("path"),
			).Dot("Inc").Call(),
		),
	)

	return group
}
