package metricservice

import "context"

// @gomosaic
type MetricService interface {
	GetUser(ctx context.Context, id int) (user string, err error)

	// @metric-skip
	DeleteUser(ctx context.Context, id int) (err error)
}
