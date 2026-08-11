package model

// User модель с полным набором валидаций
// @gomosaic
type User struct {
	// @validate required min-len=2 max-len=50
	Name string

	// @validate required email
	Email string

	// @validate required min=18 max=150
	Age int

	// @validate url
	Website string

	// @validate pattern=^[a-zA-Z0-9_]+$
	Username string

	// @validate required min-len=8
	Password string
}
