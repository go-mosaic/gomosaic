package logservice

import "context"

// @gomosaic
type UserService interface {
	GetUser(ctx context.Context, id int) (user string, err error)
	CreateUser(ctx context.Context, name string) (err error)
}
