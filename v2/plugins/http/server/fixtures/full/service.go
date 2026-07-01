package service

import "context"

type User struct {
	Name  string
	Email string
}

type CreateUserRequest struct {
	Name  string
	Email string
	Age   int
}

type SearchParams struct {
	Query  string
	Limit  int
	Offset int
}

// @http-default accept=application/json
// @http-default content-type=application/json
// @http-client-enable
// @openapi-tags users
type UserService interface {
	// @http-method GET
	// @http-path /users
	// @openapi-summary Список пользователей
	ListUsers(ctx context.Context) (users []User, err error)

	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context,
		// @http-type path
		// @http-required
		id int,
	) (user *User, err error)

	// @http-method POST
	// @http-path /user
	// @http-wrap-req data
	// @http-single req
	// @http-default content-type=application/json
	CreateUser(ctx context.Context,
		// @http-type body
		// @http-name user_req format=snake
		req *CreateUserRequest,
	) (
		// @http-type header
		// @http-name X-Request-Id
		requestID string,
		user *User,
		err error,
	)

	// @http-method DELETE
	// @http-path /user/{id}
	DeleteUser(ctx context.Context,
		// @http-type path
		id int,
	) (err error)

	// @http-method GET
	// @http-path /search
	// @http-time-format 2006-01-02
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
}
