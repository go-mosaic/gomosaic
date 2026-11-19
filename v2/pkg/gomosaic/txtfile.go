package gomosaic

import (
	"bytes"
	"fmt"
	"io"
)

var _ File = &TxtFile{}

// TxtFile собирает текст в буфере и пишет его в io.Writer.
type TxtFile struct {
	buf bytes.Buffer
}

// Line добавляет перевод строки.
func (f *TxtFile) Line() {
	f.WriteTextf("\n")
}

// WriteBytes дописывает байты.
func (f *TxtFile) WriteBytes(p []byte) {
	_, _ = f.buf.Write(p)
}

// Write реализует io.Writer.
func (f *TxtFile) Write(p []byte) (n int, err error) {
	return f.buf.Write(p)
}

// WriteTextf дописывает форматированный текст.
func (f *TxtFile) WriteTextf(format string, a ...any) {
	_, _ = fmt.Fprintf(&f.buf, format, a...)
}

// Render реализует File.
func (f *TxtFile) Render(w io.Writer, version string) error {
	return nil
}

// NewTxtFile создаёт пустой текстовый файл.
func NewTxtFile() *TxtFile {
	return &TxtFile{}
}
