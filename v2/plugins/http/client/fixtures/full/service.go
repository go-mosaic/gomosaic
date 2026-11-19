package service

import (
	"context"
	"time"
)

type User struct {
	Name  string
	Email string
}

type CreateUserRequest struct {
	Name  string
	Email string
	Age   int
}

type Filter struct {
	Title       string   `json:"title"`
	Statuses    []string `json:"statuses"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
	TargetUsers []string `json:"targetUsers"`
	PerPage     int      `json:"perPage"`
	Page        int      `json:"page"`
}

// UserService интерфейс сервиса пользователей.
// @gomosaic
// @http-default accept=application/json
// @http-default content-type=application/json
// @http-client-enable
// @http-copy-types
type UserService interface {
	// @http-method GET
	// @http-path /users
	// @http-default accept=application/vnd.api+json
	// @http-default content-type=application/vnd.api+json
	ListUsers(ctx context.Context) (users []User, err error)

	// @http-method GET
	// @http-path /user/{id}
	// @http-single resp
	GetUser(ctx context.Context,
		// @http-type path
		// @http-required
		id int,
	) (user *User, err error)

	// @http-method POST
	// @http-path /user
	// @http-wrap-req-path data
	// @http-single req
	// @http-default content-type=application/vnd.api+json
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
		// @http-type query
		since time.Time,
		// @http-type header
		// @http-name X-Offset
		offset int,
	) (users []User, total int, err error)

	// @http-method GET
	// @http-path /profile
	GetProfile(ctx context.Context) (
		// @http-flat
		user *User,
		total int,
		err error,
	)

	// @http-method POST
	// @http-path /internal/notify
	// @http-client-skip
	InternalNotify(ctx context.Context,
		// @http-type body
		payload *CreateUserRequest,
	) (err error)

	// @http-method GET
	// @http-path /tasks
	ListTasks(ctx context.Context,
		// @http-type query
		filter Filter,
	) (users []User, err error)

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
	// @http-path /upload
	// @http-use multipart
	Upload(ctx context.Context,
		// @http-type body
		filename string,
	) (err error)

	// @http-method POST
	// @http-path /note
	// @http-wrap-req-path data
	CreateNote(ctx context.Context,
		// @http-type body
		title string,
		// @http-type body
		body string,
	) (err error)

	// @http-method GET
	// @http-path /raw
	GetRaw(ctx context.Context) (data []byte, err error)
}
