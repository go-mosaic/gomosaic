# Миграция с gomosaic v1 на v2

## Обзор изменений

Версия v2 представляет значительные архитектурные улучшения, сохраняя совместимость
на уровне генерируемого кода. Основные изменения касаются способа регистрации плагинов,
управления трансформерами и сборки генератора.

## Ключевые отличия

| Аспект | v1 | v2 |
|--------|----|----|
| Регистрация плагинов | `init()` + blank imports | Явная через `Builder.WithPlugin()` |
| PluginManager | Глобальный `DefaultPluginManager` | Изолированный `PluginRegistry` |
| Трансформеры | Глобальные `parseFactories`/`formatFactories` | `TransformRegistry` (изолированный) |
| Сборка генератора | Прямое создание `CodeGenerator` | `Builder` API |
| Middleware-плагины | Отдельные: `logmiddleware`, `metricmiddleware` | Единый: `middleware.NewPlugin(cfg)` |
| CLI | Жёсткая привязка через импорты | `cmd.Run(builder)` — передача Builder |

## Пошаговая миграция

### 1. Обновление go.mod

```bash
# Было (v1)
module github.com/myproject

require github.com/go-mosaic/gomosaic v1.x.x

# Стало (v2)
module github.com/myproject

require github.com/go-mosaic/gomosaic/v2 v2.0.0
```

Обновите импорты:
```go
// Было
import "github.com/go-mosaic/gomosaic/pkg/gomosaic"

// Стало
import "github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
```

### 2. Создание main.go (было → стало)

**v1 — жёсткая привязка через init():**
```go
package main

import (
    "github.com/go-mosaic/gomosaic/pkg/cmd"
    
    // Плагины регистрируются автоматически через init()
    _ "github.com/go-mosaic/gomosaic/internal/plugin/http"
    _ "github.com/go-mosaic/gomosaic/internal/plugin/logmiddleware"
    _ "github.com/go-mosaic/gomosaic/internal/plugin/metricmiddleware"
)

func main() {
    cmd.Run("1.0.0")
}
```

**v2 — явная сборка через Builder:**
```go
package main

import (
    "github.com/go-mosaic/gomosaic/v2/pkg/cmd"
    "github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
    "github.com/go-mosaic/gomosaic/v2/plugins/http/client"
    "github.com/go-mosaic/gomosaic/v2/plugins/http/server"
    "github.com/go-mosaic/gomosaic/v2/plugins/middleware"
)

func main() {
    // Явная сборка генератора
    builder := gomosaic.NewBuilder("myapp", "1.0.0", "./output")

    // Регистрируем только нужные плагины
    builder.WithPlugins(
        server.NewPlugin(&server.ChiStrategy{}),
        server.NewPlugin(&server.EchoStrategy{}),
        client.NewPlugin(),
        
        // Middleware-плагины конфигурируются
        middleware.NewPlugin(middleware.Config{
            Name:           "log-middleware",
            MiddlewareName: "Log",
            Annotation:     "log",
            Fields: []jen.Code{
                jen.Id("logger"), jen.Qual(gomosaic.SpanPkg, "Logger"),
            },
            BeforeFn: func(g *jen.Group) {
                g.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartLogSpan").Call(
                    jen.Id("m").Dot("logger"),
                    jen.Id("ctx"),
                )
            },
            AfterFn: func(g *jen.Group) {
                g.Id("span").Dot("Finish").Call(jen.Id("ctx"))
            },
        }),
        
        middleware.NewPlugin(middleware.Config{
            Name:           "metric-middleware",
            MiddlewareName: "Metric",
            Annotation:     "metric",
            Fields: []jen.Code{
                jen.Id("collector"), jen.Qual(gomosaic.SpanPkg, "MetricsCollector"),
            },
            BeforeFn: func(g *jen.Group) {
                g.Id("span").Op(":=").Qual(gomosaic.SpanPkg, "StartMetricSpan").Call(
                    jen.Id("m").Dot("collector"),
                    jen.Id("ctx"),
                )
            },
            AfterFn: func(g *jen.Group) {
                g.Id("span").Dot("Finish").Call(jen.Id("ctx"))
            },
        }),
    )

    // Запуск CLI с собранным генератором
    cmd.Run(builder)
}
```

### 3. Миграция плагинов

**v1 — отдельный тип для каждого middleware:**
```go
// internal/plugin/logmiddleware/plugin.go
type Plugin struct{}

func (p *Plugin) Name() string { return "log-middleware" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
    // ~80 строк кода, специфичного для log
}

// internal/plugin/metricmiddleware/plugin.go
type Plugin struct{}

func (p *Plugin) Name() string { return "metric-middleware" }

func (p *Plugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
    // ~80 строк кода, практически идентичного log
}
```

