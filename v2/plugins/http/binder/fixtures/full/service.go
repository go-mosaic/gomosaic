package service

import "mime/multipart"

// PostsRequest параметры запроса для получения постов.
// @gomosaic
type PostsRequest struct {
	// @http-default all
	Tag string

	// @http-name limit_val
	Limit int

	Offset int

	// @http-source header
	// @http-name X-Request-ID
	RequestID string

	// @http-source path
	// @http-name id
	// @http-required
	PostID int

	// @http-source cookie
	// @http-name session_token
	// @http-default guest
	Token string

	// @http-source form
	// @http-name title
	Title string

	// @http-source file
	// @http-name avatar
	Avatar *multipart.FileHeader
}
