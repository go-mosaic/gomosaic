package service

import "context"

// @metric
type MetricService interface {
	GetUser(ctx context.Context, id int) (user string, err error)
}
