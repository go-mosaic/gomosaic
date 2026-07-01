// Package server предоставляет HTTP-серверный плагин для v2.
//
// В отличие от v1, где PluginServerChi и PluginServerEcho были отдельными типами,
// v2 использует единый ServerPlugin, конфигурируемый через Strategy.
//
// Пример использования:
//
//	chiPlugin := server.NewPlugin(&server.ChiStrategy{})
//	echoPlugin := server.NewPlugin(&server.EchoStrategy{})
//
//	builder.WithPlugin(chiPlugin)
//	builder.WithPlugin(echoPlugin)
package server

// Strategy определяет стратегию HTTP-сервера (Chi, Echo, и т.д.).
type Strategy interface {
	// Name возвращает имя стратегии (часть имени плагина).
	Name() string

	// UsePtrType указывает, используется ли указатель на тип роутера.
	UsePtrType() bool

	// RouterType возвращает имя типа роутера.
	RouterType() string

	// RouterPkg возвращает путь пакета роутера.
	RouterPkg() string

	// TransportFactoryType возвращает тип транспортной фабрики.
	TransportFactoryType() string

	// PathParamWrap оборачивает имя параметра пути в формат роутера.
	PathParamWrap(paramName string) string
}
