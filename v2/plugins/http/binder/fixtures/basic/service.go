package service

// ListPostsParams параметры запроса для списка постов.
// @gomosaic
type ListPostsParams struct {
	// @http-default all
	Tag string

	// @http-name limit_val
	Limit int

	Offset int
}
