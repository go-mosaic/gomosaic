package metricservice

import "context"

// @gomosaic
type MetricService interface {
	GetUser(ctx context.Context, id int) (user string, err error)
}
