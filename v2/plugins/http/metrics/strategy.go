// Package metrics предоставляет плагины-генераторы HTTP-middleware,
// которая собирает метрики запросов (RPS, длительность, ошибки) в Prometheus.
//
// В отличие от method-level middleware из `plugins/middleware`, эта мидлвара
// вешается на сам роутер (chi/echo/fiber/net/http) и поэтому видит реальные
// HTTP-запросы, включая отклонённые на авторизации или не прошедшие парсинг.
//
// Пример использования:
//
//	builder.WithPlugins(
//	    metrics.NewPlugin(&metrics.ChiStrategy{}),
//	    metrics.NewPlugin(&metrics.MuxStrategy{}),
//	)
package metrics

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Strategy определяет роутер, под который генерируется мидлвара.
type Strategy interface {
	// Name возвращает имя плагина, например "http-metrics-chi".
	Name() string

	// FileName возвращает имя сгенерированного файла, например "metrics_chi_gen.go".
	FileName() string

	// ConstructorName возвращает имя конструктора, например "NewChiMetricsMiddleware".
	ConstructorName() string

	// StructName возвращает имя типа-хранилища метрик, например "chiMetrics".
	StructName() string

	// MiddlewareType возвращает тип возвращаемого middleware.
	MiddlewareType(f *gomosaic.GoFile) jen.Code

	// ExtraTypes возвращает дополнительные объявления, необходимые стратегии
	// (например, обёртку над http.ResponseWriter). Может быть nil.
	ExtraTypes(f *gomosaic.GoFile) jen.Code

	// GenerateReturn генерирует `return <middleware>` внутри конструктора.
	GenerateReturn(f *gomosaic.GoFile) jen.Code
}