**v2 — единый плагин, конфигурируемый через Config:**
```go
// Оба плагина создаются из одного типа!
logPlugin := middleware.NewPlugin(middleware.Config{
    Name:           "log-middleware",
    MiddlewareName: "Log",
    Annotation:     "log",
    Fields:         []jen.Code{...},
    BeforeFn:       func(g *jen.Group) { /* логика log */ },
    AfterFn:        func(g *jen.Group) { /* логика log */ },
})

metricPlugin := middleware.NewPlugin(middleware.Config{
    Name:           "metric-middleware",
    MiddlewareName: "Metric",
    Annotation:     "metric",
    Fields:         []jen.Code{...},
    BeforeFn:       func(g *jen.Group) { /* логика metric */ },
    AfterFn:        func(g *jen.Group) { /* логика metric */ },
})
```

### 4. Миграция серверных плагинов

**v1 — отдельные типы:**
```go
type PluginServerChi struct{}
type PluginServerEcho struct{}
```

**v2 — единый тип + стратегия:**
```go
chiPlugin  := server.NewPlugin(&server.ChiStrategy{})
echoPlugin := server.NewPlugin(&server.EchoStrategy{})
```

### 5. Кастомные трансформеры

**v1 — глобальная регистрация:**
```go
import "github.com/go-mosaic/gomosaic/pkg/typetransform"

func init() {
    typetransform.AddTransformer(func() typetransform.Transformer {
        return &MyTransformer{}
    })
}
```

**v2 — регистрация через Builder:**
```go
builder.WithTransformer(func() gomosaic.Transformer {
    return &MyTransformer{}
})

// Или через кастомный реестр:
reg := gomosaic.NewTransformRegistry()
reg.AddTransformer(myTransformFactory)
builder := gomosaic.NewBuilder("app", "1.0", "./out",
    gomosaic.WithTransformRegistry(reg),
)
```

### 6. Пайплайн генерации (новое в v2)

```go
import "github.com/go-mosaic/gomosaic/v2/pkg/gomosaic/pipeline"

// Создаём пайплайн с промежуточными обработчиками
pl := pipeline.NewPipeline(
    pipeline.NewPluginAdapter(myPlugin),
)

// Добавляем middleware
handler := pipeline.Chain(
    pipeline.LoggingMiddleware(log.Default()),
    pipeline.TimingMiddleware(),
    pipeline.RecoveryMiddleware(),
)(func(ctx context.Context, data *pipeline.StageData) (*pipeline.StageData, error) {
    return pl.Run(ctx, data)
})

// Запускаем
result, err := handler(ctx, pipeline.NewStageData(module, types))
```

### 7. Программное использование (новое в v2)

```go
// Генерация без CLI
builder := gomosaic.NewBuilder("myapp", "1.0.0", "./output")
builder.WithPlugin(myPlugin)

gen := builder.Build()

ctx := gomosaic.ContextWithOutputDir(context.Background(), "./output")
files, err := gen.Generate(ctx, moduleInfo, typesInfo, "my-plugin")
```

## Совместимость

- **Генерируемый код** — полностью совместим. Структура выходных файлов не изменилась.
- **Аннотации** — полностью совместимы. Синтаксис `@http-method`, `@http-path` и др. не изменился.
- **Интерфейсы** — не изменились. `Generator`, `File`, `TypeInfo` работают так же.
- **Runtime** — не изменился. Используется тот же `github.com/go-mosaic/runtime`.

## Часто задаваемые вопросы

### Нужно ли менять аннотации в коде?
Нет. Все аннотации (`@http-method`, `@http-path`, `@log`, `@metric` и т.д.)
работают без изменений.

### Могу ли я использовать v1 и v2 одновременно?
Да. v2 — отдельный модуль (`github.com/go-mosaic/gomosaic/v2`), который может
сосуществовать с v1 в одном проекте.

### Как добавить свой HTTP-роутер?
Реализуйте интерфейс `server.Strategy`:
```go
type MyRouterStrategy struct{}

func (s *MyRouterStrategy) Name() string                 { return "http-server-myrouter" }
func (s *MyRouterStrategy) UsePtrType() bool              { return true }
func (s *MyRouterStrategy) RouterType() string            { return "MyRouter" }
func (s *MyRouterStrategy) RouterPkg() string             { return "github.com/my/router" }
func (s *MyRouterStrategy) TransportFactoryType() string  { return "TransportTypeMyRouter" }
func (s *MyRouterStrategy) PathParamWrap(n string) string { return "{" + n + "}" }
```
