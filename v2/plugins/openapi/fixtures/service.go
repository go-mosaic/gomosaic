package service

import "context"

// @gomosaic
// @openapi-tags users
type UserService interface {
	// @http-method GET
	// @http-path /user/{id}
	GetUser(ctx context.Context, id int) (user *User, err error)
}

type User struct {
	Name  string
	Email string
}
