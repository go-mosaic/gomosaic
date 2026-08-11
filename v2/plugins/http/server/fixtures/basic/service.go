package basic

import "context"

// UserService интерфейс сервиса для работы с пользователями
// @gomosaic
// @openapi-tags users
type UserService interface {
	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context, id int) (user *User, err error)
}

// User структура пользователя для теста генерации
type User struct {
	Name  string
	Email string
}
