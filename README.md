# GoMosaic утилита для генерации кода на основе GoLang интерфейсов.

[![Lint Status](https://github.com/go-mosaic/gomosaic/workflows/golangci-lint/badge.svg)](https://github.com/go-mosaic/gomosaic/actions)

Утилита автоматически генерирует код для транспортного слоя HTTP-сервера и клиента на основе заданных GoLang интерфейсов. Она также поддерживает добавление middleware и загрузку конфигураций из файлов.

Пример использования:

### 1. Определение интерфейса:

```go
// @gomosaic
type UserService interface {
    // @http-method GET
    // @http-path /user
    GetUser(ctx context.Context, id int) (user *User, err error)
    // @http-method POST
    // @http-path /user
    CreateUser(ctx context.Context, user *User) (err error)
}
```

### 2. Генерация кода:

```bash
gomosaic http-server-chi ./internal/usecase/controller/... ./controller
```

### 3. Результат:

В папке ./controller будут создан файл:

- server.go — HTTP-сервер Chi с роутами для методов GetUser и CreateUser.

Установка:

```bash
go install github.com/go-mosaic/gomosaic/cmd/gomosaic@latest
```

Преимущества:

- Экономия времени: автоматизация рутинных задач.
- Типизация: минимизация ошибок благодаря типизированным запросам и ответам.
- Гибкость: возможность кастомизации и конфигураций.

## HTTP Binder — хелперы сборки параметров HTTP-запроса

Плагин генерирует метод `Bind(r *http.Request) error` для  извлечения параметров в структуру.

### Аннотации структуры

| Аннотация | Описание |
|-----------|----------|
| `@http-form-max-memory <байт>` | Максимальный размер multipart-формы (по умолчанию 32 MB) |

### Аннотации полей

| Аннотация | Описание | Значение по умолчанию |
|-----------|----------|----------------------|
| `@http-source query\|path\|header\|cookie\|form\|file` | Источник параметра | `query` |
| `@http-name <имя>` | Имя HTTP-параметра (если отличается от имени поля) | Имя Go-поля |
| `@http-default <значение>` | Значение по умолчанию | — |
| `@http-required` | Параметр обязателен | — |

### Источники параметров

| Источник | Генерируемый код | Примечание |
|----------|------------------|------------|
| `query` | `r.URL.Query().Get("name")` | Query-строка URL |
| `path` | `r.PathValue("name")` / `chi.URLParam(r, "name")` | Зависит от стратегии |
| `header` | `r.Header.Get("name")` | Заголовки запроса |
| `cookie` | `r.Cookie("name")` | Куки |
| `form` | `r.FormValue("name")` | Поля формы (автоматически вызывает `ParseMultipartForm`) |
| `file` | `r.FormFile("name")` | Загружаемые файлы (тип поля: `*multipart.FileHeader`) |

### Стратегии

Плагин поддерживает стратегии для разных HTTP-роутеров:

```go
// Стандартный роутер 
stdPlugin := binder.NewPlugin(&binder.StdStrategy{})

// Chi-роутер
chiPlugin := binder.NewPlugin(&binder.ChiStrategy{})
```

### Пример

**Описание структуры:**

```go
// @gomosaic
type PostsRequest struct {
    // @http-default all
    Tag string

    // @http-name limit_val
    Limit int

    // @http-source header
    // @http-name X-Request-ID
    RequestID string

    // @http-source path
    // @http-name id
    // @http-required
    PostID int

    // @http-source form
    // @http-name title
    Title string

    // @http-source file
    // @http-name avatar
    Avatar *multipart.FileHeader
}
```

**Использование в хендлере:**

```go
func (h *Handler) handlePosts(w http.ResponseWriter, r *http.Request) {
    var params PostsRequest
    if err := params.Bind(r); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
}
```
