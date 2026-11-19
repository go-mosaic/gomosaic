package gomosaic

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// File представляет сгенерированный файл.
type File interface {
	Render(w io.Writer, version string) error
}

// FileSystem сохраняет сгенерированные файлы.
type FileSystem interface {
	SaveFile(filename string, file File) (path string, err error)
}

// DiskFileSystem пишет файлы на диск в outputDir.
type DiskFileSystem struct {
	version   string
	outputDir string
}

// NewFileSystem создаёт файловую систему, пишущую файлы в outputDir.
func NewFileSystem(version, outputDir string) *DiskFileSystem {
	return &DiskFileSystem{
		version:   version,
		outputDir: outputDir,
	}
}

// SaveFile реализует FileSystem.
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

// MemoryFileSystem хранит файлы в памяти. Удобна в тестах.
type MemoryFileSystem struct {
	version string
	files   map[string][]byte
}

// NewMemoryFileSystem создаёт файловую систему в памяти.
func NewMemoryFileSystem(version string) *MemoryFileSystem {
	return &MemoryFileSystem{
		version: version,
		files:   make(map[string][]byte),
	}
}

// SaveFile реализует FileSystem.
func (fs *MemoryFileSystem) SaveFile(filename string, file File) (path string, err error) {
	var buf bytes.Buffer

	if err := file.Render(&buf, fs.version); err != nil {
		return "", fmt.Errorf("не удалось записать файл: %w", err)
	}

	fs.files[filename] = buf.Bytes()

	return filename, nil
}

// File возвращает содержимое файла и признак наличия.
func (fs *MemoryFileSystem) File(filename string) ([]byte, bool) {
	data, ok := fs.files[filename]
	return data, ok
}

// Files возвращает все сохранённые файлы.
func (fs *MemoryFileSystem) Files() map[string][]byte {
	return fs.files
}

var (
	_ FileSystem = (*DiskFileSystem)(nil)
	_ FileSystem = (*MemoryFileSystem)(nil)
)
