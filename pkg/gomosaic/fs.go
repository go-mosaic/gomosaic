package gomosaic

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileSystem отвечает за запись сгенерированных файлов
type FileSystem struct {
	version   string
	outputDir string
}

// NewFileSystem создает новый экземпляр FileSystem
func NewFileSystem(version, outputDir string) *FileSystem {
	return &FileSystem{
		version:   version,
		outputDir: outputDir,
	}
}

// SaveFile сохраняет AST в файл
func (fs *FileSystem) SaveFile(filename string, file File) (path string, err error) {
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
