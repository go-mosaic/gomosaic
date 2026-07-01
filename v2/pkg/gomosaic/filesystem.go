package gomosaic

import (
	"io"
	"os"
	"path/filepath"
)

// File представляет файл, который может быть отрендерен.
type File interface {
	Render(w io.Writer, version string) error
}

// FileSystem отвечает за запись сгенерированных файлов.
type FileSystem struct {
	version   string
	outputDir string
}

func NewFileSystem(version, outputDir string) *FileSystem {
	return &FileSystem{
		version:   version,
		outputDir: outputDir,
	}
}

func (fs *FileSystem) SaveFile(filename string, file File) (path string, err error) {
	path = filepath.Join(fs.outputDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer f.Close()

	if err := file.Render(f, fs.version); err != nil {
		return "", err
	}

	return path, nil
}
