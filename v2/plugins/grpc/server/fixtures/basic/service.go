package basic

import "context"

// @gomosaic
// @proto-package github.com/go-mosaic/gomosaic/v2/plugins/grpc/server/fixtures/pb
type UserService interface {
	// @grpc-method
	GetUser(ctx context.Context, req *GetUserRequest) (resp *User, err error)

	// @grpc-method server-stream
	ListUsers(ctx context.Context, req *ListUsersRequest, stream UserService_ListUsersServer) error
}

type GetUserRequest struct {
	Id int //nolint:staticcheck // имя поля совпадает с protobuf-генерацией
}

type User struct {
	Name  string
	Email string
}

type ListUsersRequest struct {
	PageSize int32
	Filter   string
}

// UserService_ListUsersServer: заглушка gRPC серверного стрима.
//
//nolint:staticcheck // имя соответствует protobuf-генерации gRPC-стрима
type UserService_ListUsersServer interface {
	Send(*User) error
}
