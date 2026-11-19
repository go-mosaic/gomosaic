package service

import "context"

// @gomosaic
type Service interface {
	GetUser(ctx context.Context, id string) (user string, err error)
}
