
package service

import "context"

// @grpc-service
type UserService interface {
	// @grpc-method
	GetUser(ctx context.Context, req *GetUserRequest) (resp *User, err error)
}

type GetUserRequest struct {
	Id int
}

type User struct {
	Name  string
	Email string
}
