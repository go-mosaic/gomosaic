package basic

import "time"

// @gomosaic
type Config struct {
	// @env-name HOST
	// @env-validate required max-len=100
	Host string

	// @env-name PORT
	// @env-default 8080
	Port int

	// @env-name DEBUG
	Debug bool

	// @env-name TIMEOUT
	// @env-default 30s
	Timeout time.Duration

	// @env-name RETRY_COUNT
	// @env-default 3
	RetryCount int
}
