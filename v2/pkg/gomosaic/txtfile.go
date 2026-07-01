package gomosaic

import (
	"bytes"
	"fmt"
	"io"
)

var _ File = &TxtFile{}

type TxtFile struct {
	buf bytes.Buffer
}

func (f *TxtFile) Line() {
	f.WriteText("\n")
}

func (f *TxtFile) WriteBytes(p []byte) {
	_, _ = f.buf.Write(p)
}

func (f *TxtFile) Write(p []byte) (n int, err error) {
	return f.buf.Write(p)
}

func (f *TxtFile) WriteText(format string, a ...any) {
	_, _ = fmt.Fprintf(&f.buf, format, a...)
}

func (f *TxtFile) Render(w io.Writer, version string) error {
	return nil
}

func NewTxtFile() *TxtFile {
	return &TxtFile{}
}
