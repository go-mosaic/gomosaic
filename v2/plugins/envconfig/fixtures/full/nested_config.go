package config

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

	Database DatabaseConfig
}

type DatabaseConfig struct {
	// @env-name DB_ADDR
	// @env-validate required
	Addr string

	// @env-name DB_PORT
	// @env-default 5432
	Port int
}
