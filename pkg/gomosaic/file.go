package gomosaic

import "io"

type File interface {
	Render(w io.Writer, version string) error
}
