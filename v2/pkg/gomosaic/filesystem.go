package gomosaic

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type File interface {
	Render(w io.Writer, version string) error
}

type FileSystem interface {
	SaveFile(filename string, file File) (path string, err error)
}

type DiskFileSystem struct {
	version   string
	outputDir string
}

func NewFileSystem(version, outputDir string) *DiskFileSystem {
	return &DiskFileSystem{
		version:   version,
		outputDir: outputDir,
	}
}

func (fs *DiskFileSystem) SaveFile(filename string, file File) (path string, err error) {
	path = filepath.Join(fs.outputDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("не удалось создать файл: %w", err)
	}

	defer f.Close()

	if err := file.Render(f, fs.version); err != nil {
		return "", fmt.Errorf("не удалось записать файл: %w", err)
	}

	return path, nil
}

type MemoryFileSystem struct {
	version string
	files   map[string][]byte
}

func NewMemoryFileSystem(version string) *MemoryFileSystem {
	return &MemoryFileSystem{
		version: version,
		files:   make(map[string][]byte),
	}
}

func (fs *MemoryFileSystem) SaveFile(filename string, file File) (path string, err error) {
	var buf bytes.Buffer

	if err := file.Render(&buf, fs.version); err != nil {
		return "", fmt.Errorf("не удалось записать файл: %w", err)
	}

	fs.files[filename] = buf.Bytes()

	return filename, nil
}

func (fs *MemoryFileSystem) File(filename string) ([]byte, bool) {
	data, ok := fs.files[filename]
	return data, ok
}

func (fs *MemoryFileSystem) Files() map[string][]byte {
	return fs.files
}

var (
	_ FileSystem = (*DiskFileSystem)(nil)
	_ FileSystem = (*MemoryFileSystem)(nil)
)
