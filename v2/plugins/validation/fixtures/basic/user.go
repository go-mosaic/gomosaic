package basic

// User модель пользователя с валидацией.
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
}
