package service

import "context"

type User struct {
	Name  string
	Email string
}

type CreateUserRequest struct {
	Name  string
	Email string
}

type UserService interface {
	// @http-method GET
	// @http-path /users
	ListUsers(ctx context.Context) (users []User, err error)

	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context, id int) (user *User, err error)

	// @http-method POST
	// @http-path /user
	CreateUser(ctx context.Context, req *CreateUserRequest) (user *User, err error)

	// @http-method DELETE
	// @http-path /user/{id}
	DeleteUser(ctx context.Context, id int) (err error)
}
