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

Лицензия: 

MIT