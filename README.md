# GoMosaic — генератор кода по Go-интерфейсам и структурам

[![Lint Status](https://github.com/go-mosaic/gomosaic/workflows/golangci-lint/badge.svg)](https://github.com/go-mosaic/gomosaic/actions)
[![Tests](https://github.com/go-mosaic/gomosaic/workflows/tests/badge.svg)](https://github.com/go-mosaic/gomosaic/actions)

GoMosaic читает Go-интерфейсы и структуры, размеченные аннотациями `@gomosaic`,
и генерирует транспортный слой: HTTP-сервер, HTTP-клиент, gRPC-сервер, OpenAPI,
валидацию, загрузку конфигурации из переменных окружения, middleware и хелперы
связывания параметров HTTP запроса.

Актуальная версия — v2 (`github.com/go-mosaic/gomosaic/v2`). Инструкция по переходу
с v1: [руководство по миграции](v2/docs/MIGRATION.md).

## Установка CLI

В примерах ниже используется запуск через `go tool gomosaic`. Если CLI установлен
через `go install`, замените `go tool gomosaic` на `gomosaic`.

### Вариант 1. `go install`

```bash
go install github.com/go-mosaic/gomosaic/v2/cmd/gomosaic@latest
```
### Вариант 2. Как инструмент модуля (`go tool`, Go 1.24+)

```bash
go get -tool github.com/go-mosaic/gomosaic/v2/cmd/gomosaic@latest
```

После этого CLI запускается через `go tool` и всегда использует версию,
зафиксированную в `go.mod`:

```bash
go tool gomosaic codegen http-server-chi ./internal/controller/... ./internal/controller/gen
```

Обновление версии:

```bash
go get -tool github.com/go-mosaic/gomosaic/v2/cmd/gomosaic@latest
go mod tidy
```

Посмотреть список инструментов модуля:

```bash
go tool
```

### Вариант 3. `go:generate`

Добавьте директиву рядом с интерфейсом:

```go
//go:generate go tool gomosaic codegen http-server-chi ./... ./gen
```

и запускайте `go generate ./...`.

## Быстрый старт

### 1. Разметьте интерфейс

```go
package controller

import "context"

// @gomosaic
type UserService interface {
	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context, id int) (user *User, err error)

	// @http-method POST
	// @http-path /user
	// @http-single req
	CreateUser(ctx context.Context, req *CreateUserRequest) (err error)
}

type User struct {
	Name  string
	Email string
}

type CreateUserRequest struct {
	Name  string
	Email string
}
```

### 2. Сгенерируйте код

```bash
go tool gomosaic codegen http-server-chi ./internal/controller/... ./internal/controller/gen
```

### 3. Используйте

В каталоге `./internal/controller/gen` появится файл `server_chi_gen.go`
с функцией `UserServiceRegisterHandlers`.

```go
router := chi.NewRouter()
controller.UserServiceRegisterHandlers(router, svc, nil)
```

Для gRPC, OpenAPI, валидации и конфигурации используются те же принципы:
аннотируется интерфейс или структура, затем запускается нужный плагин.

## Общие правила аннотаций

- `@gomosaic` обязателен для каждого интерфейса и структуры, которые должен видеть
  генератор. Без него тип пропускается.
- Аннотации размещаются в комментариях непосредственно перед типом, методом,
  параметром или полем.
- Формат: `@<префикс>-<опция> [значения...] [ключ=значение...]`.
- Значения с пробелами можно брать в кавычки: `@openapi-tags "users read"`.
- Аннотации уровня интерфейса действуют как значения по умолчанию для методов.
  Аннотации метода их переопределяют.
- Для HTTP-методов первый параметр должен быть `context.Context`, последний
  результат — `error`.
- Имя параметра по умолчанию — имя Go-поля, приведённое к `lowerCamel`.
  `@http-name format=<формат>` меняет это преобразование, а `@http-name <имя>`
  задаёт имя явно (в этом случае `format` не применяется). Доступные форматы:
  `lowerCamel` (по умолчанию), `kebab`, `screamingKebab`, `snake`, `screamingSnake`.

## Плагины

| Плагин | Назначение |
|--------|------------|
| `http-server-chi` | HTTP-сервер для `go-chi/chi` |
| `http-server-echo`| HTTP-сервер для `labstack/echo` |
| `http-client` | Типизированный HTTP-клиент |
| `http-client-test` | Тесты сгенерированного HTTP-клиента |
| `http-bind-std` | `Bind` для `net/http` (`r.PathValue`) |
| `http-bind-chi` | `Bind` для `go-chi/chi` (`chi.URLParam`) |
| `grpc-server` | gRPC-сервер и конвертеры protobuf-типов |
| `openapi` | HTTP-хендлер с OpenAPI 3.0 JSON |
| `validation` Методы `Validate() error` |
| `env-config` | Методы `LoadFromEnv() error` |
| `log-middleware` | Method-level middleware логирования |
| `metric-middleware` | Method-level middleware метрик |
| `http-metrics-chi` | HTTP-middleware метрик для chi |
| `http-metrics-echo` | HTTP-middleware метрик для echo |
| `http-metrics-fiber` | HTTP-middleware метрик для fiber |
| `http-metrics-mux` | HTTP-middleware метрик для `net/http` |


## HTTP-сервер (`http-server-chi`, `http-server-echo`)

Генерирует регистрацию маршрутов и хендлеры, которые декодируют параметры,
вызывают метод сервиса и пишут ответ.

### Аннотации интерфейса

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки интерфейса |
| `@http-default content-type=<mime>` | Content-Type по умолчанию для методов интерфейса |

### Аннотации метода

| Аннотация | Описание |
|-----------|----------|
| `@http-method <METHOD>` | HTTP-метод: `GET`, `HEAD`, `POST`, `PUT`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`, `PATCH` |
| `@http-path <путь>` | Путь маршрута. Параметры пути задаются как `{id}` или `:id` |
| `@http-single req` / `@http-single-req` | Единственный body-параметр передаётся без обёртки `{"param": value}` |
| `@http-single resp` / `@http-single-resp` | Единственный результат отдаётся без обёртки `{"result": value}` |
| `@http-wrap-req-path <путь>` | Обернуть тело запроса во вложенный JSON-объект. Путь через точку: `data.user` |
| `@http-wrap-resp-path <путь>` | Обернуть тело ответа во вложенный JSON-объект |
| `@http-default content-type=<mime>` | Content-Type для конкретного метода, если запрос пришёл без заголовка |
| `@http-time-format <layout>` | Формат `time.Time` для query/header/cookie/path и form-полей |
| `@http-form-max-memory <байт>` | Лимит multipart-формы для метода (по умолчанию 32 MB) |
| `@http-use url-encoded` | Декодировать тело как `application/x-www-form-urlencoded` |
| `@http-use multipart` | Декодировать тело как `multipart/form-data` |
| `@http-client-skip` | Не генерировать метод в HTTP-клиенте (на сервер не влияет) |

Позиционная форма `@http-wrap-req <путь>` / `@http-wrap-resp <путь>` не
заполняет путь. Используйте `@http-wrap-req-path` / `@http-wrap-resp-path`.

### Аннотации параметров

| Аннотация | Описание |
|-----------|----------|
| `@http-type body` | Значение берётся из тела запроса (по умолчанию) |
| `@http-type query` | Query-параметр |
| `@http-type path` | Параметр пути |
| `@http-type header` | HTTP-заголовок |
| `@http-type cookie` | Cookie |
| `@http-name <имя>` | Имя параметра в запросе |
| `@http-name format=<формат>` | Формат преобразования имени поля в имя параметра |
| `@http-required` | Параметр обязателен |

Если имя параметра совпадает с именованной частью пути (`{id}` или `:id`) в
`lowerCamel`, тип автоматически становится `path`, а параметр — обязательным.

### Аннотации результатов

| Аннотация | Описание |
|-----------|----------|
| `@http-name <имя>` | JSON-ключ результата в теле ответа |
| `@http-flat` | Развернуть поля структуры-результата в корень JSON |

Последний результат метода всегда `error`. Остальные результаты попадают в тело
ответа; `@http-flat` убирает обёртку по имени результата.

### Пример

```go
// @gomosaic
type UserService interface {
	// @http-method GET
	// @http-path /users
	ListUsers(ctx context.Context) (users []User, err error)

	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context, id int) (user *User, err error)

	// @http-method POST
	// @http-path /user
	// @http-wrap-req-path data
	// @http-single resp
	CreateUser(ctx context.Context,
		// @http-type body
		// @http-name user_req
		req *CreateUserRequest,
	) (
		user *User,
		err error,
	)

	// @http-method GET
	// @http-path /search
	SearchUsers(ctx context.Context,
		// @http-type query
		query string,
		// @http-type query
		// @http-name limit_val
		limit int,
		// @http-type header
		// @http-name X-Offset
		offset int,
	) (users []User, total int, err error)

	// @http-method DELETE
	// @http-path /user/{id}
	DeleteUser(ctx context.Context,
		// @http-type path
		id int,
	) (err error)
}
```

Генерация:

```bash
go tool gomosaic codegen http-server-chi ./internal/controller/... ./internal/controller/gen
go tool gomosaic codegen http-server-echo ./internal/controller/... ./internal/controller/gen
```

Сгенерированный API для chi (для echo — `router *echo.Echo`):

```go
func UserServiceRegisterHandlers(
	router *chi.Mux,
	svc UserService,
	opt *UserServiceOptions,
)
```

## HTTP-клиент (`http-client`)

Генерирует клиент с конструктором `New<Interface>Client(target string, opts ...ClientOption)`,
методами интерфейса, builder-методами `<Method>Request(...)` и константами имён методов.

### Аннотации интерфейса

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки интерфейса |
| `@http-client-enable` | Включить генерацию клиента и тестов клиента для интерфейса |
| `@http-copy-types` | Скопировать DTO из исходного пакета в пакет клиента |
| `@http-default accept=<mime>` | Значение заголовка `Accept` (по умолчанию `application/json`) |
| `@http-default content-type=<mime>` | Content-Type JSON-тела (по умолчанию `application/json`) |

Без `@http-client-enable` интерфейс не попадает ни в `http-client`, ни в
`http-client-test`. Аннотации `@http-default` наследуются от интерфейса и могут
быть переопределены на методе.

### Аннотации метода

Помимо общих `@http-method`, `@http-path`, `@http-single` и `@http-wrap-*-path`,
клиент поддерживает:

| Аннотация | Описание |
|-----------|----------|
| `@http-client-skip` | Исключить метод из клиента |
| `@http-query-value <имя> [значения...]` | Добавить фиксированный query-параметр |
| `@http-use url-encoded` | Отправлять примитивные body-параметры как `application/x-www-form-urlencoded` |
| `@http-use multipart` | Отправлять примитивные body-параметры как `multipart/form-data` |
| `@http-time-format <layout>` | Формат `time.Time` для query/header/cookie/path и form-полей |
| `@http-default accept=<mime>` | Переопределить `Accept` для метода |
| `@http-default content-type=<mime>` | Переопределить Content-Type JSON-тела для метода |
| `@http-wrap-req-path <путь>` | Обернуть тело запроса во вложенный JSON-объект |
| `@http-wrap-resp-path <путь>` | Обернуть тело ответа во вложенный JSON-объект |

`@http-use` работает, когда body-параметр примитивный (не структура). По умолчанию
тело отправляется как JSON.

### Аннотации параметров и результатов

Используются те же аннотации, что и для сервера: `@http-type`, `@http-name`,
`@http-required`, `@http-flat`. На клиенте `@http-required` влияет только на
`omitempty` у body-полей и не добавляет runtime-проверок. `@http-flat` на
результате-структуре разворачивает её поля в корень JSON.

### Пример

```go
// @gomosaic
// @http-client-enable
// @http-copy-types
type UserService interface {
	// @http-method GET
	// @http-path /user/{id}
	// @http-single resp
	GetUser(ctx context.Context,
		// @http-type path
		id int,
	) (user *User, err error)

	// @http-method GET
	// @http-path /feed
	// @http-query-value perpage 10
	// @http-query-value status active pending
	GetFeed(ctx context.Context) (users []User, err error)

	// @http-method POST
	// @http-path /form
	// @http-use url-encoded
	SubmitForm(ctx context.Context,
		// @http-type body
		name string,
		// @http-type body
		age int,
	) (err error)

	// @http-method POST
	// @http-path /internal/notify
	// @http-client-skip
	InternalNotify(ctx context.Context,
		// @http-type body
		payload *CreateUserRequest,
	) (err error)
}
```

Генерация и использование:

```bash
go tool gomosaic codegen http-client ./internal/controller/... ./internal/controller/gen
```

```go
client := controller.NewUserServiceClient("http://localhost:8080")
user, err := client.GetUser(ctx, 42)

// Builder-форма позволяет задать опции конкретного запроса.
user, err = client.GetUserRequest(42).
	Execute(controller.WithContext(ctx))
```

## Тесты HTTP-клиента (`http-client-test`)

Генерирует `client_gen_test.go`: тесты методов клиента с использованием
`github.com/jaswdr/faker/v2`. Для каждого метода проверяются успешный ответ
(HTTP 200) и ошибочный сценарий (HTTP 400).

### Аннотации

Отдельных аннотаций нет. Используются `@gomosaic` и HTTP-аннотации интерфейса,
методов, параметров и результатов. Методы с `@http-client-skip` в тесты не попадают.

```bash
go tool gomosaic codegen http-client-test ./internal/controller/... ./internal/controller/gen
```

## HTTP Binder (`http-bind-std`, `http-bind-chi`)

Генерирует метод `Bind(r *http.Request) error`, который извлекает параметры
запроса в поля структуры.

### Аннотации структуры

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки структуры |
| `@http-form-max-memory <байт>` | Максимальный размер multipart-формы (по умолчанию 32 MB) |

Лимит формы для Binder задаётся на структуре. На уровне метода
`@http-form-max-memory` относится к декодированию form-тела HTTP-сервером.

### Аннотации полей

| Аннотация | Описание | По умолчанию |
|-----------|----------|--------------|
| `@http-source query\|path\|header\|cookie\|form\|file` | Источник параметра | `query` |
| `@http-name <имя>` | Имя HTTP-параметра | Имя Go-поля |
| `@http-default <значение>` | Значение по умолчанию | — |
| `@http-required` | Параметр обязателен | — |

### Источники параметров

| Источник | Генерируемый код | Примечание |
|----------|------------------|------------|
| `query` | `r.URL.Query().Get("name")` | Query-строка URL |
| `path` | `r.PathValue("name")` / `chi.URLParam(r, "name")` | Зависит от стратегии |
| `header` | `r.Header.Get("name")` | Заголовки запроса |
| `cookie` | `r.Cookie("name")` | Cookie |
| `form` | `r.FormValue("name")` | Поля формы, автоматически вызывает `ParseMultipartForm` |
| `file` | `r.FormFile("name")` | Файл, тип поля `*multipart.FileHeader` |

### Пример

```go
// @gomosaic
type PostsRequest struct {
	// @http-default all
	Tag string

	// @http-name limit_val
	Limit int

	Offset int

	// @http-source header
	// @http-name X-Request-ID
	RequestID string

	// @http-source path
	// @http-name id
	// @http-required
	PostID int

	// @http-source cookie
	// @http-name session_token
	// @http-default guest
	Token string

	// @http-source form
	// @http-name title
	Title string

	// @http-source file
	// @http-name avatar
	Avatar *multipart.FileHeader
}
```

```go
func (h *Handler) handlePosts(w http.ResponseWriter, r *http.Request) {
	var params PostsRequest
	if err := params.Bind(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
```

## gRPC-сервер (`grpc-server`)

Генерирует регистрацию gRPC-сервиса, адаптер с middleware и, при указании
`@proto-package`, функции конвертации между protobuf- и Go-типами.

### Аннотации интерфейса

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки интерфейса |
| `@proto-package <import path>` | Пакет с protobuf-типами для конвертации |

### Аннотации метода

| Аннотация | Описание |
|-----------|----------|
| `@grpc-method` | Унарный метод (по умолчанию) |
| `@grpc-method server-stream` | Серверный стриминг |
| `@grpc-method client-stream` | Клиентский стриминг |
| `@grpc-method bidi-stream` | Двунаправленный стриминг |

### Пример

```go
// @gomosaic
// @proto-package github.com/example/project/api/proto
type UserService interface {
	// @grpc-method
	GetUser(ctx context.Context, req *GetUserRequest) (resp *User, err error)

	// @grpc-method server-stream
	ListUsers(ctx context.Context, req *ListUsersRequest, stream UserService_ListUsersServer) error
}
```

Сгенерированный API:

```go
func RegisterUserServiceServer(
	s *grpc.Server,
	svc UserService,
	mw ...runtime.Middleware[UserService],
)
```

```bash
go tool gomosaic codegen grpc-server ./internal/service/... ./internal/transport/grpc
```

## OpenAPI (`openapi`)

Генерирует `openapi_gen.go` с функцией `OpenAPIHandler() http.Handler`, которая
отдаёт OpenAPI 3.0 JSON. Пути и параметры строятся по HTTP-аннотациям.

### Аннотации интерфейса

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки интерфейса |
| `@openapi-tags <тег> [тег...]` | Теги эндпоинтов. Если не заданы, используется имя интерфейса |

### Аннотации метода

Используются `@http-method` и `@http-path`. В спецификацию попадают path- и
query-параметры, размеченные через `@http-type path` и `@http-type query`.
`summary` и `operationId` формируются автоматически из имени метода.

### Пример

```go
// @gomosaic
// @openapi-tags users
type UserService interface {
	// @http-method GET
	// @http-path /users
	ListUsers(ctx context.Context,
		// @http-type query
		// @http-name limit
		limit int,
	) (users []User, err error)

	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context,
		// @http-type path
		id int,
	) (user *User, err error)
}
```

```bash
go tool gomosaic codegen openapi ./internal/controller/... ./internal/controller/gen
```

```go
http.Handle("/openapi.json", controller.OpenAPIHandler())
```

## Валидация (`validation`)

Генерирует метод `Validate() error` для структур с аннотациями `@validate`.

### Аннотации

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки структуры |
| `@validate required` | Поле обязательно: строка не пустая, число не ноль, указатель/слайс/мапа не `nil` |
| `@validate min=<N>` | Минимальное значение для числовых полей |
| `@validate max=<N>` | Максимальное значение для числовых полей |
| `@validate min-len=<N>` | Минимальная длина строки |
| `@validate max-len=<N>` | Максимальная длина строки |
| `@validate pattern=<regexp>` | Соответствие регулярному выражению |
| `@validate email` | Проверка email |
| `@validate url` | Проверка URL (`http://` или `https://`) |

Правила можно комбинировать в одной аннотации через пробел.

### Пример

```go
// @gomosaic
type User struct {
	// @validate required min-len=2 max-len=50
	Name string

	// @validate required email
	Email string

	// @validate required min=18 max=150
	Age int

	// @validate url
	Website string

	// @validate pattern=^[a-zA-Z0-9_]+$
	Username string
}
```

```go
user := &User{ /* ... */ }
if err := user.Validate(); err != nil {
	// ...
}
```

```bash
go tool gomosaic codegen validation ./internal/model/... ./internal/model/gen
```

## Конфигурация из окружения (`env-config`)

Генерирует метод `LoadFromEnv() error` для структур конфигурации.

### Аннотации

| Аннотация | Описание |
|-----------|----------|
| `@gomosaic` | Обязательный маркер обработки структуры |
| `@env-name <VAR>` | Имя переменной окружения. По умолчанию — `SCREAMING_SNAKE_CASE` имени поля |
| `@env-default <значение>` | Значение по умолчанию, если переменная не задана |
| `@env-validate required` | Ошибка, если переменная не задана и нет значения по умолчанию |
| `@env-validate max-len=<N>` | Максимальная длина строкового значения |
| `@env-validate min-len=<N>` | Минимальная длина строкового значения |

Поддерживаются вложенные структуры: для поля-структуры генерируется отдельный
`LoadFromEnv`, а ошибки оборачиваются именем поля.

### Пример

```go
// @gomosaic
type Config struct {
	// @env-name HOST
	// @env-validate required max-len=100
	Host string

	// @env-name PORT
	// @env-default 8080
	Port int

	// @env-name DEBUG
	Debug bool

	// @env-name TIMEOUT
	// @env-default 30s
	Timeout time.Duration

	Database DatabaseConfig
}

type DatabaseConfig struct {
	// @env-name DB_ADDR
	// @env-validate required
	Addr string

	// @env-name DB_PORT
	// @env-default 5432
	Port int
}
```

```go
cfg := &Config{}
if err := cfg.LoadFromEnv(); err != nil {
	// ...
}
```

```bash
go tool gomosaic codegen env-config ./internal/config/... ./internal/config/gen
```

## Method-level middleware (`log-middleware`, `metric-middleware`)

Генерируют middleware, реализующий интерфейс сервиса и оборачивающий каждый метод.
Обрабатываются все интерфейсы с `@gomosaic`; для отдельных методов можно указать
`@log-skip` или `@metric-skip`.

| Плагин | Что генерирует |
|--------|----------------|
| `log-middleware` `<Service>LogMiddleware` и `Log<Service>Middleware(logger span.Logger) runtime.Middleware[<Service>]` |
| `metric-middleware` | `<Service>MetricMiddleware` и `Metric<Service>Middleware(collector span.MetricsCollector) runtime.Middleware[<Service>]` |

Пример для интерфейса `UserService`:

```go
logMW := controller.LogUserServiceMiddleware(logger)
metricMW := controller.MetricUserServiceMiddleware(collector)
```

```bash
go tool gomosaic codegen log-middleware ./internal/service/... ./internal/middleware
go tool gomosaic codegen metric-middleware ./internal/service/... ./internal/middleware
```

Для отдельных методов можно отключить обёртку, не убирая метод из middleware:

| Аннотация | Плагин | Описание |
|-----------|--------|----------|
| `@log-skip` | `log-middleware` | Не создавать log-span, метод просто делегируется в `next` |
| `@metric-skip` | `metric-middleware` | Не создавать metric-span, метод просто делегируется в `next` |

## HTTP-middleware метрик (`http-metrics-*`)

Генерируют middleware уровня роутера для Prometheus. В отличие от
`metric-middleware`, такая middleware видит все HTTP-запросы, включая отклонённые
до вызова сервиса. Плагины не используют аннотации.

| Плагин | Выходной файл | Конструктор и тип |
|--------|---------------|-------------------|
| `http-metrics-chi` | `metrics_chi_gen.go` | `NewChiMetricsMiddleware(reg, namespace, constLabels) func(http.Handler) http.Handler` |
| `http-metrics-mux` | `metrics_mux_gen.go` | `NewMuxMetricsMiddleware(reg, namespace, constLabels) func(http.Handler) http.Handler` |
| `http-metrics-echo` | `metrics_echo_gen.go` | `NewEchoMetricsMiddleware(reg, namespace, constLabels) echo.MiddlewareFunc` |
| `http-metrics-fiber` | `metrics_fiber_gen.go` | `NewFiberMetricsMiddleware(reg, namespace, constLabels) func(*fiber.Ctx) error` |

Собираются метрики `http_requests_total`, гистограмма длительности и счётчик
ошибок. Метки: метод, путь (шаблон маршрута), HTTP-статус.

```go
mux.Use(controller.NewChiMetricsMiddleware(reg, "myapp", prometheus.Labels{}))
```

```bash
go tool gomosaic codegen http-metrics-chi ./... ./internal/metrics
```

## Программное использование

Вместо CLI можно собрать генератор в своём `main.go`:

```go
package main

import (
	"github.com/go-mosaic/gomosaic/v2/pkg/cmd"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic/defaults"
)

func main() {
	builder := gomosaic.NewBuilder(defaults.WithPlugins())
	cmd.Run("1.0.0", builder)
}
```

Регистрация только нужных плагинов:

```go
builder := gomosaic.NewBuilder()
builder.WithPlugins(
	server.NewPlugin(&server.ChiStrategy{}),
	client.NewPlugin(),
	validation.NewPlugin(),
	envconfig.NewPlugin(),
)
```

Генерация без CLI:

```go
gen := builder.Build(gomosaic.NewFileSystem("1.0.0", "./gen"))

ctx := gomosaic.ContextWithOutputDir(context.Background(), "./gen")
files, err := gen.Generate(ctx, moduleInfo, typesInfo, "http-server-chi")
```
